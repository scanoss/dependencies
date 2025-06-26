package transdep

import (
	"go.uber.org/zap"
)

// ProcessCollectorResult process collector results and save result in a adjacencyList structure.
func ProcessCollectorResult(s *zap.SugaredLogger, depGraph *DependencyGraph, maxDependencyResponseSize int) func(Result) bool {
	return func(result Result) bool {
		parentDep, err := ExtractDependencyFromJob(result.Parent)
		if err != nil {
			s.Errorf("failed to convert dependency:%v, %v", result.Parent, err)
			return false
		}
		for _, td := range result.TransitiveDependencies {
			tDep, err := ExtractDependencyFromJob(td)
			if err == nil {
				// Connects a dependency within a child
				depGraph.Connect(parentDep, tDep)
				// Stop if max limit response is reached
				if depGraph.GetDependenciesCount() == maxDependencyResponseSize {
					return true
				}
			} else {
				s.Errorf("failed to convert transitive dependency:%v, %v", td.PurlName, err)
			}
		}
		return false
	}
}
