package indigo_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
)

func TestParallelEvaluation(t *testing.T) {
	ctx := context.Background()
	e := indigo.NewEngine(cel.NewEvaluator())

	// Create a parent rule with multiple child rules
	parent := indigo.NewRule("parent", "true")

	// Add 10 child rules that each take some time to evaluate
	for i := 0; i < 10; i++ {
		child := indigo.NewRule(fmt.Sprintf("child_%d", i), "true")
		parent.Rules[child.ID] = child
	}

	// Compile the rules
	err := e.Compile(parent)
	if err != nil {
		t.Fatal(err)
	}

	data := map[string]interface{}{}

	// Test sequential evaluation
	start := time.Now()
	result1, err := e.Eval(ctx, parent, data)
	if err != nil {
		t.Fatal(err)
	}
	sequentialTime := time.Since(start)

	// Test parallel evaluation with batch size 3
	start = time.Now()
	result2, err := e.Eval(ctx, parent, data, indigo.EnableParallel(3))
	if err != nil {
		t.Fatal(err)
	}
	parallelTime := time.Since(start)

	// Both should have the same results
	if result1.Pass != result2.Pass {
		t.Errorf("Results differ: sequential=%v, parallel=%v", result1.Pass, result2.Pass)
	}

	if len(result1.Results) != len(result2.Results) {
		t.Errorf("Number of child results differ: sequential=%d, parallel=%d",
			len(result1.Results), len(result2.Results))
	}

	// Verify all child results are present
	for childID, seqResult := range result1.Results {
		parResult, exists := result2.Results[childID]
		if !exists {
			t.Errorf("Child result missing in parallel execution: %s", childID)
			continue
		}
		if seqResult.Pass != parResult.Pass {
			t.Errorf("Child result differs for %s: sequential=%v, parallel=%v",
				childID, seqResult.Pass, parResult.Pass)
		}
	}

	t.Logf("Sequential time: %v, Parallel time: %v", sequentialTime, parallelTime)
}

func TestParallelEvaluationWithErrors(t *testing.T) {
	e := indigo.NewEngine(cel.NewEvaluator())

	// Create a parent rule with multiple child rules, one of which will fail
	parent := indigo.NewRule("parent", "true")

	// Add some good child rules
	for i := 0; i < 5; i++ {
		child := indigo.NewRule(fmt.Sprintf("child_%d", i), "true")
		parent.Rules[child.ID] = child
	}

	// Add a bad child rule that will cause an error
	badChild := indigo.NewRule("bad_child", "undefined_variable > 0")
	parent.Rules[badChild.ID] = badChild

	// Compile the rules (this should fail due to undefined_variable)
	err := e.Compile(parent)
	if err == nil {
		t.Fatal("Expected compilation error for undefined variable")
	}

	// This test verifies that error handling works correctly
	t.Logf("Got expected compilation error: %v", err)
}

func TestParallelEvaluationWithStopConditions(t *testing.T) {
	ctx := context.Background()
	e := indigo.NewEngine(cel.NewEvaluator())

	// Create a parent rule with multiple child rules
	parent := indigo.NewRule("parent", "true")

	// Add child rules with different results
	for i := 0; i < 10; i++ {
		var expr string
		if i == 2 {
			expr = "false" // This will be the first failing rule
		} else {
			expr = "true"
		}
		child := indigo.NewRule(fmt.Sprintf("child_%d", i), expr)
		parent.Rules[child.ID] = child
	}

	// Compile the rules
	err := e.Compile(parent)
	if err != nil {
		t.Fatal(err)
	}

	data := map[string]interface{}{}

	// Test with StopFirstNegativeChild - should work with parallel too
	// Note: With parallel processing, we can't guarantee exact early stopping
	// behavior since batches run concurrently, but the final result should be consistent
	result, err := e.Eval(ctx, parent, data, indigo.EnableParallel(3), indigo.StopFirstNegativeChild(true))
	if err != nil {
		t.Fatal(err)
	}

	// The result should still be correct even if not all rules were processed
	if result.Pass {
		t.Error("Expected result to be false due to failing child rule")
	}

	t.Logf("Parallel evaluation with StopFirstNegativeChild completed, result: %v", result.Pass)
}

func TestParallelEvaluationBatchSizes(t *testing.T) {
	ctx := context.Background()
	e := indigo.NewEngine(cel.NewEvaluator())

	// Create a parent rule with 20 child rules
	parent := indigo.NewRule("parent", "true")

	for i := 0; i < 20; i++ {
		child := indigo.NewRule(fmt.Sprintf("child_%d", i), "true")
		parent.Rules[child.ID] = child
	}

	// Compile the rules
	err := e.Compile(parent)
	if err != nil {
		t.Fatal(err)
	}

	data := map[string]interface{}{}

	// Test different batch sizes
	batchSizes := []int{1, 5, 10, 20, 50}

	for _, batchSize := range batchSizes {
		t.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(t *testing.T) {
			result, err := e.Eval(ctx, parent, data, indigo.EnableParallel(batchSize))
			if err != nil {
				t.Fatal(err)
			}

			if !result.Pass {
				t.Error("Expected result to be true")
			}

			if len(result.Results) != 20 {
				t.Errorf("Expected 20 child results, got %d", len(result.Results))
			}
		})
	}
}
