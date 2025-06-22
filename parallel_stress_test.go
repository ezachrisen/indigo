package indigo_test

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
)

// Test for race conditions when multiple goroutines evaluate the same rule tree concurrently
func TestParallelRaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	// Create a rule tree with many child rules
	root := createLargeRuleTree(500)

	err := engine.Compile(root)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	data := map[string]any{
		"value": 42,
		"name":  "test",
	}

	// Run multiple concurrent evaluations to stress test for race conditions
	const numGoroutines = 50
	const numIterations = 20

	var wg sync.WaitGroup
	var errors int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				// Mix parallel and sequential evaluations
				var err error
				//				if j%2 == 0 {
				_, err = engine.Eval(context.Background(), root, data,
					indigo.Parallel(10, 20))
				// } else {
				// 	_, err = engine.Eval(context.Background(), root, data)
				// }

				if err != nil {
					atomic.AddInt64(&errors, 1)
					t.Errorf("worker %d iteration %d failed: %v", workerID, j, err)
				}
			}
		}(i)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("detected %d errors during concurrent evaluation", errors)
	}
}

// Test for goroutine leaks when context is cancelled
func TestParallelGoroutineLeaksWithCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	engine := indigo.NewEngine(cel.NewEvaluator())
	root := createLargeRuleTree(1000)

	err := engine.Compile(root)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	data := map[string]any{"value": 42}

	// Get initial goroutine count
	initialGoroutines := runtime.NumGoroutine()

	// Test cancellation at various points
	const numTests = 100

	for i := 0; i < numTests; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)

		// Start evaluation but cancel quickly
		_, err := engine.Eval(ctx, root, data, indigo.Parallel(5, 50))

		cancel()

		// Context cancellation is expected
		if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
			t.Errorf("unexpected error (not cancellation): %v", err)
		}

		// Give time for goroutines to clean up
		time.Sleep(1 * time.Millisecond)
	}

	// Allow more time for cleanup
	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	runtime.GC() // Force GC twice to ensure cleanup

	finalGoroutines := runtime.NumGoroutine()

	// Allow some tolerance for background goroutines
	if finalGoroutines > initialGoroutines+5 {
		t.Errorf("potential goroutine leak: started with %d, ended with %d",
			initialGoroutines, finalGoroutines)
	}
}

// Test for goroutine leaks with evaluation errors
func TestParallelGoroutineLeaksWithErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "value", Type: indigo.Int{}},
		},
	}

	// Create rules that will cause evaluation errors
	root := &indigo.Rule{
		ID:     "root",
		Expr:   "true",
		Schema: schema,
		Rules:  make(map[string]*indigo.Rule),
	}

	// Add rules that will cause runtime errors (division by zero)
	for i := 0; i < 100; i++ {
		rule := &indigo.Rule{
			ID:         fmt.Sprintf("rule_%d", i),
			Expr:       "value / 0", // This will cause a runtime error
			Schema:     schema,
			ResultType: indigo.Int{},
		}
		root.Rules[rule.ID] = rule
	}

	err := engine.Compile(root)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	data := map[string]any{}

	initialGoroutines := runtime.NumGoroutine()

	// Run evaluations that will fail
	const numTests = 50

	for i := 0; i < numTests; i++ {
		_, err := engine.Eval(context.Background(), root, data,
			indigo.Parallel(10, 20))

		// We expect errors here due to invalid expressions
		if err == nil {
			t.Error("expected evaluation to fail due to invalid expressions")
		}

		time.Sleep(1 * time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	runtime.GC()

	finalGoroutines := runtime.NumGoroutine()

	if finalGoroutines > initialGoroutines+5 {
		t.Errorf("potential goroutine leak: started with %d, ended with %d",
			initialGoroutines, finalGoroutines)
	}
}

// Test memory usage with very large rule sets
func TestParallelMemoryStress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	// Create increasingly large rule trees
	sizes := []int{1000, 2000, 5000}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			root := createLargeRuleTree(size)

			err := engine.Compile(root)
			if err != nil {
				t.Fatalf("failed to compile rule tree of size %d: %v", size, err)
			}

			data := map[string]any{
				"value": 42,
				"name":  "test",
			}

			// Record memory before evaluation
			var m1 runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&m1)

			// Run parallel evaluation
			result, err := engine.Eval(context.Background(), root, data,
				indigo.Parallel(100, 50))
			if err != nil {
				t.Fatalf("evaluation failed for size %d: %v", size, err)
			}

			// Verify we got results
			if len(result.Results) == 0 {
				t.Error("expected some results from evaluation")
			}

			// Record memory after evaluation
			var m2 runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&m2)

			// Check for reasonable memory usage (this is a rough heuristic)
			// Handle the case where m1.Alloc might be larger than m2.Alloc due to GC
			var memUsed uint64
			if m2.Alloc > m1.Alloc {
				memUsed = m2.Alloc - m1.Alloc
			} else {
				memUsed = 0 // Memory was reclaimed by GC
			}

			if memUsed > 0 {
				memPerRule := memUsed / uint64(size)

				// If we're using more than 1MB per rule, something might be wrong
				if memPerRule > 1024*1024 {
					t.Errorf("high memory usage: %d bytes per rule for %d rules",
						memPerRule, size)
				}
			}
		})
	}
}

// Test edge cases with parallel processing
func TestParallelEdgeCases(t *testing.T) {
	engine := indigo.NewEngine(cel.NewEvaluator())
	data := map[string]any{"value": 42}

	testCases := []struct {
		name        string
		rule        *indigo.Rule
		batchSize   int
		maxParallel int
		shouldFail  bool
	}{
		{
			name: "zero_rules",
			rule: &indigo.Rule{
				ID:    "empty",
				Expr:  "true",
				Rules: make(map[string]*indigo.Rule),
			},
			batchSize:   10,
			maxParallel: 5,
			shouldFail:  false,
		},
		{
			name: "single_rule",
			rule: &indigo.Rule{
				ID:   "single",
				Expr: "true",
				Rules: map[string]*indigo.Rule{
					"child": {ID: "child", Expr: "true"},
				},
			},
			batchSize:   10,
			maxParallel: 5,
			shouldFail:  false,
		},
		{
			name:        "batch_larger_than_rules",
			rule:        createLargeRuleTree(5),
			batchSize:   100,
			maxParallel: 10,
			shouldFail:  false,
		},
		{
			name:        "very_small_batch",
			rule:        createLargeRuleTree(100),
			batchSize:   1,
			maxParallel: 50,
			shouldFail:  false,
		},
		{
			name:        "max_parallel_1",
			rule:        createLargeRuleTree(100),
			batchSize:   10,
			maxParallel: 1,
			shouldFail:  false,
		},
		{
			name:        "zero_batch_size",
			rule:        createLargeRuleTree(10),
			batchSize:   0,
			maxParallel: 5,
			shouldFail:  false, // Should fallback to sequential
		},
		{
			name:        "zero_max_parallel",
			rule:        createLargeRuleTree(10),
			batchSize:   5,
			maxParallel: 0,
			shouldFail:  false, // Should fallback to sequential
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := engine.Compile(tc.rule)
			if err != nil {
				t.Fatalf("failed to compile: %v", err)
			}

			result, err := engine.Eval(context.Background(), tc.rule, data,
				indigo.Parallel(tc.batchSize, tc.maxParallel))

			if tc.shouldFail {
				if err == nil {
					t.Error("expected evaluation to fail")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result == nil {
					t.Error("expected non-nil result")
				}
			}
		})
	}
}

// Test rapid context cancellation during parallel evaluation
func TestParallelRapidCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	engine := indigo.NewEngine(cel.NewEvaluator())
	root := createLargeRuleTree(500)

	err := engine.Compile(root)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	data := map[string]any{"value": 42}

	// Test rapid cancellation scenarios
	const numTests = 100

	for i := 0; i < numTests; i++ {
		ctx, cancel := context.WithCancel(context.Background())

		// Start evaluation
		go func() {
			_, _ = engine.Eval(ctx, root, data, indigo.Parallel(10, 20))
		}()

		// Cancel immediately
		cancel()

		// Small delay to allow goroutines to react to cancellation
		time.Sleep(time.Microsecond * 100)
	}

	// Give time for cleanup
	time.Sleep(50 * time.Millisecond)
}

// Test concurrent modifications during evaluation (should be safe since we compile first)
func TestParallelConcurrentModification(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	engine := indigo.NewEngine(cel.NewEvaluator())
	root := createLargeRuleTree(200)

	err := engine.Compile(root)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	data := map[string]any{"value": 42}

	var wg sync.WaitGroup
	var errors int64

	// Start evaluation goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_, err := engine.Eval(context.Background(), root, data,
				indigo.Parallel(20, 10))
			if err != nil {
				atomic.AddInt64(&errors, 1)
			}
			time.Sleep(time.Millisecond)
		}
	}()

	// Start rule modification goroutine (this should be safe since rules are compiled)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			// Create a new rule tree
			newRoot := createLargeRuleTree(200)
			engine.Compile(newRoot)
			time.Sleep(time.Millisecond)
		}
	}()

	wg.Wait()

	if errors > 0 {
		t.Errorf("detected %d errors during concurrent modification test", errors)
	}
}

// Test with extremely high parallelism settings
func TestParallelExtremeParallelism(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	engine := indigo.NewEngine(cel.NewEvaluator())
	root := createLargeRuleTree(1000)

	err := engine.Compile(root)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	data := map[string]any{"value": 42}

	// Test with very high parallelism
	testCases := []struct {
		name        string
		batchSize   int
		maxParallel int
	}{
		{"high_batch", 500, 100},
		{"high_parallel", 10, 500},
		{"both_high", 200, 200},
		{"extreme", 1000, 1000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), root, data,
				indigo.Parallel(tc.batchSize, tc.maxParallel))

			if err != nil {
				t.Errorf("evaluation failed with batch_size=%d, max_parallel=%d: %v",
					tc.batchSize, tc.maxParallel, err)
			}

			if result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

// Helper function to create a large rule tree for testing
func createLargeRuleTree(numRules int) *indigo.Rule {
	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "value", Type: indigo.Int{}},
			{Name: "name", Type: indigo.String{}},
		},
	}

	root := &indigo.Rule{
		ID:     "root",
		Expr:   "true",
		Schema: schema,
		Rules:  make(map[string]*indigo.Rule),
	}

	for i := 0; i < numRules; i++ {
		rule := &indigo.Rule{
			ID:     fmt.Sprintf("rule_%d", i),
			Expr:   "value > 0", // Simple expression that should evaluate to true
			Schema: schema,
		}
		root.Rules[rule.ID] = rule
	}

	return root
}

// Test for proper cleanup when evaluation panics
func TestParallelPanicRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	// Create a mock evaluator that panics on certain conditions
	engine := indigo.NewEngine(&panicEvaluator{})

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "value", Type: indigo.Int{}},
		},
	}

	root := &indigo.Rule{
		ID:     "root",
		Expr:   "true",
		Schema: schema,
		Rules: map[string]*indigo.Rule{
			"panic_rule":  {ID: "panic_rule", Expr: "panic", Schema: schema},
			"normal_rule": {ID: "normal_rule", Expr: "true", Schema: schema},
		},
	}

	err := engine.Compile(root)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	data := map[string]any{}

	initialGoroutines := runtime.NumGoroutine()

	// This should handle panics gracefully
	_, err = engine.Eval(context.Background(), root, data,
		indigo.Parallel(10, 5))

	// We expect an error due to the panic
	if err == nil {
		t.Error("expected evaluation to fail due to panic")
	}

	// Verify the error message indicates it was a panic
	if !strings.Contains(err.Error(), "panic during parallel rule evaluation") {
		t.Errorf("expected panic error message, got: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	runtime.GC()

	finalGoroutines := runtime.NumGoroutine()

	// Check that we didn't leak goroutines even with panics
	if finalGoroutines > initialGoroutines+5 {
		t.Errorf("potential goroutine leak after panic: started with %d, ended with %d",
			initialGoroutines, finalGoroutines)
	}
}

// Mock evaluator that panics on specific expressions
type panicEvaluator struct{}

func (e *panicEvaluator) Compile(expr string, _ indigo.Schema, _ indigo.Type, _ bool, _ bool) (any, error) {
	return expr, nil
}

func (e *panicEvaluator) Evaluate(data map[string]any, expr string, s indigo.Schema, self any, prog any, resultType indigo.Type, returnDiagnostics bool) (any, *indigo.Diagnostics, error) {
	if expr == "panic" {
		panic("test panic")
	}
	return true, nil, nil
}
