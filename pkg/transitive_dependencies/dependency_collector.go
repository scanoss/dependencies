package transitive_dependencies

import (
	"fmt"
	"scanoss.com/dependencies/pkg/models"
	"sync"
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
}

type Component struct {
	Purl    string
	Version string
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
			Purl:      component.Purl,
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
		go dc.worker(i, jobsChannel, &wg, &mu, cond, &activeWorkers, resultsChannel)
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
			if activeWorkers == 0 && len(jobsChannel) == 0 {
				fmt.Println("All workers idle and queue empty. Closing jobs channel.")
				close(jobsChannel)
				mu.Unlock()
				return
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
	go func() {
		defer resultWg.Done()
		for result := range resultsChannel {
			dc.Callback(result)
		}
	}()

	wg.Wait()
	fmt.Println("All workers have exited. Processing completed.")

	close(resultsChannel)
	// Wait for all workers to exit
	resultWg.Wait()
}

func (dc *DependencyCollector) worker(id int, jobs chan Job, wg *sync.WaitGroup, mu *sync.Mutex, cond *sync.Cond, activeWorkers *int, results chan Result) {
	defer wg.Done()
	for job := range jobs {
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

	fmt.Printf("Worker %d: Exiting\n", id)
}
