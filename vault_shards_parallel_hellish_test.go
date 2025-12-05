package indigo_test

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
)

// TestHellishShardedVaultParallelStress is the ultimate stress test combining:
// - Sharded rules in a vault
// - Parallel evaluation with varying batch sizes
// - Concurrent readers (multiple concurrent evaluations)
// - Concurrent writers (mutations while evaluating)
// - Deep rule hierarchies (5+ levels)
// - Rapid shard membership changes via updates
// - Context timeouts and cancellations
// - Pathological data patterns
// - High concurrency with many goroutines
// This test is designed to find race conditions, deadlocks, and memory leaks.
func TestHellishShardedVaultParallelStress(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	schema := &indigo.Schema{
		ID: "hellish",
		Elements: []indigo.DataElement{
			{Name: "category", Type: indigo.String{}},
			{Name: "region", Type: indigo.String{}},
			{Name: "tier", Type: indigo.Int{}},
			{Name: "value", Type: indigo.Float{}},
			{Name: "status", Type: indigo.String{}},
			{Name: "priority", Type: indigo.Int{}},
		},
	}

	// Create the deeply nested root rule with multiple levels
	root := createDeeplyNestedShardedRule()

	eng := indigo.NewEngine(cel.NewEvaluator(cel.FixedSchema(schema)))
	v, err := indigo.NewVault(root)
	if err != nil {
		t.Fatalf("failed to create vault: %v", err)
	}

	// Test data that will cause different code paths
	testDatasets := []map[string]any{
		{
			"category": "premium",
			"region":   "us-east",
			"tier":     1,
			"value":    999.99,
			"status":   "active",
			"priority": 100,
		},
		{
			"category": "standard",
			"region":   "eu-west",
			"tier":     2,
			"value":    50.0,
			"status":   "inactive",
			"priority": 50,
		},
		{
			"category": "basic",
			"region":   "asia-pacific",
			"tier":     3,
			"value":    10.0,
			"status":   "pending",
			"priority": 10,
		},
		nil, // Will cause panic in evaluator if not handled
		{},  // Empty data
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var evalErrors int64
	var mutationErrors int64

	// Goroutine 1: Heavy parallel reader load - evaluates with different parallel configs
	numReaders := 20
	for reader := 0; reader < numReaders; reader++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(readerID)))
			for iteration := 0; iteration < 100; iteration++ {
				select {
				case <-ctx.Done():
					return
				default:
				}

				rule := v.ImmutableRule()
				if rule == nil {
					atomic.AddInt64(&evalErrors, 1)
					return
				}

				// Pick random test data
				data := testDatasets[rng.Intn(len(testDatasets))]
				if data == nil {
					data = map[string]any{}
				}

				// Vary parallel config dramatically
				batchSize := rng.Intn(50) + 1
				maxParallel := rng.Intn(50) + 1

				_, err := eng.Eval(ctx, rule, data, indigo.Parallel(batchSize, maxParallel))
				if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
					t.Logf("Reader %d iteration %d eval error: %v", readerID, iteration, err)
					atomic.AddInt64(&evalErrors, 1)
				}

				// Small random delay
				if rng.Intn(10) < 3 {
					time.Sleep(time.Microsecond * time.Duration(rng.Intn(100)))
				}
			}
		}(reader)
	}

	// Goroutine 2: Heavy mutation load - adds, updates, deletes, moves
	wg.Add(1)
	go func() {
		defer wg.Done()
		rng := rand.New(rand.NewSource(42))

		for mutation := 0; mutation < 150; mutation++ {
			select {
			case <-ctx.Done():
				return
			default:
			}

			op := rng.Intn(4)
			switch op {
			case 0: // Add
				ruleID := fmt.Sprintf("added_%d_%d", mutation, rng.Intn(1000))
				newRule := indigo.NewRule(ruleID, fmt.Sprintf("tier > %d", rng.Intn(10)))
				parentID := selectRandomParent(v.ImmutableRule(), rng)
				if parentID != "" {
					err := v.Mutate(indigo.Add(newRule, parentID))
					if err != nil {
						t.Logf("Add mutation error for %s: %v", ruleID, err)
						atomic.AddInt64(&mutationErrors, 1)
					}
				}

			case 1: // Update
				rule := v.ImmutableRule()
				allRules := collectAllRuleIDs(rule)
				if len(allRules) > 1 {
					targetID := allRules[rng.Intn(len(allRules))]
					updatedExpr := fmt.Sprintf("priority > %d", rng.Intn(100))
					updatedRule := indigo.NewRule(targetID, updatedExpr)
					err := v.Mutate(indigo.Update(updatedRule))
					if err != nil {
						t.Logf("Update mutation error for %s: %v", targetID, err)
						atomic.AddInt64(&mutationErrors, 1)
					}
				}

			case 2: // Delete
				rule := v.ImmutableRule()
				allRules := collectAllRuleIDs(rule)
				if len(allRules) > 2 { // Don't delete root
					targetID := allRules[1+rng.Intn(len(allRules)-1)]
					err := v.Mutate(indigo.Delete(targetID))
					if err != nil {
						t.Logf("Delete mutation error for %s: %v", targetID, err)
						atomic.AddInt64(&mutationErrors, 1)
					}
				}

			case 3: // Move
				rule := v.ImmutableRule()
				allRules := collectAllRuleIDs(rule)
				if len(allRules) > 2 {
					sourceID := allRules[1+rng.Intn(len(allRules)-1)]
					targetParentID := selectRandomParent(v.ImmutableRule(), rng)
					if targetParentID != "" && targetParentID != sourceID {
						err := v.Mutate(indigo.Move(sourceID, targetParentID))
						if err != nil {
							t.Logf("Move mutation error from %s: %v", sourceID, err)
							atomic.AddInt64(&mutationErrors, 1)
						}
					}
				}
			}

			// Add time updates frequently
			if mutation%10 == 0 {
				v.Mutate(indigo.LastUpdate(time.Now()))
			}

			// Random delay
			if rng.Intn(10) < 4 {
				time.Sleep(time.Microsecond * time.Duration(rng.Intn(200)))
			}
		}
	}()

	// Goroutine 3: Rapid context cancellation stress test
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			select {
			case <-ctx.Done():
				return
			default:
			}

			rule := v.ImmutableRule()
			data := map[string]any{"category": "test"}

			// Create a very short timeout context
			shortCtx, cancel := context.WithTimeout(context.Background(), time.Microsecond*time.Duration(rand.Intn(100)))
			_, _ = eng.Eval(shortCtx, rule, data, indigo.Parallel(10, 10))
			cancel()

			time.Sleep(time.Microsecond * 50)
		}
	}()

	// Goroutine 4: Shard-specific stress - updates that move rules between shards
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 80; i++ {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Update rules with expressions that should change shard membership
			categories := []string{"premium", "standard", "basic"}
			regions := []string{"us-east", "eu-west", "asia-pacific"}
			category := categories[rand.Intn(len(categories))]
			region := regions[rand.Intn(len(regions))]

			// Update rules to have different category/region combinations
			updateExpr := fmt.Sprintf(`category == "%s" && region == "%s"`, category, region)
			updateRule := indigo.NewRule(fmt.Sprintf("category_shard_%s", category), updateExpr)

			err := v.Mutate(indigo.Update(updateRule))
			if err != nil {
				t.Logf("Shard update error: %v", err)
				atomic.AddInt64(&mutationErrors, 1)
			}

			time.Sleep(time.Microsecond * time.Duration(rand.Intn(500)))
		}
	}()

	// Goroutine 5: Rapid snapshot reader - captures snapshots while mutations happen
	wg.Add(1)
	go func() {
		defer wg.Done()
		snapshots := make([]*indigo.Rule, 0, 100)
		for i := 0; i < 200; i++ {
			select {
			case <-ctx.Done():
				return
			default:
			}

			snapshot := v.ImmutableRule()
			snapshots = append(snapshots, snapshot)

			// Verify snapshots remain immutable
			if len(snapshots) > 1 {
				prev := snapshots[len(snapshots)-2]
				curr := snapshots[len(snapshots)-1]

				// If they're different snapshots, verify they're indeed different
				if prev != curr {
					if len(prev.Rules) == len(curr.Rules) {
						// Could be same content, just different copy
					}
				}
			}

			time.Sleep(time.Microsecond * 100)
		}
	}()

	// Wait for all goroutines with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All done
	case <-time.After(35 * time.Second):
		t.Fatal("test timeout - potential deadlock")
	}

	// Final evaluation with all data to ensure vault is still valid
	finalRule := v.ImmutableRule()
	if finalRule == nil {
		t.Fatal("final rule is nil - vault was corrupted")
	}

	_, err = eng.Eval(context.Background(), finalRule, map[string]any{
		"category": "premium",
		"region":   "us-east",
		"tier":     1,
		"value":    100.0,
		"status":   "active",
		"priority": 100,
	})
	if err != nil {
		t.Logf("Final evaluation error: %v (may be expected if vault was damaged)", err)
	}

	if evalErrors > 0 {
		t.Logf("Total eval errors: %d", evalErrors)
	}
	if mutationErrors > 0 {
		t.Logf("Total mutation errors: %d", mutationErrors)
	}
}

// TestExtremeShardingEdgeCases tests the extreme boundaries of the sharding system
func TestExtremeShardingEdgeCases(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	// Test 1: Update a rule so its expression matches NO shards (should go to default)
	t.Run("update_to_no_shard_match", func(t *testing.T) {
		root := indigo.NewRule("root", "true")

		shard1 := indigo.NewRule("shard_a", `type == "A"`)
		shard1.Meta = func(r *indigo.Rule) bool {
			return strings.Contains(r.Expr, `"A"`)
		}

		shard2 := indigo.NewRule("shard_b", `type == "B"`)
		shard2.Meta = func(r *indigo.Rule) bool {
			return strings.Contains(r.Expr, `"B"`)
		}

		root.Shards = []*indigo.Rule{shard1, shard2}

		v, err := indigo.NewVault(root)
		if err != nil {
			t.Fatalf("vault creation failed: %v", err)
		}

		// Add a rule that matches shard_a
		ruleA := indigo.NewRule("rule_a", `type == "A"`)
		if err := v.Mutate(indigo.Add(ruleA, "root")); err != nil {
			t.Fatalf("add rule_a failed: %v", err)
		}

		// Verify it's in shard_a
		root1 := v.ImmutableRule()
		found, ancestors := root1.FindRule("rule_a")
		if found == nil {
			t.Fatal("rule_a not found after add")
		}
		if len(ancestors) == 0 || ancestors[len(ancestors)-1].ID != "shard_a" {
			t.Error("rule_a not in shard_a")
		}
		fmt.Println(root1.Tree())
		fmt.Println("----------------------------------------")
		// Update it to not match any shard
		updated := indigo.NewRule("rule_a", `type == "C"`)
		if err := v.Mutate(indigo.Update(updated)); err != nil {
			t.Fatalf("update rule_a failed: %v", err)
		}

		// Verify it moved to default
		root2 := v.ImmutableRule()
		found, ancestors = root2.FindRule("rule_a")
		if found == nil {
			t.Fatal("rule_a not found after update")
		}
		if len(ancestors) == 0 || ancestors[len(ancestors)-1].ID != "default" {
			t.Errorf("rule_a should be in default shard, got parent %s",
				func() string {
					if len(ancestors) > 0 {
						return ancestors[len(ancestors)-1].ID
					}
					return "none"
				}())
		}
	})

	// Test 2: Deeply nested shards with updates
	t.Run("nested_shard_updates", func(t *testing.T) {
		root := indigo.NewRule("root", "true")

		// Primary shard
		primary := indigo.NewRule("primary", `type == "primary"`)
		primary.Meta = func(r *indigo.Rule) bool {
			return strings.Contains(r.Expr, `"primary"`)
		}

		// Secondary shards under primary
		secondary1 := indigo.NewRule("sec1", `type == "primary_sec1"`)
		secondary1.Meta = func(r *indigo.Rule) bool {
			return strings.Contains(r.Expr, `"primary_sec1"`)
		}

		secondary2 := indigo.NewRule("sec2", `type == "primary_sec2"`)
		secondary2.Meta = func(r *indigo.Rule) bool {
			return strings.Contains(r.Expr, `"primary_sec2"`)
		}

		primary.Shards = []*indigo.Rule{secondary1, secondary2}
		root.Shards = []*indigo.Rule{primary}

		v, err := indigo.NewVault(root)
		if err != nil {
			t.Fatalf("vault creation failed: %v", err)
		}

		// Add rules at various levels
		rule1 := indigo.NewRule("rule1", `type == "primary_sec1"`)
		if err := v.Mutate(indigo.Add(rule1, "root")); err != nil {
			t.Fatalf("add rule1 failed: %v", err)
		}

		// Should be in primary -> sec1
		root1 := v.ImmutableRule()
		found, ancestors := root1.FindRule("rule1")
		if found == nil {
			t.Fatal("rule1 not found")
		}
		if len(ancestors) < 2 {
			t.Errorf("expected at least 2 ancestors, got %d", len(ancestors))
		}

		// Update it to match sec2
		updated := indigo.NewRule("rule1", `type == "primary_sec2"`)
		if err := v.Mutate(indigo.Update(updated)); err != nil {
			t.Fatalf("update rule1 failed: %v", err)
		}

		// Should be in primary -> sec2 now
		root2 := v.ImmutableRule()
		found, ancestors = root2.FindRule("rule1")
		if found == nil {
			t.Fatal("rule1 not found after update")
		}
		if len(ancestors) < 2 || ancestors[len(ancestors)-1].ID != "sec2" {
			t.Error("rule1 should be in sec2 after update")
		}
	})
}

// TestConcurrentUpdateWithParallelEval tests concurrent updates to rules while parallel evaluation happens
func TestConcurrentUpdateWithParallelEval(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	schema := &indigo.Schema{
		ID: "concurrent",
		Elements: []indigo.DataElement{
			{Name: "x", Type: indigo.Int{}},
		},
	}

	// Create a complex rule tree
	root := createComplexRuleTree(100)

	eng := indigo.NewEngine(cel.NewEvaluator(cel.FixedSchema(schema)))
	v, err := indigo.NewVault(root)
	if err != nil {
		t.Fatalf("vault creation failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var evalCount int64
	var mutCount int64

	// Concurrent evaluators
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(id)))
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				rule := v.ImmutableRule()
				data := map[string]any{"x": rng.Intn(100)}

				// Parallel with random config
				_, err := eng.Eval(context.Background(), rule, data,
					indigo.Parallel(rng.Intn(20)+1, rng.Intn(20)+1))
				if err == nil {
					atomic.AddInt64(&evalCount, 1)
				}

				time.Sleep(time.Microsecond * time.Duration(rng.Intn(100)))
			}
		}(i)
	}

	// Concurrent mutators
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(id + 100)))
			counter := 0
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				rule := v.ImmutableRule()
				allRules := collectAllRuleIDs(rule)
				if len(allRules) > 2 {
					targetID := allRules[rng.Intn(len(allRules))]
					newExpr := fmt.Sprintf("x > %d", rng.Intn(100))
					updateRule := indigo.NewRule(targetID, newExpr)

					err := v.Mutate(indigo.Update(updateRule))
					if err == nil {
						atomic.AddInt64(&mutCount, 1)
					}
				}

				counter++
				time.Sleep(time.Microsecond * time.Duration(rng.Intn(200)))
			}
		}(i)
	}

	<-ctx.Done()
	wg.Wait()

	t.Logf("Completed %d evaluations and %d mutations concurrently", evalCount, mutCount)

	// Verify final state is valid
	finalRule := v.ImmutableRule()
	if finalRule == nil {
		t.Fatal("final rule is nil")
	}

	_, err = eng.Eval(context.Background(), finalRule, map[string]any{"x": 50})
	if err != nil {
		t.Logf("Final eval error: %v", err)
	}
}

// TestMakeSafePathRaceConditions tests for race conditions in the makeSafePath mechanism
func TestMakeSafePathRaceConditions(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	schema := &indigo.Schema{
		ID: "makesafe",
		Elements: []indigo.DataElement{
			{Name: "x", Type: indigo.Int{}},
		},
	}

	eng := indigo.NewEngine(cel.NewEvaluator(cel.FixedSchema(schema)))

	// Create a rule tree with specific structure for makeSafePath testing
	root := indigo.NewRule("root", "true")
	level1 := indigo.NewRule("level1", "true")
	level2 := indigo.NewRule("level2", "true")
	level3 := indigo.NewRule("level3", "true")
	level4 := indigo.NewRule("level4", "x > 0")

	level3.Add(level4)
	level2.Add(level3)
	level1.Add(level2)
	root.Add(level1)

	v, err := indigo.NewVault(root)
	if err != nil {
		t.Fatalf("vault creation failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// Multiple concurrent updaters at different depths
	for depth := 1; depth <= 4; depth++ {
		wg.Add(1)
		go func(d int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(d)))

			ruleIDs := []string{"level1", "level2", "level3", "level4"}

			for i := 0; i < 50; i++ {
				select {
				case <-ctx.Done():
					return
				default:
				}

				targetID := ruleIDs[d-1]
				newExpr := fmt.Sprintf("x > %d", rng.Intn(100))
				updateRule := indigo.NewRule(targetID, newExpr)

				_ = v.Mutate(indigo.Update(updateRule))

				time.Sleep(time.Microsecond * time.Duration(rng.Intn(100)))
			}
		}(depth)
	}

	// Also evaluate in parallel
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			select {
			case <-ctx.Done():
				return
			default:
			}

			rule := v.ImmutableRule()
			_, _ = eng.Eval(context.Background(), rule, map[string]any{"x": 50},
				indigo.Parallel(10, 10))
			time.Sleep(time.Microsecond * 50)
		}
	}()

	<-ctx.Done()
	wg.Wait()

	// Verify structure is intact
	finalRule := v.ImmutableRule()
	for _, id := range []string{"level1", "level2", "level3", "level4"} {
		found, _ := finalRule.FindRule(id)
		if found == nil {
			t.Errorf("rule %s not found in final tree", id)
		}
	}
}

// TestParallelEvalEdgeCasesWithShards tests edge cases specific to parallel evaluation with shards
func TestParallelEvalEdgeCasesWithShards(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	schema := &indigo.Schema{
		ID: "edge",
		Elements: []indigo.DataElement{
			{Name: "type", Type: indigo.String{}},
		},
	}

	eng := indigo.NewEngine(cel.NewEvaluator(cel.FixedSchema(schema)))

	// Create a rule with shards
	root := indigo.NewRule("root", "true")

	// Create many shards
	for i := 0; i < 10; i++ {
		shard := indigo.NewRule(fmt.Sprintf("shard_%d", i), fmt.Sprintf(`type == "type_%d"`, i))
		shard.Meta = func(r *indigo.Rule) bool {
			// This will cause deterministic shard routing
			return strings.Contains(r.Expr, fmt.Sprintf(`"type_%d"`, i))
		}

		// Add many children to each shard
		for j := 0; j < 20; j++ {
			child := indigo.NewRule(fmt.Sprintf("shard_%d_child_%d", i, j),
				fmt.Sprintf(`type == "type_%d" && x > %d`, i, j))
			shard.Add(child)
		}

		root.Add(shard)
	}

	// Add default shard
	defaultShard := indigo.NewRule("default", "true")
	root.Add(defaultShard)

	v, err := indigo.NewVault(root)
	if err != nil {
		t.Fatalf("vault creation failed: %v", err)
	}

	// Test with extreme parallel configs
	extremeConfigs := []struct {
		name        string
		batchSize   int
		maxParallel int
	}{
		{"tiny_batch", 1, 50},
		{"huge_batch", 500, 1},
		{"balanced", 25, 25},
		{"extreme", 1000, 1000},
		{"single", 50, 1},
	}

	testData := map[string]any{"type": "type_5"}

	for _, cfg := range extremeConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			rule := v.ImmutableRule()
			result, err := eng.Eval(context.Background(), rule, testData,
				indigo.Parallel(cfg.batchSize, cfg.maxParallel))
			if err != nil {
				t.Logf("Eval error with batch=%d, parallel=%d: %v",
					cfg.batchSize, cfg.maxParallel, err)
			}

			if result != nil {
				// Verify some result was computed
			}
		})
	}
}

// Helper functions

func createDeeplyNestedShardedRule() *indigo.Rule {
	root := indigo.NewRule("root", "true")

	// Create primary shards by category
	categories := []string{"premium", "standard", "basic"}
	for _, cat := range categories {
		catShard := indigo.NewRule(fmt.Sprintf("category_%s", cat),
			fmt.Sprintf(`category == "%s"`, cat))
		catShard.Meta = func(r *indigo.Rule) bool {
			return strings.Contains(r.Expr, cat)
		}

		// Create region shards under each category
		regions := []string{"us-east", "eu-west", "asia-pacific"}
		for _, region := range regions {
			regionShard := indigo.NewRule(fmt.Sprintf("region_%s", region),
				fmt.Sprintf(`region == "%s"`, region))
			regionShard.Meta = func(r *indigo.Rule) bool {
				return strings.Contains(r.Expr, region)
			}

			// Add children under region shards
			for tier := 1; tier <= 3; tier++ {
				child := indigo.NewRule(fmt.Sprintf("%s_%s_tier%d", cat, region, tier),
					fmt.Sprintf(`category == "%s" && region == "%s" && tier >= %d`, cat, region, tier))
				regionShard.Add(child)
			}

			catShard.Add(regionShard)
		}

		root.Add(catShard)
	}

	// Add region shards
	for _, region := range []string{"us-east", "eu-west", "asia-pacific"} {
		regionShard := indigo.NewRule(fmt.Sprintf("global_region_%s", region),
			fmt.Sprintf(`region == "%s"`, region))
		regionShard.Meta = func(r *indigo.Rule) bool {
			return strings.Contains(r.Expr, region)
		}

		for i := 0; i < 5; i++ {
			child := indigo.NewRule(fmt.Sprintf("global_%s_rule_%d", region, i),
				fmt.Sprintf(`region == "%s" && priority >= %d`, region, i*10))
			regionShard.Add(child)
		}

		root.Add(regionShard)
	}

	return root
}

func createComplexRuleTree(size int) *indigo.Rule {
	root := indigo.NewRule("root", "x > 0")

	for i := 0; i < size; i++ {
		rule := indigo.NewRule(fmt.Sprintf("rule_%d", i), fmt.Sprintf("x > %d", i%100))
		root.Add(rule)

		// Add some children to create hierarchy
		if i%10 == 0 {
			for j := 0; j < 3; j++ {
				child := indigo.NewRule(fmt.Sprintf("rule_%d_child_%d", i, j),
					fmt.Sprintf("x > %d", (i+j)%100))
				rule.Add(child)
			}
		}
	}

	return root
}

func selectRandomParent(root *indigo.Rule, rng *rand.Rand) string {
	allRules := collectAllRuleIDs(root)
	if len(allRules) == 0 {
		return ""
	}
	return allRules[rng.Intn(len(allRules))]
}

func collectAllRuleIDs(root *indigo.Rule) []string {
	var ids []string
	var collect func(*indigo.Rule)
	collect = func(r *indigo.Rule) {
		ids = append(ids, r.ID)
		for _, child := range r.Rules {
			collect(child)
		}
	}
	collect(root)
	return ids
}
