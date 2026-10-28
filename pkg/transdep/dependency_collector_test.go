package transdep

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/jmoiron/sqlx"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	myconfig "scanoss.com/dependencies/pkg/config"
	"scanoss.com/dependencies/pkg/models"
)

// setupTestDependencyCollector creates a DependencyCollector for testing purposes.
func setupTestDependencyCollector(t *testing.T) (*DependencyCollector, *DependencyGraph, func()) {
	t.Helper() // Marks this function as a test helper

	// Setup logger
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}

	// Create context with logger
	ctx := context.Background()
	ctx = ctxzap.ToContext(ctx, zlog.L)
	s := ctxzap.Extract(ctx).Sugar()
	// Load config
	myConfig, err := myconfig.NewServerConfig(nil)
	if err != nil {
		t.Fatalf("an error '%s' was not expected when loading config", err)
	}
	// Setup test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	// An in-memory SQLite database is per connection: a second one would be empty.
	db.SetMaxOpenConns(1)
	// Load test data
	err = models.LoadTestSQLData(db, nil, nil)
	if err != nil {
		t.Fatalf("an error '%s' was not expected when loading test data", err)
	}
	// Create dependency collector configuration
	dependencyCollectorCfg := DependencyCollectorCfg{
		MaxWorkers:    myConfig.TransitiveResources.MaxWorkers,
		MaxQueueLimit: 5,
		TimeOut:       myConfig.TransitiveResources.TimeOut,
	}
	// Create dependency graph
	depGraph := NewDepGraph()
	// Create dependency collector
	transitiveDependencyCollector := NewDependencyCollector(
		ctx,
		ProcessCollectorResult(s, depGraph, 1),
		dependencyCollectorCfg,
		models.NewDependencyModel(ctx, s, db),
		s)
	// Return cleanup function
	cleanup := func() {
		models.CloseDB(db)
		zlog.SyncZap()
	}
	return transitiveDependencyCollector, depGraph, cleanup
}

func TestDependencyCollector_InitJobsDependency(t *testing.T) {
	transitiveDependencyCollector, _, cleanup := setupTestDependencyCollector(t)
	defer cleanup()
	tests := []struct {
		name    string
		jobs    []DependencyJob
		wantErr bool
	}{
		{
			name: "Valid dependency collector jobs",
			jobs: []DependencyJob{
				{
					PurlName:  "example/package1",
					Version:   "1.0.0",
					Depth:     1,
					Ecosystem: "npm",
				},
				{
					PurlName:  "example/package2",
					Version:   "2.3.1",
					Depth:     2,
					Ecosystem: "npm",
				},
				{
					PurlName:  "example/library",
					Version:   "0.9.5",
					Depth:     1,
					Ecosystem: "maven",
				},
				{
					PurlName:  "example/tool",
					Version:   "3.2.1",
					Depth:     3,
					Ecosystem: "golang",
				},
			},
			wantErr: false,
		},
		{
			name:    "Empty dependency collector jobs",
			jobs:    []DependencyJob{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := transitiveDependencyCollector.InitJobs(tt.jobs)
			if tt.wantErr && err == nil {
				t.Fatalf("an error '%s' was expected when initializing dependency collector", err)
			}
		})
	}
}

func TestDependencyCollector_GetTransitiveDependencies(t *testing.T) {
	transitiveDependencyCollector, depGraph, cleanup := setupTestDependencyCollector(t)
	defer cleanup()
	tests := []struct {
		name           string
		jobs           []DependencyJob
		expectedOutput []Dependency
		wantErr        bool
	}{
		{
			name: "Transitive dependencies depth 1",
			jobs: []DependencyJob{
				{
					PurlName:  "vue-phone",
					Version:   "1.0.9",
					Depth:     1,
					Ecosystem: "npm",
				},
			},
			// Every scope is followed (@types/node is a dev dependency), a requirement
			// resolves to the first version it names, and a scoped name keeps its scope.
			// isbinaryfile declares no requirement, so it cannot be followed.
			expectedOutput: []Dependency{
				{Purl: "pkg:npm/vue-phone", Version: "1.0.9"},
				{Purl: "pkg:npm/lodash", Version: "4.17.0"},
				{Purl: "pkg:npm/%40types%2Fnode", Version: "18.0.0"},
			},
			wantErr: false,
		},
		{
			name: "Unknown component",
			jobs: []DependencyJob{
				{
					PurlName:  "kcodecdaajvcmsp",
					Version:   "1.0.0",
					Depth:     1,
					Ecosystem: "npm",
				},
			},
			expectedOutput: []Dependency{},
			wantErr:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := transitiveDependencyCollector.InitJobs(tt.jobs)
			if (err != nil) != tt.wantErr {
				t.Errorf("transitiveDependencyCollector.InitJobs() error = %v, wantErr %v", err, tt.wantErr)
			}
			transitiveDependencyCollector.Start()

			// Create a map of actual dependencies for quick lookup
			actualDepsMap := make(map[string]bool)
			for _, dep := range depGraph.Flatten() {
				key := dep.Purl + "@" + dep.Version
				actualDepsMap[key] = true
			}

			// Check if any expected dependency is missing
			for _, dep := range tt.expectedOutput {
				key := dep.Purl + "@" + dep.Version
				if !actualDepsMap[key] {
					t.Errorf("Missing dependency: %s", key)
					return
				}
			}
		})
	}
}

func TestDependencyCollector_Worker(t *testing.T) {
	transitiveDependencyCollector, _, cleanup := setupTestDependencyCollector(t)
	defer cleanup()
	// Test cases
	testCases := []struct {
		name            string
		cancelDelay     time.Duration
		expectedTimeout time.Duration
		jobsToSubmit    int
	}{
		{
			name:            "CancelBeforeProcessing",
			cancelDelay:     0, // Cancel immediately
			expectedTimeout: 100 * time.Millisecond,
			jobsToSubmit:    1,
		},
		{
			name:            "CancelDuringProcessing",
			cancelDelay:     50 * time.Millisecond, // Cancel during processing
			expectedTimeout: 200 * time.Millisecond,
			jobsToSubmit:    1,
		},
		{
			name:            "CancelWithMultipleJobs",
			cancelDelay:     10 * time.Millisecond,
			expectedTimeout: 200 * time.Millisecond,
			jobsToSubmit:    5, // Multiple jobs in queue
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			jobs := make(chan DependencyJob, 10)
			results := make(chan Result, 10)
			var wg sync.WaitGroup
			wg.Add(1)
			go transitiveDependencyCollector.worker(1, jobs, &wg, results, ctx)
			for i := 0; i < tt.jobsToSubmit; i++ {
				jobs <- DependencyJob{
					PurlName:  "scanoss",
					Version:   "0.15.7",
					Ecosystem: "npm",
					Depth:     2,
				}
			}
			// Delay and then cancel the context
			time.Sleep(tt.cancelDelay)
			cancel()
			// Create a channel to signal when the worker has exited
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()
			// Check if the worker exits within the expected timeout
			select {
			case <-done:
				t.Log("Worker exits within the expected timeout")
			case <-time.After(tt.expectedTimeout):
				t.Errorf("Worker did not exit within %v after cancellation", tt.expectedTimeout)
			}
		})
	}
}

// TestProcessResultCancellation tests that the processResult method
// properly handles context cancellation.
func TestProcessResultCancellation(t *testing.T) {
	dc, _, cleanup := setupTestDependencyCollector(t)
	defer cleanup()
	// Create context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	// Set up a wait group for the processor
	var wg sync.WaitGroup
	wg.Add(1)
	// Start the processor
	go dc.processResult(&wg, ctx, cancel)
	// Immediately cancel the context
	cancel()
	// Wait for processor to exit with timeout
	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()
	select {
	case <-waitCh:
		// Success: processor exited after cancellation
	case <-time.After(1 * time.Second):
		t.Fatal("Processor did not exit in time after context cancellation")
	}
}

// TestProcessResultJobCompletion tests that the processResult method
// exits when all jobs are completed.
func TestProcessResultJobCompletion(t *testing.T) {
	dc, _, cleanup := setupTestDependencyCollector(t)
	defer cleanup()
	// Create a result handler that continues processing
	resultHandler := func(result Result) bool {
		return false // Continue processing
	}
	dc.ResultHandler = resultHandler
	dc.pendingJobs = 1
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cleanup
	// Set up a wait group for the processor
	var wg sync.WaitGroup
	wg.Add(1)
	// Start the processor
	go dc.processResult(&wg, ctx, cancel)
	// Send a result with no new jobs to decrease pendingJobs to zero
	dc.resultChannel <- Result{
		Parent:                 DependencyJob{PurlName: "test", Version: "1.0.0", Ecosystem: "npm"},
		TransitiveDependencies: []DependencyJob{},
	}
	// Wait for processor to exit with timeout
	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()
	select {
	case <-waitCh:
		// Success: processor exited when all jobs were completed
	case <-time.After(1 * time.Second):
		t.Fatal("Processor did not exit in time after all jobs completed")
	}
}

// TestResultHandlerStopsProcessing tests that when the ResultHandler returns true,
// processing stops immediately with the appropriate debug message.
func TestResultHandlerStopsProcessing(t *testing.T) {
	dc, _, cleanup := setupTestDependencyCollector(t)
	defer cleanup()
	dc.pendingJobs = 1
	// Create a counter to track handler calls
	handlerCalls := 0
	// Create a result handler that signals to stop after the first call
	resultHandler := func(result Result) bool {
		handlerCalls++
		return true // Signal to stop
	}
	dc.ResultHandler = resultHandler
	// Create context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	// Set up a wait group for the processor
	var wg sync.WaitGroup
	wg.Add(1)
	// Start the processor
	go dc.processResult(&wg, ctx, cancel)
	// Send a result to trigger the handler
	dc.resultChannel <- Result{
		Parent: DependencyJob{PurlName: "test", Version: "1.0.0", Ecosystem: "npm"},
		TransitiveDependencies: []DependencyJob{
			{PurlName: "dep1", Version: "1.0.0", Depth: 2, Ecosystem: "npm"},
		},
	}
	// Wait for processor to exit with timeout
	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()
	select {
	case <-waitCh:
		// Success: processor exited
	case <-time.After(1 * time.Second):
		t.Fatal("Processor did not exit in time after handler signaled to stop")
	}
	// Verify handler was called once
	if handlerCalls != 1 {
		t.Errorf("Expected handler to be called once, got %d", handlerCalls)
	}
}
