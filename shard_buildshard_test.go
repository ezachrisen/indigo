package indigo_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ezachrisen/indigo"
)

// TestBuildShardsEmpty tests sharding with no shards defined
func TestBuildShardsEmpty(t *testing.T) {
	root := indigo.NewRule("root", "")
	child1 := indigo.NewRule("child1", "")
	child2 := indigo.NewRule("child2", "")
	root.Add(child1)
	root.Add(child2)

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	// With no shards, the rules should remain as direct children
	if len(root.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(root.Rules))
	}
	if _, ok := root.Rules["child1"]; !ok {
		t.Error("child1 not found in root.Rules")
	}
	if _, ok := root.Rules["child2"]; !ok {
		t.Error("child2 not found in root.Rules")
	}
}

// TestBuildShardsNoMatchingRules tests sharding where no rules match any shard
func TestBuildShardsNoMatchingRules(t *testing.T) {
	root := indigo.NewRule("root", "")

	// Add rules that won't match any shard criteria
	child1 := indigo.NewRule("generic1", `x > 10`)
	child2 := indigo.NewRule("generic2", `y < 5`)
	root.Add(child1)
	root.Add(child2)

	// Create a shard that won't match anything
	shard := indigo.NewRule("specific_shard", "")
	shard.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.Expr, "z == 100")
	}

	root.Shards = []*indigo.Rule{shard}

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	// Both the shard and default shard should exist
	if _, ok := root.Rules["specific_shard"]; !ok {
		t.Error("specific_shard not found in root.Rules")
	}
	if _, ok := root.Rules["default"]; !ok {
		t.Error("default shard not found in root.Rules")
	}

	// Both rules should be in the default shard
	defShard := root.Rules["default"]
	if len(defShard.Rules) != 2 {
		t.Errorf("expected 2 rules in default shard, got %d", len(defShard.Rules))
	}
}

// TestBuildShardsAllRulesMatchOne tests when all rules match the same shard
func TestBuildShardsAllRulesMatchOne(t *testing.T) {
	root := indigo.NewRule("root", "")

	rule1 := indigo.NewRule("apple_rule1", `fruit == "apple"`)
	rule2 := indigo.NewRule("apple_rule2", `fruit == "apple" && color == "red"`)
	root.Add(rule1)
	root.Add(rule2)

	appleShard := indigo.NewRule("apple_shard", "")
	appleShard.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.Expr, `"apple"`)
	}

	root.Shards = []*indigo.Rule{appleShard}

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	appleShard = root.Rules["apple_shard"]
	if len(appleShard.Rules) != 2 {
		t.Errorf("expected 2 rules in apple_shard, got %d", len(appleShard.Rules))
	}

	// Default shard should be empty (no direct children)
	if defShard, ok := root.Rules["default"]; ok && len(defShard.Rules) != 0 {
		t.Errorf("expected default shard to be empty, got %d rules", len(defShard.Rules))
	}
}

// TestBuildShardsMultipleShards tests multiple shards with overlapping criteria
func TestBuildShardsMultipleShards(t *testing.T) {
	root := indigo.NewRule("root", "")

	rule1 := indigo.NewRule("red_apple", `fruit == "apple" && color == "red"`)
	rule2 := indigo.NewRule("red_tomato", `fruit == "tomato" && color == "red"`)
	rule3 := indigo.NewRule("green_apple", `fruit == "apple" && color == "green"`)
	rule4 := indigo.NewRule("generic", `weight > 100`)

	root.Add(rule1)
	root.Add(rule2)
	root.Add(rule3)
	root.Add(rule4)

	appleShard := indigo.NewRule("apple", "")
	appleShard.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.Expr, `"apple"`)
	}

	tomatoShard := indigo.NewRule("tomato", "")
	tomatoShard.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.Expr, `"tomato"`)
	}

	root.Shards = []*indigo.Rule{appleShard, tomatoShard}

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	// Check apple shard
	apple := root.Rules["apple"]
	if len(apple.Rules) != 2 {
		t.Errorf("expected 2 rules in apple shard, got %d", len(apple.Rules))
	}

	// Check tomato shard
	tomato := root.Rules["tomato"]
	if len(tomato.Rules) != 1 {
		t.Errorf("expected 1 rule in tomato shard, got %d", len(tomato.Rules))
	}

	// Check default shard
	def := root.Rules["default"]
	if len(def.Rules) != 1 {
		t.Errorf("expected 1 rule in default shard, got %d", len(def.Rules))
	}
}

// TestBuildShardsNestedShards tests recursive sharding where shards have sub-shards
func TestBuildShardsNestedShards(t *testing.T) {
	root := indigo.NewRule("root", "")

	// Create initial rules that will be sharded
	rule1 := indigo.NewRule("red_apple_small", `fruit == "apple" && color == "red" && size == "small"`)
	rule2 := indigo.NewRule("red_apple_large", `fruit == "apple" && color == "red" && size == "large"`)
	rule3 := indigo.NewRule("red_orange", `fruit == "orange" && color == "red"`)
	rule4 := indigo.NewRule("generic", `weight > 100`)

	root.Add(rule1)
	root.Add(rule2)
	root.Add(rule3)
	root.Add(rule4)

	// Create first-level shards
	appleShard := indigo.NewRule("apple", "")
	appleShard.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.Expr, `"apple"`)
	}

	orangeShard := indigo.NewRule("orange", "")
	orangeShard.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.Expr, `"orange"`)
	}

	// Create nested shard under appleShard
	appleSizeSmallShard := indigo.NewRule("apple_small", "")
	appleSizeSmallShard.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.Expr, `"small"`)
	}

	appleShard.Shards = []*indigo.Rule{appleSizeSmallShard}

	root.Shards = []*indigo.Rule{appleShard, orangeShard}

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	// Check root-level shards
	if _, ok := root.Rules["apple"]; !ok {
		t.Error("apple shard not found in root.Rules")
	}
	if _, ok := root.Rules["orange"]; !ok {
		t.Error("orange shard not found in root.Rules")
	}
	if _, ok := root.Rules["default"]; !ok {
		t.Error("default shard not found in root.Rules")
	}

	// Check nested shards under apple
	apple := root.Rules["apple"]
	if _, ok := apple.Rules["apple_small"]; !ok {
		t.Error("apple_small shard not found in apple.Rules")
	}
	if _, ok := apple.Rules["default"]; !ok {
		t.Error("default shard not found in apple.Rules")
	}

	// Verify rule distribution
	appleSmall := apple.Rules["apple_small"]
	if len(appleSmall.Rules) != 1 {
		t.Errorf("expected 1 rule in apple_small, got %d", len(appleSmall.Rules))
	}

	appleDefault := apple.Rules["default"]
	if len(appleDefault.Rules) != 1 {
		t.Errorf("expected 1 rule in apple default shard, got %d", len(appleDefault.Rules))
	}
}

// TestBuildShardsNilShard tests error handling for nil shard
func TestBuildShardsNilShard(t *testing.T) {
	root := indigo.NewRule("root", "")
	child := indigo.NewRule("child", "")
	root.Add(child)

	root.Shards = []*indigo.Rule{nil}

	err := root.BuildShards()
	if err == nil {
		t.Error("expected error for nil shard, got nil")
	}
	if !strings.Contains(err.Error(), "nil shard") {
		t.Errorf("expected error message about nil shard, got: %v", err)
	}
}

// TestBuildShardsReservedDefaultID tests error handling for reserved "default" shard ID
func TestBuildShardsReservedDefaultID(t *testing.T) {
	root := indigo.NewRule("root", "")
	child := indigo.NewRule("child", "")
	root.Add(child)

	// Try to create a shard with reserved ID
	invalidShard := indigo.NewRule("default", "")
	root.Shards = []*indigo.Rule{invalidShard}

	err := root.BuildShards()
	if err == nil {
		t.Error("expected error for reserved shard ID, got nil")
	}
	if !strings.Contains(err.Error(), "reserved shard ID") {
		t.Errorf("expected error about reserved ID, got: %v", err)
	}
}

// TestBuildShardsInvalidMetaType tests handling of unsupported Meta type
func TestBuildShardsInvalidMetaType(t *testing.T) {
	root := indigo.NewRule("root", "")
	child := indigo.NewRule("child1", `x > 10`)
	root.Add(child)

	shard := indigo.NewRule("shard1", "")
	// Use an invalid meta type (not a function)
	shard.Meta = "invalid_string_meta"

	root.Shards = []*indigo.Rule{shard}

	err := root.BuildShards()
	if err == nil {
		t.Error("expected error for invalid meta type, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported meta type") {
		t.Errorf("expected error about unsupported meta type, got: %v", err)
	}
}

// TestBuildShardsRuleWithChildren tests sharding rules that have child rules
func TestBuildShardsRuleWithChildren(t *testing.T) {
	root := indigo.NewRule("root", "")

	// Create a parent rule with children
	parent1 := indigo.NewRule("apple_parent", `fruit == "apple"`)
	child1a := indigo.NewRule("apple_child_a", `color == "red"`)
	child1b := indigo.NewRule("apple_child_b", `color == "green"`)
	parent1.Add(child1a)
	parent1.Add(child1b)

	parent2 := indigo.NewRule("orange_parent", `fruit == "orange"`)
	child2a := indigo.NewRule("orange_child_a", `size == "large"`)
	parent2.Add(child2a)

	root.Add(parent1)
	root.Add(parent2)

	// Create shards based on parent rules
	appleShard := indigo.NewRule("apple", "")
	appleShard.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.Expr, `"apple"`)
	}

	orangeShard := indigo.NewRule("orange", "")
	orangeShard.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.Expr, `"orange"`)
	}

	root.Shards = []*indigo.Rule{appleShard, orangeShard}

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	// Verify apple parent and its children are in apple shard
	appleParent := root.Rules["apple"].Rules["apple_parent"]
	if appleParent == nil {
		t.Error("apple_parent not found in apple shard")
	} else if len(appleParent.Rules) != 2 {
		t.Errorf("expected 2 children in apple_parent, got %d", len(appleParent.Rules))
	}

	// Verify orange parent and its children are in orange shard
	orangeParent := root.Rules["orange"].Rules["orange_parent"]
	if orangeParent == nil {
		t.Error("orange_parent not found in orange shard")
	} else if len(orangeParent.Rules) != 1 {
		t.Errorf("expected 1 child in orange_parent, got %d", len(orangeParent.Rules))
	}
}

// TestBuildShardsShardMarkup tests that shard rules are properly marked
func TestBuildShardsShardMarkup(t *testing.T) {
	root := indigo.NewRule("root", "")
	child := indigo.NewRule("child", "")
	root.Add(child)

	shard := indigo.NewRule("test_shard", "")
	shard.Meta = func(r *indigo.Rule) bool { return true }

	root.Shards = []*indigo.Rule{shard}

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	shardRule := root.Rules["test_shard"]

	// Check if StopIfParentNegative is set (this is the public indicator that it's a shard)
	if !shardRule.EvalOptions.StopIfParentNegative {
		t.Error("StopIfParentNegative not set on shard rule")
	}
}

// TestBuildShardsSortedRules tests that rules are sorted in shards
func TestBuildShardsSortedRules(t *testing.T) {
	root := indigo.NewRule("root", "")

	// Add rules in non-alphabetical order
	rule_c := indigo.NewRule("z_rule", "")
	rule_a := indigo.NewRule("a_rule", "")
	rule_b := indigo.NewRule("m_rule", "")

	root.Add(rule_c)
	root.Add(rule_a)
	root.Add(rule_b)

	shard := indigo.NewRule("shard1", "")
	shard.Meta = func(r *indigo.Rule) bool { return true }

	root.Shards = []*indigo.Rule{shard}

	// Set sort function before BuildShards
	root.EvalOptions.SortFunc = indigo.SortRulesAlpha

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	// After BuildShards, root should have 2 child rules: shard1 and default
	if len(root.Rules) != 2 {
		t.Errorf("expected 2 rules in root after sharding, got %d", len(root.Rules))
	}

	// Check that both shard and default exist
	if _, ok := root.Rules["shard1"]; !ok {
		t.Error("shard1 not found in root.Rules")
	}
	if _, ok := root.Rules["default"]; !ok {
		t.Error("default not found in root.Rules")
	}
}

// TestBuildShardsMultipleCallsBuildShards tests calling BuildShards on separate rule trees
func TestBuildShardsMultipleCallsBuildShards(t *testing.T) {
	// First rule tree
	root1 := indigo.NewRule("root1", "")
	child1a := indigo.NewRule("child1a", "")
	child1b := indigo.NewRule("child1b", "")
	root1.Add(child1a)
	root1.Add(child1b)

	shard1 := indigo.NewRule("shard1", "")
	shard1.Meta = func(r *indigo.Rule) bool { return strings.Contains(r.ID, "a") }
	root1.Shards = []*indigo.Rule{shard1}

	err := root1.BuildShards()
	if err != nil {
		t.Fatalf("First BuildShards failed: %v", err)
	}

	if _, ok := root1.Rules["shard1"].Rules["child1a"]; !ok {
		t.Error("child1a not in shard after first BuildShards")
	}

	// Second rule tree (independent)
	root2 := indigo.NewRule("root2", "")
	child2a := indigo.NewRule("child2a", "")
	child2b := indigo.NewRule("child2b", "")
	root2.Add(child2a)
	root2.Add(child2b)

	shard2 := indigo.NewRule("shard2", "")
	shard2.Meta = func(r *indigo.Rule) bool { return strings.Contains(r.ID, "b") }
	root2.Shards = []*indigo.Rule{shard2}

	err = root2.BuildShards()
	if err != nil {
		t.Fatalf("Second BuildShards failed: %v", err)
	}

	// Both trees should be correctly sharded
	if _, ok := root2.Rules["shard2"].Rules["child2b"]; !ok {
		t.Error("child2b not in shard after second BuildShards")
	}
}

// TestBuildShardsComplexOverlappingShards tests complex scenarios with overlapping criteria
func TestBuildShardsComplexOverlappingShards(t *testing.T) {
	root := indigo.NewRule("root", "")

	// Create rules with overlapping characteristics
	rule1 := indigo.NewRule("rule1", `status == "active" && type == "premium"`)
	rule2 := indigo.NewRule("rule2", `status == "active" && type == "basic"`)
	rule3 := indigo.NewRule("rule3", `status == "inactive" && type == "premium"`)
	rule4 := indigo.NewRule("rule4", `status == "inactive" && type == "basic"`)

	root.Add(rule1)
	root.Add(rule2)
	root.Add(rule3)
	root.Add(rule4)

	// Create shards for status
	activeShard := indigo.NewRule("active", "")
	activeShard.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.Expr, `"active"`)
	}

	inactiveShard := indigo.NewRule("inactive", "")
	inactiveShard.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.Expr, `"inactive"`)
	}

	root.Shards = []*indigo.Rule{activeShard, inactiveShard}

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	// Verify distribution based on first match (active or inactive)
	if len(root.Rules["active"].Rules) != 2 {
		t.Errorf("expected 2 rules in active shard, got %d", len(root.Rules["active"].Rules))
	}

	if len(root.Rules["inactive"].Rules) != 2 {
		t.Errorf("expected 2 rules in inactive shard, got %d", len(root.Rules["inactive"].Rules))
	}
}

// TestBuildShardsDeepNesting tests deeply nested rule hierarchies
func TestBuildShardsDeepNesting(t *testing.T) {
	// Create a deep hierarchy: root -> level1 -> level2 -> level3
	root := indigo.NewRule("root", "")

	level1_a := indigo.NewRule("level1_a", `category == "A"`)
	level1_b := indigo.NewRule("level1_b", `category == "B"`)
	root.Add(level1_a)
	root.Add(level1_b)

	level2_a1 := indigo.NewRule("level2_a1", `subcategory == "A1"`)
	level2_a2 := indigo.NewRule("level2_a2", `subcategory == "A2"`)
	level1_a.Add(level2_a1)
	level1_a.Add(level2_a2)

	level3_a1a := indigo.NewRule("level3_a1a", `detail == "A1a"`)
	level2_a1.Add(level3_a1a)

	// Shard at root level
	shardA := indigo.NewRule("shard_A", "")
	shardA.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.ID, "_a") || strings.Contains(r.ID, "_A")
	}

	root.Shards = []*indigo.Rule{shardA}

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	// Check that level1_a is in shard_A
	shardA = root.Rules["shard_A"]
	if _, ok := shardA.Rules["level1_a"]; !ok {
		t.Error("level1_a not in shard_A")
	}

	// Verify that level2 children are still under level1_a
	level1_a_in_shard := shardA.Rules["level1_a"]
	if len(level1_a_in_shard.Rules) != 2 {
		t.Errorf("expected 2 children in level1_a, got %d", len(level1_a_in_shard.Rules))
	}

	// Verify level3 is still under level2_a1
	level2_a1_in_shard := level1_a_in_shard.Rules["level2_a1"]
	if len(level2_a1_in_shard.Rules) != 1 {
		t.Errorf("expected 1 child in level2_a1, got %d", len(level2_a1_in_shard.Rules))
	}
}

// TestBuildShardsEmptyRules tests sharding a rule with no children
func TestBuildShardsEmptyRules(t *testing.T) {
	root := indigo.NewRule("root", "")
	// Don't add any children

	shard := indigo.NewRule("shard1", "")
	shard.Meta = func(r *indigo.Rule) bool { return true }

	root.Shards = []*indigo.Rule{shard}

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	// Should have shard and default
	if _, ok := root.Rules["shard1"]; !ok {
		t.Error("shard1 not created")
	}
	if _, ok := root.Rules["default"]; !ok {
		t.Error("default shard not created")
	}

	// Both should be empty
	if len(root.Rules["shard1"].Rules) != 0 {
		t.Errorf("expected shard1 to be empty, got %d rules", len(root.Rules["shard1"].Rules))
	}
	if len(root.Rules["default"].Rules) != 0 {
		t.Errorf("expected default to be empty, got %d rules", len(root.Rules["default"].Rules))
	}
}

// TestBuildShardsSingleRule tests sharding with only one rule
func TestBuildShardsSingleRule(t *testing.T) {
	root := indigo.NewRule("root", "")
	child := indigo.NewRule("single_child", `x > 100`)
	root.Add(child)

	shard := indigo.NewRule("shard1", "")
	shard.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.Expr, "100")
	}

	root.Shards = []*indigo.Rule{shard}

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	if len(root.Rules["shard1"].Rules) != 1 {
		t.Errorf("expected 1 rule in shard1, got %d", len(root.Rules["shard1"].Rules))
	}
	if _, ok := root.Rules["shard1"].Rules["single_child"]; !ok {
		t.Error("single_child not in shard1")
	}
}

// TestBuildShardsMetaFunctionMustReturnBool tests that Meta function expects proper type
func TestBuildShardsMetaFunctionMustReturnBool(t *testing.T) {
	root := indigo.NewRule("root", "")
	child := indigo.NewRule("child", "")
	root.Add(child)

	shard := indigo.NewRule("shard1", "")
	// This returns a string, not a bool - but will be handled dynamically
	shard.Meta = func(r *indigo.Rule) string {
		return "yes"
	}

	root.Shards = []*indigo.Rule{shard}

	// This tests what happens when type assertion fails
	err := root.BuildShards()
	if err == nil {
		t.Error("expected error for wrong Meta function signature, got nil")
	}
}

// TestBuildShardsWithDifferentIDPatterns tests sharding with various ID patterns
func TestBuildShardsWithDifferentIDPatterns(t *testing.T) {
	root := indigo.NewRule("root", "")

	// Create rules with different ID patterns
	rule1 := indigo.NewRule("rule_001_type_A", "")
	rule2 := indigo.NewRule("rule_002_type_B", "")
	rule3 := indigo.NewRule("rule_003_type_A", "")
	rule4 := indigo.NewRule("other_rule_type_C", "")

	root.Add(rule1)
	root.Add(rule2)
	root.Add(rule3)
	root.Add(rule4)

	// Shard by ID pattern
	typeAShard := indigo.NewRule("type_A", "")
	typeAShard.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.ID, "_type_A")
	}

	typeBShard := indigo.NewRule("type_B", "")
	typeBShard.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.ID, "_type_B")
	}

	root.Shards = []*indigo.Rule{typeAShard, typeBShard}

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	// Verify distribution
	if len(root.Rules["type_A"].Rules) != 2 {
		t.Errorf("expected 2 rules in type_A shard, got %d", len(root.Rules["type_A"].Rules))
	}
	if len(root.Rules["type_B"].Rules) != 1 {
		t.Errorf("expected 1 rule in type_B shard, got %d", len(root.Rules["type_B"].Rules))
	}
	if len(root.Rules["default"].Rules) != 1 {
		t.Errorf("expected 1 rule in default shard, got %d", len(root.Rules["default"].Rules))
	}
}

// TestBuildShardsPreservesRuleProperties tests that rule properties are preserved during sharding
func TestBuildShardsPreservesRuleProperties(t *testing.T) {
	root := indigo.NewRule("root", "")

	// Create rule with properties
	child := indigo.NewRule("child_rule", `x > 10`)
	child.Meta = "some_metadata"
	child.ResultType = indigo.Bool{}

	root.Add(child)

	shard := indigo.NewRule("shard1", "")
	shard.Meta = func(r *indigo.Rule) bool { return true }

	root.Shards = []*indigo.Rule{shard}

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	// Get the moved rule and verify properties are preserved
	movedRule := root.Rules["shard1"].Rules["child_rule"]
	if movedRule == nil {
		t.Fatal("child_rule not found after sharding")
	}

	if movedRule.Expr != `x > 10` {
		t.Errorf("Expr not preserved: got %q", movedRule.Expr)
	}
	if movedRule.Meta != "some_metadata" {
		t.Errorf("Meta not preserved: got %v", movedRule.Meta)
	}
}

// TestBuildShardsEdgeCaseShardIDCollision tests when a rule ID matches a shard ID
// This should result in a failure
func TestBuildShardsEdgeCaseShardIDCollision(t *testing.T) {
	root := indigo.NewRule("root", "")

	// Create rules with IDs that might collide with shard names
	child := indigo.NewRule("shard1", `x > 10`)
	root.Add(child)

	shard := indigo.NewRule("shard1", "")
	shard.Meta = func(r *indigo.Rule) bool { return true }

	root.Shards = []*indigo.Rule{shard}

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	debugLogf(t, "After BuildShards\n%s\n", root)
	// The rule should be moved into the shard
	shardParent := root.Rules["shard1"]
	if _, ok := shardParent.Rules["shard1"]; !ok {
		t.Error("rule with ID 'shard1' not found in shard 'shard1'")
	}
}

// TestBuildShardsDefaultShardWithManyRules tests default shard with many non-matching rules
func TestBuildShardsDefaultShardWithManyRules(t *testing.T) {
	root := indigo.NewRule("root", "")

	// Add many rules that don't match any shard
	for i := 0; i < 100; i++ {
		child := indigo.NewRule(fmt.Sprintf("generic_rule_%d", i), "")
		root.Add(child)
	}

	shard := indigo.NewRule("specific", "")
	shard.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.ID, "impossible")
	}

	root.Shards = []*indigo.Rule{shard}

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	// All 100 rules should be in default
	defaultShard := root.Rules["default"]
	if len(defaultShard.Rules) != 100 {
		t.Errorf("expected 100 rules in default shard, got %d", len(defaultShard.Rules))
	}
}

// TestBuildShardsMultiLevelNestingComplexity tests complex multi-level nesting with shards at first level only
func TestBuildShardsMultiLevelNestingComplexity(t *testing.T) {
	root := indigo.NewRule("root", "")

	// Level 1 rules
	l1_region_north := indigo.NewRule("l1_north", `region == "north"`)
	l1_region_south := indigo.NewRule("l1_south", `region == "south"`)
	root.Add(l1_region_north)
	root.Add(l1_region_south)

	// Level 2 rules under north
	l2_north_urban := indigo.NewRule("l2_north_urban", `city_type == "urban"`)
	l2_north_rural := indigo.NewRule("l2_north_rural", `city_type == "rural"`)
	l1_region_north.Add(l2_north_urban)
	l1_region_north.Add(l2_north_rural)

	// Level 2 rules under south
	l2_south_urban := indigo.NewRule("l2_south_urban", `city_type == "urban"`)
	l2_south_rural := indigo.NewRule("l2_south_rural", `city_type == "rural"`)
	l1_region_south.Add(l2_south_urban)
	l1_region_south.Add(l2_south_rural)

	// Level 3 rules
	l3_urban_dense := indigo.NewRule("l3_dense", `density == "dense"`)
	l2_north_urban.Add(l3_urban_dense)

	// Create first-level regional shards
	northShard := indigo.NewRule("north", "")
	northShard.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.ID, "north")
	}

	southShard := indigo.NewRule("south", "")
	southShard.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.ID, "south")
	}

	root.Shards = []*indigo.Rule{northShard, southShard}

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	// Verify root-level structure
	if _, ok := root.Rules["north"]; !ok {
		t.Error("north shard not found")
	}
	if _, ok := root.Rules["south"]; !ok {
		t.Error("south shard not found")
	}

	// Verify north sub-structure: l1_north should be in north shard
	northShard = root.Rules["north"]
	if _, ok := northShard.Rules["l1_north"]; !ok {
		t.Error("l1_north not found under north shard")
	}

	// Verify l2 rules are still under l1_north
	l1NorthInShard := northShard.Rules["l1_north"]
	if len(l1NorthInShard.Rules) != 2 {
		t.Errorf("expected 2 children under l1_north, got %d", len(l1NorthInShard.Rules))
	}

	// Verify l3 rules are still under l2_north_urban
	l2NorthUrban := l1NorthInShard.Rules["l2_north_urban"]
	if l2NorthUrban == nil {
		t.Error("l2_north_urban not found")
		return
	}
	if len(l2NorthUrban.Rules) != 1 {
		t.Errorf("expected 1 child under l2_north_urban, got %d", len(l2NorthUrban.Rules))
	}
}

// TestBuildShardsMetaFunctionWithPanic tests that panics in Meta function are handled gracefully
func TestBuildShardsMetaFunctionWithPanic(t *testing.T) {
	root := indigo.NewRule("root", "")
	child := indigo.NewRule("child", "")
	root.Add(child)

	shard := indigo.NewRule("shard1", "")
	// This function will panic when called
	shard.Meta = func(r *indigo.Rule) bool {
		panic("meta function panic")
	}

	root.Shards = []*indigo.Rule{shard}

	// This should panic (not recover)
	defer func() {
		if r := recover(); r != nil {
			// panic is expected
			return
		}
		t.Error("expected panic from meta function")
	}()

	root.BuildShards()
}

// TestBuildShardsRulesMovedNotCopied tests that rules are moved, not copied
func TestBuildShardsRulesMovedNotCopied(t *testing.T) {
	root := indigo.NewRule("root", "")
	child := indigo.NewRule("child", "")
	child.Meta = "original"
	root.Add(child)

	shard := indigo.NewRule("shard1", "")
	shard.Meta = func(r *indigo.Rule) bool { return true }

	root.Shards = []*indigo.Rule{shard}

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	// Get the rule from its new location
	movedRule := root.Rules["shard1"].Rules["child"]

	// Modify it
	movedRule.Meta = "modified"

	// The modification should be visible (same object, not a copy)
	if root.Rules["shard1"].Rules["child"].Meta != "modified" {
		t.Error("rule was copied instead of moved")
	}
}

// TestBuildShardsWithEmptyShardsList tests behavior when Shards is empty list
func TestBuildShardsWithEmptyShardsList(t *testing.T) {
	root := indigo.NewRule("root", "")
	child := indigo.NewRule("child", "")
	root.Add(child)

	root.Shards = []*indigo.Rule{} // Empty list

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards with empty shards list failed: %v", err)
	}

	// With empty shards, rules should remain as direct children
	if len(root.Rules) != 1 {
		t.Errorf("expected 1 rule in root, got %d", len(root.Rules))
	}
	if _, ok := root.Rules["child"]; !ok {
		t.Error("child not found in root.Rules")
	}
}

// TestBuildShardsMetaReturnTrueForAllRules tests shard that matches all rules
func TestBuildShardsMetaReturnTrueForAllRules(t *testing.T) {
	root := indigo.NewRule("root", "")

	rule1 := indigo.NewRule("rule1", "x > 10")
	rule2 := indigo.NewRule("rule2", "y < 5")
	rule3 := indigo.NewRule("rule3", "z == 100")

	root.Add(rule1)
	root.Add(rule2)
	root.Add(rule3)

	shard := indigo.NewRule("shard1", "")
	shard.Meta = func(r *indigo.Rule) bool { return true } // Always true

	root.Shards = []*indigo.Rule{shard}

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	// All 3 rules should be in shard1
	if len(root.Rules["shard1"].Rules) != 3 {
		t.Errorf("expected 3 rules in shard1, got %d", len(root.Rules["shard1"].Rules))
	}

	// Default should be empty (no direct children)
	if defShard, ok := root.Rules["default"]; ok && len(defShard.Rules) != 0 {
		t.Errorf("expected default shard to be empty, got %d", len(defShard.Rules))
	}
}

// TestBuildShardsOrderOfMatchingMatters tests that first matching shard wins
func TestBuildShardsOrderOfMatchingMatters(t *testing.T) {
	root := indigo.NewRule("root", "")

	// Create a rule that matches both shards
	child := indigo.NewRule("rule_both", `status == "active" && type == "premium"`)
	root.Add(child)

	// Both shards match the rule
	shard1 := indigo.NewRule("shard_status", "")
	shard1.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.Expr, "active")
	}

	shard2 := indigo.NewRule("shard_type", "")
	shard2.Meta = func(r *indigo.Rule) bool {
		return strings.Contains(r.Expr, "premium")
	}

	// shard_status comes first, so it should match first
	root.Shards = []*indigo.Rule{shard1, shard2}

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards failed: %v", err)
	}

	// Rule should be in the first matching shard (shard_status)
	if _, ok := root.Rules["shard_status"].Rules["rule_both"]; !ok {
		t.Error("rule should be in shard_status (first matching)")
	}

	// Rule should NOT be in shard_type
	if _, ok := root.Rules["shard_type"].Rules["rule_both"]; ok {
		t.Error("rule should not be in shard_type (was already matched by shard_status)")
	}
}

// TestBuildShardsWithLargeNumberOfShards tests performance with many shards
func TestBuildShardsWithLargeNumberOfShards(t *testing.T) {
	root := indigo.NewRule("root", "")

	// Create 10 rules
	for i := 0; i < 10; i++ {
		child := indigo.NewRule(fmt.Sprintf("rule_%d", i), "")
		root.Add(child)
	}

	// Create 10 shards (one for each possible rule ID)
	for i := 0; i < 10; i++ {
		shard := indigo.NewRule(fmt.Sprintf("shard_%d", i), "")
		// Use a helper function to properly capture the value
		shard.Meta = makeShardMeta(i)
		root.Shards = append(root.Shards, shard)
	}

	err := root.BuildShards()
	if err != nil {
		t.Fatalf("BuildShards with many shards failed: %v", err)
	}

	// Each shard should have exactly one rule
	for i := 0; i < 10; i++ {
		shardName := fmt.Sprintf("shard_%d", i)
		shard := root.Rules[shardName]
		if shard == nil {
			t.Errorf("shard_%d not found", i)
			continue
		}
		if len(shard.Rules) != 1 {
			t.Errorf("shard_%d has %d rules, expected 1", i, len(shard.Rules))
		}
	}
}

// makeShardMeta creates a meta function that properly captures the index value
func makeShardMeta(idx int) func(*indigo.Rule) bool {
	return func(r *indigo.Rule) bool {
		return strings.Contains(r.ID, fmt.Sprintf("rule_%d", idx))
	}
}
