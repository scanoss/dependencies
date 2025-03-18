package transitive_dependencies

import (
	"fmt"
	"scanoss.com/dependencies/pkg/models"
	"strings"
	"sync"
)

type Job struct {
	Purl      string
	Version   string
	Depth     int
	Ecosystem string
}

type Result struct {
	Parent    string
	Purls     []string
	Depth     int
	Ecosystem string
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
	jobMutex        sync.RWMutex
	cache           map[string][]models.UnresolvedDependency
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

func NewDependencyCollector(c func(result Result), config DependencyCollectorCfg, model *models.DependencyModel) *DependencyCollector {
	return &DependencyCollector{
		Callback:        c,
		Config:          config,
		dependencyModel: model,
		mapMutex:        sync.RWMutex{},
		cache:           make(map[string][]models.UnresolvedDependency),
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
	resultsChannel := make(chan Result, dc.Config.MaxQueueLimit)

	var mu sync.Mutex
	channelCond := sync.NewCond(&mu)

	pendingJobs := len(dc.jobs)

	maxWorkers := dc.Config.MaxWorkers
	var wg sync.WaitGroup

	// Start workers
	for i := 1; i <= maxWorkers; i++ {
		wg.Add(1)
		go dc.worker(i, &jobsChannel, resultsChannel, &wg, channelCond, &mu)
	}

	// Send initial jobs
	for _, job := range dc.jobs {
		fmt.Printf("Main: Sending job %s with depth %d\n", job.Purl, job.Depth)
		jobsChannel <- job
	}

	wg.Add(1)

	go func(jc *chan Job) {
		for result := range resultsChannel {
			dc.Callback(result)
			// Increment result counter
			if result.Depth > 0 {
				pendingJobs += len(result.Purls)
			}
			for _, p := range result.Purls {
				if result.Depth > 0 {
					key := strings.Split(p, "@")
					newJob := Job{Purl: key[0], Version: key[1], Depth: result.Depth, Ecosystem: result.Ecosystem}
					select {
					case *jc <- newJob:
						fmt.Printf("Main: Sent job %s\n", newJob.Purl)
					default:
						fmt.Printf("❌Cannot add more job to process:%s\n", newJob.Purl)
						oldCapacity := cap(*jc)
						newCapacity := oldCapacity * 2
						newChannel := make(chan Job, newCapacity)

						// First add the new job
						newChannel <- newJob

						// Then transfer as many existing jobs as possible from the old channel
						// without blocking
						drainJobs := func() {
							for {
								select {
								case job, ok := <-*jc:
									if !ok {
										return // Channel closed
									}
									newChannel <- job
								default:
									return // No more items without blocking
								}
							}
						}
						drainJobs()

						// Replace the old channel with the new one
						*jc = newChannel

					}
				}
			}

			pendingJobs--
			fmt.Printf("Pending jobs: %d\n", pendingJobs)
			mu.Lock()
			channelCond.Broadcast()
			mu.Unlock()

			if pendingJobs == 0 {
				fmt.Printf("Pending results 0...")
				fmt.Printf("Bradcasting new jobs...: %d\n", pendingJobs)
				mu.Lock()
				channelCond.Broadcast()
				mu.Unlock()
				close(jobsChannel)
				wg.Done()

			}
		}

	}(&jobsChannel)

	wg.Wait()
	fmt.Println("All workers have exited. Processing completed.")

	close(resultsChannel)
}
func (dc *DependencyCollector) worker(id int, jobsPtr *chan Job, results chan Result, wg *sync.WaitGroup, cond *sync.Cond, mu *sync.Mutex) {
	fmt.Printf("Worker %d: Starting\n", id)
	defer wg.Done()
	for {
		// Try to get a job from the current channel
		var job Job
		var ok bool

		select {
		case job, ok = <-*jobsPtr:
			// Got a job or channel closed
			if !ok {
				// Channel was closed, exit worker
				fmt.Printf("Worker %d: Channel closed, exiting\n", id)
				return
			}

			// Process the job
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

			// Sanitize versions
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
			fmt.Printf("Worker %d: Adding new result with depth %d\n", id, newJobDepth)
			fmt.Printf("Result size: %d\n", len(results))

			newResult := Result{
				Parent:    job.Purl + "@" + job.Version, // parent purl
				Purls:     transitiveDependenciesPurls,  // transitives dependencies
				Depth:     newJobDepth,
				Ecosystem: job.Ecosystem,
			}

			select {
			case results <- newResult:
				fmt.Printf("✅ Worker %d: New Result added\n", id)
			default:
				fmt.Printf("❌ Worker %d: Cannot add result - results channel full\n", id)
				continue
			}

		default:
			// No job available right now, wait on condition
			fmt.Printf("Worker %d: No jobs available, waiting...\n", id)
			mu.Lock()
			cond.Wait() // Will release lock and wait until signaled
			mu.Unlock()
			fmt.Printf("Worker %d: Woke up to check for new jobs\n", id)
		}
	}
}
