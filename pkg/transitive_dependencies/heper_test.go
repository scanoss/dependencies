package transitive_dependencies

import (
	"context"
	"fmt"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	"testing"
)

func TestProcessCollectorResult(t *testing.T) {
	// Setup test SugaredLogger
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared S", err)
	}
	defer zlog.SyncZap()
	ctx := context.Background()
	ctx = ctxzap.ToContext(ctx, zlog.L)
	s := ctxzap.Extract(ctx).Sugar()

	t.Run("successfully processes dependencies", func(t *testing.T) {
		// Setup mock graph
		mockGraph := NewDepGraph()

		// Create test data
		parent := DependencyJob{
			PurlName:  "parent-dep",
			Version:   "1.0.0",
			Ecosystem: "npmjs",
			Depth:     0,
		}
		child1 := DependencyJob{
			PurlName:  "child-dep1",
			Version:   "1.0.0",
			Ecosystem: "npmjs",
			Depth:     1,
		}
		child2 := DependencyJob{
			PurlName:  "child-dep2",
			Version:   "2.0.0",
			Ecosystem: "npmjs",
			Depth:     1,
		}

		result := Result{
			Parent:                 parent,
			TransitiveDependencies: []DependencyJob{child1, child2},
		}

		// Execute function under test
		callback := ProcessCollectorResult(s, mockGraph, 10)
		shouldStop := callback(result)
		if shouldStop {
			t.Error("should not signal to stop processing")
		}
	})

	t.Run("stops when max dependencies reached", func(t *testing.T) {
		// Setup mock graph that reports being at the max size
		mockGraph := NewDepGraph()

		// Create test data
		parent := DependencyJob{
			PurlName:  "parent-dep",
			Version:   "1.0.0",
			Ecosystem: "npmjs",
			Depth:     0,
		}
		child1 := DependencyJob{
			PurlName:  "child-dep1",
			Version:   "1.0.0",
			Ecosystem: "npmjs",
			Depth:     1,
		}
		child2 := DependencyJob{
			PurlName:  "child-dep2",
			Version:   "1.0.0",
			Ecosystem: "npmjs",
			Depth:     1,
		}

		result := Result{
			Parent:                 parent,
			TransitiveDependencies: []DependencyJob{child1, child2},
		}

		// Execute function under test
		callback := ProcessCollectorResult(s, mockGraph, 2)
		shouldStop := callback(result)
		fmt.Printf("STOP: %v", shouldStop)
		if !shouldStop {
			t.Error("should signal to stop processing")
		}
	})

}
