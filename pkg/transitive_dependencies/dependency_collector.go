package transitive_dependencies

import (
	"container/list"
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

	maxWorkers := dc.Config.MaxWorkers
	var wg sync.WaitGroup

	// Start workers
	for i := 1; i <= maxWorkers; i++ {
		wg.Add(1)
		go dc.worker(i, jobsChannel, resultsChannel, &wg)
	}

	// Send initial jobs
	for _, job := range dc.jobs {
		fmt.Printf("Main: Sending job %s with depth %d\n", job.Purl, job.Depth)
		jobsChannel <- job
	}

	wg.Add(1)
	go func() {
		backlog := list.New()
		pendingJobs := len(dc.jobs)

		processResult := func(r Result) {
			dc.Callback(r)

			count := 0
			if r.Depth > 0 {
				count = len(r.Purls)
			}

			pendingJobs += count

			for _, purl := range r.Purls {
				key := strings.Split(purl, "@")
				if r.Depth > 0 {
					newJob := Job{Purl: key[0], Version: key[1], Depth: r.Depth, Ecosystem: r.Ecosystem}
					backlog.PushBack(newJob)
				}
			}

			// Only decrement one because we processed only one purl
			pendingJobs--
		}

		for {
			if element := backlog.Front(); element != nil {
				select {
				case jobsChannel <- element.Value.(Job):
					backlog.Remove(element)

				case r := <-resultsChannel:
					processResult(r)
				}
			} else {
				r := <-resultsChannel //Block until a result is ready to be consumed
				processResult(r)
			}

			if pendingJobs == 0 {
				wg.Done()
				close(jobsChannel)
				break
			}
		}
	}()

	wg.Wait()
	fmt.Println("All workers have exited. Processing completed.")

	close(resultsChannel)
}

func (dc *DependencyCollector) worker(id int, jobs chan Job, results chan Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {

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
			Parent:    job.Purl + "@" + job.Version, //parent purl
			Purls:     transitiveDependenciesPurls,  //transitives dependencies
			Depth:     newJobDepth,
			Ecosystem: job.Ecosystem,
		}

	}

	fmt.Printf("Worker %d: Exiting\n", id)
}
