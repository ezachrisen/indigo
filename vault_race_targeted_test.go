package indigo_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
)

// TestRaceConditionRulesMapMutation specifically targets the Rules map race condition
// This test is designed to maximize the chance of detecting the specific race where
// one goroutine modifies Rules map while another reads it during sortChildRules().
func TestRaceConditionRulesMapMutation(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	schema := &indigo.Schema{
		ID: "race_test",
		Elements: []indigo.DataElement{
			{Name: "x", Type: indigo.Int{}},
		},
	}

	eng := indigo.NewEngine(cel.NewEvaluator(cel.FixedSchema(schema)))

	// Create a rule with many children to increase race likelihood
	root := indigo.NewRule("root", "true")
	for i := 0; i < 50; i++ {
		child := indigo.NewRule(fmt.Sprintf("child_%d", i), "x > 0")
		root.Add(child)
	}

	v, err := indigo.NewVault(eng, root)
	if err != nil {
		t.Fatalf("vault creation failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Goroutine 1: Continuously update rules (modifies Rules map)
	wg.Add(1)
	go func() {
		defer wg.Done()
		counter := 0
		for {
			select {
			case <-stop:
				return
			case <-ctx.Done():
				return
			default:
			}

			// Update a rule with new expression
			targetID := fmt.Sprintf("child_%d", counter%50)
			newRule := indigo.NewRule(targetID, fmt.Sprintf("x > %d", counter%100))

			_ = v.Mutate(indigo.Update(newRule))
			counter++
		}
	}()

	// Goroutine 2: Continuously evaluate (reads Rules map during sort)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			case <-ctx.Done():
				return
			default:
			}

			rule := v.ImmutableRule()
			_, _ = eng.Eval(context.Background(), rule, map[string]any{"x": 42})
		}
	}()

	// Goroutine 3: Hammer the Rules map with many mutations
	wg.Add(1)
	go func() {
		defer wg.Done()
		counter := 0
		for {
			select {
			case <-stop:
				return
			case <-ctx.Done():
				return
			default:
			}

			// Add many rules rapidly
			newID := fmt.Sprintf("added_%d", counter)
			newRule := indigo.NewRule(newID, "x > 50")
			parent := "root"

			if counter%2 == 0 {
				parentID := fmt.Sprintf("child_%d", counter%50)
				_ = v.Mutate(indigo.Add(newRule, parentID))
			} else {
				_ = v.Mutate(indigo.Add(newRule, parent))
			}
			counter++
		}
	}()

	// Let them run until timeout
	<-ctx.Done()
	close(stop)

	// Wait for all goroutines with a timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("Test completed successfully (race detector should find issues)")
	case <-time.After(15 * time.Second):
		t.Fatal("Test goroutines did not complete in time")
	}
}

// TestParallelEvalDuringMutation tests parallel evaluation while mutations are happening
// This tests the specific race between sortChildRules() and Rules map modification
func TestParallelEvalDuringMutation(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	schema := &indigo.Schema{
		ID: "parallel_mutation",
		Elements: []indigo.DataElement{
			{Name: "x", Type: indigo.Int{}},
		},
	}

	eng := indigo.NewEngine(cel.NewEvaluator(cel.FixedSchema(schema)))

	// Create rule tree with structure that will stress sortChildRules
	root := indigo.NewRule("root", "true")

	// Create parent rules with many children
	for p := 0; p < 5; p++ {
		parent := indigo.NewRule(fmt.Sprintf("parent_%d", p), "x > 0")
		for c := 0; c < 20; c++ {
			child := indigo.NewRule(fmt.Sprintf("parent_%d_child_%d", p, c), "x > 5")
			parent.Add(child)
		}
		root.Add(parent)
	}

	v, err := indigo.NewVault(eng, root)
	if err != nil {
		t.Fatalf("vault creation failed: %v", err)
	}

	var wg sync.WaitGroup

	// Concurrent parallel evaluators - they call sortChildRules
	for evaluator := 0; evaluator < 5; evaluator++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				rule := v.ImmutableRule()
				// Use extreme parallel settings to stress sortChildRules more
				_, _ = eng.Eval(context.Background(), rule, map[string]any{"x": 50},
					indigo.Parallel(5, 50))  // High parallelism
			}
		}(evaluator)
	}

	// Concurrent mutators - they modify Rules map
	for mutator := 0; mutator < 5; mutator++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				parentID := fmt.Sprintf("parent_%d", i%5)
				childID := fmt.Sprintf("parent_%d_child_%d_added_%d", i%5, i%20, i)
				newRule := indigo.NewRule(childID, fmt.Sprintf("x > %d", i))

				_ = v.Mutate(indigo.Add(newRule, parentID))
			}
		}(mutator)
	}

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("Parallel eval + mutation test completed")
	case <-time.After(30 * time.Second):
		t.Fatal("Test timed out - potential deadlock")
	}
}

// TestSortedRulesRaceCondition specifically targets the sortChildRules race
// by repeatedly calling it via Eval while modifying the underlying Rules map
func TestSortedRulesRaceCondition(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	schema := &indigo.Schema{
		ID: "sorted_race",
		Elements: []indigo.DataElement{
			{Name: "x", Type: indigo.Int{}},
		},
	}

	eng := indigo.NewEngine(cel.NewEvaluator(cel.FixedSchema(schema)))

	// Create a rule with many children
	root := indigo.NewRule("root", "true")
	for i := 0; i < 100; i++ {
		child := indigo.NewRule(fmt.Sprintf("child_%d", i), "x > 0")
		root.Add(child)
	}

	v, err := indigo.NewVault(eng, root)
	if err != nil {
		t.Fatalf("vault creation failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// Goroutine 1: Continuously eval (calls sortChildRules internally)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			rule := v.ImmutableRule()
			_, _ = eng.Eval(context.Background(), rule, map[string]any{"x": 50})
		}
	}()

	// Goroutine 2: Continuously add to Rules map
	wg.Add(1)
	go func() {
		defer wg.Done()
		counter := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			newRule := indigo.NewRule(fmt.Sprintf("added_%d", counter), "x > 0")
			_ = v.Mutate(indigo.Add(newRule, "root"))
			counter++
		}
	}()

	// Goroutine 3: Continuously delete and update
	wg.Add(1)
	go func() {
		defer wg.Done()
		counter := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			targetID := fmt.Sprintf("child_%d", counter%100)
			updated := indigo.NewRule(targetID, fmt.Sprintf("x > %d", counter%50))
			_ = v.Mutate(indigo.Update(updated))
			counter++
		}
	}()

	<-ctx.Done()

	// Wait for goroutines
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("SortedRules race test completed")
	case <-time.After(10 * time.Second):
		t.Fatal("Test timed out")
	}
}

// TestDestinationShardRaceCondition targets the destinationShard function
// which walks the tree while Rules maps may be modified
func TestDestinationShardRaceCondition(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	schema := &indigo.Schema{
		ID: "shard_race",
		Elements: []indigo.DataElement{
			{Name: "type", Type: indigo.String{}},
		},
	}

	eng := indigo.NewEngine(cel.NewEvaluator(cel.FixedSchema(schema)))

	// Create sharded rule structure
	root := indigo.NewRule("root", "true")

	shard1 := indigo.NewRule("shard_a", `type == "A"`)
	shard1.Meta = func(r *indigo.Rule) bool {
		return r != nil
	}

	shard2 := indigo.NewRule("shard_b", `type == "B"`)
	shard2.Meta = func(r *indigo.Rule) bool {
		return r != nil
	}

	root.Shards = []*indigo.Rule{shard1, shard2}

	v, err := indigo.NewVault(eng, root)
	if err != nil {
		t.Fatalf("vault creation failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// Goroutine 1: Add rules (triggers destinationShard)
	wg.Add(1)
	go func() {
		defer wg.Done()
		counter := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			newRule := indigo.NewRule(fmt.Sprintf("rule_%d", counter), "true")
			_ = v.Mutate(indigo.Add(newRule, "root"))
			counter++
		}
	}()

	// Goroutine 2: Update rules (may move between shards)
	wg.Add(1)
	go func() {
		defer wg.Done()
		counter := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			targetID := fmt.Sprintf("rule_%d", counter%100)
			updated := indigo.NewRule(targetID, "true")
			_ = v.Mutate(indigo.Update(updated))
			counter++
		}
	}()

	<-ctx.Done()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("DestinationShard race test completed")
	case <-time.After(10 * time.Second):
		t.Fatal("Test timed out")
	}
}
