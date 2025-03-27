package transitive_dependencies

import (
	"context"
	"go.uber.org/zap"
	"scanoss.com/dependencies/pkg/models"
	"sync"
	"time"
)

type DependencyJob struct {
	PurlName  string
	Version   string
	Depth     int
	Ecosystem string
}

type Result struct {
	Parent                 DependencyJob
	TransitiveDependencies []DependencyJob
}

type DependencyCollectorCfg struct {
	MaxWorkers    int
	MaxQueueLimit int
	TimeOut       int
}

type DependencyCollector struct {
	ResultHandler   func(Result) bool
	Config          DependencyCollectorCfg
	jobs            []DependencyJob
	dependencyModel *models.DependencyModel
	mapMutex        sync.RWMutex
	cache           map[string][]models.UnresolvedDependency
	ctx             context.Context
	resultChannel   chan Result
	jobChannel      chan DependencyJob
	pendingJobs     int
	logger          *zap.SugaredLogger
}

func NewDependencyCollector(
	ctx context.Context,
	resultHandler func(result Result) bool,
	config DependencyCollectorCfg,
	model *models.DependencyModel,
	logger *zap.SugaredLogger) *DependencyCollector {
	return &DependencyCollector{
		ctx:             ctx,
		ResultHandler:   resultHandler,
		Config:          config,
		dependencyModel: model,
		mapMutex:        sync.RWMutex{},
		cache:           make(map[string][]models.UnresolvedDependency),
		resultChannel:   make(chan Result, config.MaxQueueLimit),
		jobChannel:      make(chan DependencyJob, config.MaxQueueLimit),
		pendingJobs:     0,
		logger:          logger,
	}
}

func (dc *DependencyCollector) InitJobs(inputJobs []DependencyJob) {
	dc.jobs = inputJobs
	dc.pendingJobs = len(dc.jobs)
}

// Start initiates dependency collection by spawning workers, sending initial jobs, and
// monitoring results until completion or timeout
func (dc *DependencyCollector) Start() {
	// Create context with cancel
	ctx, cancel := context.WithCancel(dc.ctx)
	ctxTimeout, timeoutCancel := context.WithTimeout(ctx, time.Duration(dc.Config.TimeOut)*time.Second)
	// Make sure to defer both cancels (in reverse order)
	defer timeoutCancel()
	defer cancel()
	var wg sync.WaitGroup

	// Start workers
	for i := 1; i <= dc.Config.MaxWorkers; i++ {
		wg.Add(1)
		go dc.worker(i, dc.jobChannel, &wg, dc.resultChannel, ctxTimeout)
	}

	// Send initial jobs
	for _, job := range dc.jobs {
		dc.logger.Infof("Main: Sending job %s with depth %d\n", job.PurlName, job.Depth)
		dc.jobChannel <- job
	}

	// Start the completion monitor
	wg.Add(1)
	go dc.processResult(&wg, ctxTimeout, cancel)
	wg.Wait()

	dc.logger.Info("All workers have exited. Processing completed.")
}

// processResult monitors the result channel, processes dependency results
// using ResultHandler, and manages the job queue. It tracks job completion
// and signals when processing should terminate. Returns on context
// cancellation or when all work is finished.
func (dc *DependencyCollector) processResult(wg *sync.WaitGroup, ctx context.Context, cancel context.CancelFunc) {
	defer wg.Done() // Ensure we signal completion even if we exit early
	for {
		select {
		case <-ctx.Done():
			// Context was cancelled, stop processing results
			dc.logger.Debug("Results processor stopping due to context cancellation")
			return

		case result := <-dc.resultChannel:

			/* ResultHandler processes each dependency result.
			It returns true when processing should stop (e.g., when maximum dependencies limit is reached),
			which signals the collector to cancel further operations. Returns false to continue processing.
			*/
			if dc.ResultHandler(result) {
				dc.logger.Debug("Result handler signaled to stop processing")
				cancel()
				return
			}
			// Queue up new jobs
			for _, job := range result.TransitiveDependencies {
				if job.Depth > 0 {
					select {
					case dc.jobChannel <- job:
						dc.pendingJobs++
						// Job was added successfully
					case <-ctx.Done():
						return
					default:
						dc.logger.Debug("Skipping dependency due to max queue limit reached")
					}
				}
			}
			// Decrement counter after adding all new jobs
			dc.pendingJobs--
			// Check if we're done with all jobs
			if dc.pendingJobs == 0 {
				dc.logger.Debug("No more pending jobs, signaling completion")
				cancel()
				return
			}
		}
	}
}

// worker processes dependency jobs, retrieves and caches transitive dependencies,
// sanitizes version requirements, and sends results to the results channel.
// Terminates on context cancellation.
func (dc *DependencyCollector) worker(id int, jobs chan DependencyJob, wg *sync.WaitGroup, results chan Result, ctx context.Context) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			// Context was cancelled, stop the worker
			dc.logger.Debugf("Worker %d stopping due to context cancellation\n", id)
			return

		case job := <-jobs:

			cacheKey := job.PurlName + "@" + job.Version
			// First try with a read lock only: TODO: Check if we need a RLock
			dc.mapMutex.RLock()
			transitiveDependencies, exists := dc.cache[cacheKey]
			dc.mapMutex.RUnlock()

			if !exists {
				transitiveDependencies, _ = dc.dependencyModel.GetDependencies(job.PurlName, job.Version, job.Ecosystem)
				if len(transitiveDependencies) > 0 {
					dc.mapMutex.Lock()
					dc.cache[cacheKey] = transitiveDependencies
					dc.mapMutex.Unlock()
				}
			}

			// Generate new jobs with depth-1
			newJobDepth := job.Depth - 1

			// sanitize versions
			var transitiveDependenciesJobs []DependencyJob
			var sanitizedDependencies []models.UnresolvedDependency
			for _, ud := range transitiveDependencies {
				fixedVersion, err := PickFirstVersionFromRange(ud.Requirement)
				if err != nil {
					dc.logger.Debugf("Cannot resolve requirement %s\n", ud.Requirement)
					continue
				}
				sanitizedDependencies = append(sanitizedDependencies, models.UnresolvedDependency{
					Purl:        ud.Purl,
					Requirement: fixedVersion,
				})
				transitiveDependenciesJobs = append(transitiveDependenciesJobs, DependencyJob{PurlName: ud.Purl, Version: fixedVersion, Ecosystem: job.Ecosystem, Depth: newJobDepth})
			}

			// Send result, but also handle context cancellation
			select {
			case results <- Result{
				Parent:                 job,
				TransitiveDependencies: transitiveDependenciesJobs,
			}:
				dc.logger.Debugf("Worker %d: Completed job %s at depth %d\n", id, job.PurlName, newJobDepth)
			case <-ctx.Done():
				dc.logger.Debugf("Worker %d: Context cancelled while sending results\n", id)
				return
			}
			dc.logger.Infof("Worker %d: Completed job %s at depth %d\n", id, job.PurlName, newJobDepth)
		}
	}
}
