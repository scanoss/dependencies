package transdep

import (
	"go.uber.org/zap"
)

// ProcessCollectorResult process collector results and save a result in an adjacencyList structure.
func ProcessCollectorResult(s *zap.SugaredLogger, depGraph *DependencyGraph, maxDependencyResponseSize int) func(Result) bool {
	return func(result Result) bool {
		parentDep, err := ExtractDependencyFromJob(result.Parent)
		if err != nil {
			s.Errorf("failed to convert dependency:%v, %v", result.Parent, err)
			return false
		}
		for _, td := range result.TransitiveDependencies {
			tDep, tdErr := ExtractDependencyFromJob(td)
			if tdErr == nil {
				// Connects a dependency within a child
				depGraph.Connect(parentDep, tDep)
				// Stop if a max limit response is reached
				if depGraph.GetDependenciesCount() == maxDependencyResponseSize {
					return true
				}
			} else {
				s.Errorf("failed to convert transitive dependency:%v, %v", td.PurlName, tdErr)
			}
		}
		return false
	}
}
