package transitive_dependencies

import (
	"context"
	"fmt"
	"scanoss.com/dependencies/pkg/models"
	"sync"
	"time"
)

type Job struct {
	Purl      string
	Version   string
	Depth     int
	Ecosystem string
}

type Result struct {
	Parent string
	Purls  []string
}

type DependencyCollectorCfg struct {
	MaxWorkers    int
	MaxQueueLimit int
}

type DependencyCollector struct {
	Callback        func(Result)
	Config          DependencyCollectorCfg
	jobs            []Job
	dependencyModel *models.DependencyModel
	mapMutex        sync.RWMutex
	cache           map[string][]models.UnresolvedDependency
	ctx             context.Context
	resultChannel   chan Result
}

type Component struct {
	PackageName string
	Version     string
}

type TransitiveDependencyInput struct {
	Components []Component `json:"components"`
	Depth      int         `json:"depth"`
	Ecosystem  string      `json:"ecosystem"`
}

func NewDependencyCollector(ctx context.Context, c func(result Result), config DependencyCollectorCfg, model *models.DependencyModel) *DependencyCollector {
	return &DependencyCollector{
		ctx:             ctx,
		Callback:        c,
		Config:          config,
		dependencyModel: model,
		mapMutex:        sync.RWMutex{},
		cache:           make(map[string][]models.UnresolvedDependency),
		resultChannel:   make(chan Result, config.MaxQueueLimit),
	}
}

func (dc *DependencyCollector) SetResultCallback(c func(Result)) {
	dc.Callback = c
}

func (dc *DependencyCollector) InitJobs(metadata TransitiveDependencyInput) {
	dc.jobs = make([]Job, len(metadata.Components))
	for i, component := range metadata.Components {
		dc.jobs[i] = Job{
			Purl:      component.PackageName,
			Version:   component.Version,
			Depth:     metadata.Depth,
			Ecosystem: metadata.Ecosystem,
		}
	}
}

func (dc *DependencyCollector) Start() {
	// Create a buffered job channel
	jobsChannel := make(chan Job, dc.Config.MaxQueueLimit)
	// First create a context with cancel
	ctx, cancel := context.WithCancel(dc.ctx)

	// Then create a timeout context derived from the cancel context
	ctxTimeout, timeoutCancel := context.WithTimeout(ctx, 10*time.Minute) // 5-minute timeout
	// Make sure to defer both cancels (in reverse order)
	defer timeoutCancel()
	defer cancel()
	maxWorkers := dc.Config.MaxWorkers
	var wg sync.WaitGroup

	// Mutex and condition variable for synchronization
	var mu sync.Mutex
	cond := sync.NewCond(&mu)

	// Counter for active workers
	activeWorkers := 0

	// Start workers
	for i := 1; i <= maxWorkers; i++ {
		wg.Add(1)
		go dc.worker(i, jobsChannel, &wg, &mu, cond, &activeWorkers, dc.resultChannel, ctxTimeout)
	}

	// Send initial jobs
	for _, job := range dc.jobs {
		fmt.Printf("Main: Sending job %s with depth %d\n", job.Purl, job.Depth)
		jobsChannel <- job
	}

	// Start the completion monitor
	go func() {
		mu.Lock()
		for {

			select {
			case <-ctxTimeout.Done():
				fmt.Println("Context cancelled. Closing jobs channel.")
				close(jobsChannel)
				mu.Unlock()
				return
			default:
				dc.processResult(&wg, ctxTimeout)
				if activeWorkers == 0 && len(jobsChannel) == 0 {
					fmt.Println("All workers idle and queue empty. Closing jobs channel.")
					close(jobsChannel)
					mu.Unlock()
					cancel() // Signal all goroutines to stop via context
					return
				}
			}

			// Wait until there might be a completion condition
			cond.Wait() //
			fmt.Printf("Checking condition to close jobs channel. Workers %d, Jobs: %d", activeWorkers, len(jobsChannel))
			// Check if we're done
		}
	}()

	// Start a goroutine to collect results
	var resultWg sync.WaitGroup
	resultWg.Add(1)
	dc.processResult(&resultWg, ctxTimeout)

	wg.Wait()
	fmt.Println("All workers have exited. Processing completed.")

	close(dc.resultChannel)
	// Wait for all workers to exit
	resultWg.Wait()
}

func (dc *DependencyCollector) processResult(wg *sync.WaitGroup, ctx context.Context) {
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				// Context was cancelled, stop processing results
				fmt.Println("Results processor stopping due to context cancellation")
				return

			case result, ok := <-dc.resultChannel:
				if !ok {
					// Channel was closed, all results processed
					fmt.Println("Results processor: channel closed, exiting")
					return
				}
				// Process the result
				dc.Callback(result)
			}
		}
	}()
}

func (dc *DependencyCollector) worker(id int, jobs chan Job, wg *sync.WaitGroup, mu *sync.Mutex, cond *sync.Cond, activeWorkers *int, results chan Result, ctx context.Context) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			// Context was cancelled, stop the worker
			fmt.Printf("Worker %d stopping due to context cancellation\n", id)
			mu.Lock()
			*activeWorkers--
			cond.Signal() // Signal the completion monitor
			mu.Unlock()
			return

		case job, ok := <-jobs:
			if !ok {
				// Channel was closed
				fmt.Printf("Worker %d stopping due to closed jobs channel\n", id)
				return
			}
			// Mark as active
			mu.Lock()
			(*activeWorkers)++
			fmt.Printf("Worker %d: Started job %s at depth %d (Active workers: %d)\n",
				id, job.Purl, job.Depth, *activeWorkers)
			mu.Unlock()

			cacheKey := job.Purl + "@" + job.Version
			// First try with a read lock only
			dc.mapMutex.RLock()
			transitiveDependencies, exists := dc.cache[cacheKey]
			dc.mapMutex.RUnlock()

			if !exists {
				transitiveDependencies, _ = dc.dependencyModel.GetDependencies(job.Purl, job.Version, job.Ecosystem)
				if len(transitiveDependencies) > 0 {
					dc.mapMutex.Lock()
					dc.cache[cacheKey] = transitiveDependencies
					dc.mapMutex.Unlock()
				}
			}
			// sanitize versions
			var transitiveDependenciesPurls []string
			var sanitizedDependencies []models.UnresolvedDependency
			for _, ud := range transitiveDependencies {
				fixedVersion, err := PickFirstVersionFromNpmJsRange(ud.Requirement)
				fmt.Printf("Resolving requirement %s, to %s\n", ud.Requirement, fixedVersion)
				if err != nil {
					continue
				}
				sanitizedDependencies = append(sanitizedDependencies, models.UnresolvedDependency{
					Purl:        ud.Purl,
					Requirement: fixedVersion,
				})
				transitiveDependenciesPurls = append(transitiveDependenciesPurls, ud.Purl+"@"+fixedVersion)
			}

			// Generate new jobs with depth-1
			newJobDepth := job.Depth - 1

			results <- Result{
				Parent: job.Purl + "@" + job.Version, //parent purl
				Purls:  transitiveDependenciesPurls,  //transitives dependencies
			}

			// Only add new jobs if depth would be > 0
			if newJobDepth > 0 {
				for _, transitive := range sanitizedDependencies {
					fmt.Printf("Worker %d: Generated new job %s at depth %d\n", id, transitive.Purl, newJobDepth)

					jobs <- Job{Purl: transitive.Purl, Version: transitive.Requirement, Ecosystem: job.Ecosystem, Depth: newJobDepth}
				}
			}

			fmt.Printf("Worker %d: Completed job %s at depth %d\n", id, job.Purl, newJobDepth)

			// Mark as idle and signal
			mu.Lock()
			(*activeWorkers)--
			fmt.Printf("Worker %d: Became idle (Active workers: %d)\n", id, *activeWorkers)
			// Signal that status has changed
			cond.Signal()
			mu.Unlock()

		}
	}
	fmt.Printf("Worker %d: Exiting\n", id)
}
