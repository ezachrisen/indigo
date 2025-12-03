package indigo_test

import (
	"context"
	"flag"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
)

// Set flag with go test -run=MyTest --debug=true
// to print verbose rule diagnostic info
var debugOutput bool

func init() {
	flag.BoolVar(&debugOutput, "debug", false, "Enable detailed logging for tests")
}

func debugLogf(t *testing.T, format string, args ...any) {
	t.Helper()
	if debugOutput {
		t.Logf(format, args...)
	}
}

func TestVault_NestedAdd(t *testing.T) {
	e, v := setup2(t)
	r := &indigo.Rule{
		ID:   "x1",
		Expr: `11 > 10`,
	}
	debugLogf(t, "Before add\n%s\n", v.Rule().Tree())
	t1 := time.Date(2020, 10, 10, 12, 0o0, 0o0, 0o0, time.UTC)
	if err := v.Mutate(indigo.LastUpdate(t1)); err != nil {
		t.Fatal(err)
	}
	t2 := time.Date(2022, 10, 10, 12, 0o0, 0o0, 0o0, time.UTC)
	if err := v.Mutate(indigo.Add(r, "c33"), indigo.LastUpdate(t2)); err != nil {
		t.Fatal(err)
	}
	debugLogf(t, "After add\n%s\n", v.Rule().Tree())
	lu := v.LastUpdate()
	if !lu.After(t1) {
		t.Fatal("time stamp was not updated")
	}
	res, err := e.Eval(context.Background(), v.Rule(), map[string]any{"value": 15})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pass {
		t.Error("expected pass")
	}
}

func TestVault_DeleteRule(t *testing.T) {
	_, v := setup(t)

	a := indigo.Rule{ID: "a", Rules: map[string]*indigo.Rule{
		"child": {ID: "child", Expr: `true`},
	}}

	err := v.Mutate(indigo.Add(&a, "root"))
	if err != nil {
		t.Fatal(err)
	}
	// snapshot snapshot
	snapshot := v.Rule()

	debugLogf(t, "Snapshot before\n%s\n", snapshot)
	// Delete child
	if err := v.Mutate(indigo.Delete("child")); err != nil {
		t.Fatal(err)
	}

	root := v.Rule()
	if root == nil {
		t.Fatal("root became nil")
	}
	child, _ := root.FindRule("child")
	if child != nil {
		t.Error("deleted child still present")
	}

	if snapshot.Expr != "" {
		t.Error("root expression was updated")
	}

	// make sure rule is still there in the snapshot
	child, _ = snapshot.FindRule("child")
	if child == nil {
		t.Error("child deleted from snapshot")
	}

	debugLogf(t, "Snapshot after: \n%s\n", snapshot)
	debugLogf(t, "Updated after: \n%s\n", root)
	// try to delete a rule that doesn't exist
	if err := v.Mutate(indigo.Delete("XXX")); err == nil {
		t.Fatal("wanted error")
	}
}

func TestVault_UpdateRule(t *testing.T) {
	e, v := setup(t)

	old := &indigo.Rule{ID: "rule1", Expr: `2+2 == 3`}
	if err := v.Mutate(indigo.Add(old, "root")); err != nil {
		t.Fatal(err)
	}
	snapshot := v.Rule()
	debugLogf(t, "Snapshot before\n%s\n", snapshot)

	res, err := e.Eval(context.Background(), v.Rule(), map[string]any{"x": 42})
	if err != nil {
		t.Fatal(err)
	}
	if res.Pass {
		t.Error("eval failed")
	}
	newRule := &indigo.Rule{ID: "rule1", Expr: `2 + 2 == 4`}
	if err := v.Mutate(indigo.Update(newRule)); err != nil {
		t.Fatal(err)
	}

	res, err = e.Eval(context.Background(), v.Rule(), map[string]any{"x": 42})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pass {
		t.Error("update did not take effect")
	}

	debugLogf(t, "After\n%s\n", v.Rule())
	if v.Rule().Rules["rule1"].Expr != `2 + 2 == 4` {
		t.Errorf("incorrect rule")
	}

	debugLogf(t, "Snapshot After\n%s\n", snapshot)
	if e := snapshot.Rules["rule1"].Expr; e != `2+2 == 3` {
		t.Errorf("snapshot was updated: %s", e)
	}

	// Update the root (clearing the vault)
	newRoot := &indigo.Rule{ID: "root", Expr: ` 1 == 1 `}
	if err := v.Mutate(indigo.Update(newRoot)); err != nil {
		t.Fatal(err)
	}
	if e := snapshot.Rules["rule1"].Expr; e != `2+2 == 3` {
		t.Errorf("snapshot was updated: %s", e)
	}
	after := v.Rule()
	if len(after.Rules) > 0 || after.Expr != ` 1 == 1 ` {
		t.Errorf("root was not replaced")
	}
}

func TestVault_AddRule(t *testing.T) {
	e, v := setup(t)

	old := &indigo.Rule{ID: "rule1", Expr: `2+2 == 4`}
	if err := v.Mutate(indigo.Add(old, "root")); err != nil {
		t.Fatal(err)
	}

	res, err := e.Eval(context.Background(), v.Rule(), map[string]any{"x": 42})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pass {
		t.Error("eval failed")
	}

	debugLogf(t, "Before adding rule2 to root:\n%s\n", v.Rule())
	newRule := &indigo.Rule{ID: "rule2", Expr: `10<1`}
	if err := v.Mutate(indigo.Add(newRule, "root")); err != nil {
		t.Fatal(err)
	}
	snapshot := v.Rule()
	debugLogf(t, "Snapshot before adding rule1.2 to rule1:\n%s\n", snapshot)
	// fmt.Println("-------------------- ")
	newRule2 := &indigo.Rule{ID: "rule1.2", Expr: `10<1`}
	newRule3 := &indigo.Rule{ID: "rule1.3", Expr: `10<1`}
	if err := v.Mutate(indigo.Add(newRule2, "rule1"), indigo.Add(newRule3, "rule1")); err != nil {
		t.Fatal(err)
	}

	debugLogf(t, "After adding rule1.2:\n%s\n", v.Rule())
	_, err = e.Eval(context.Background(), v.Rule(), map[string]any{"x": 42})
	if err != nil {
		t.Fatal(err)
	}

	if v.Rule().Rules["rule1"].Expr != `2+2 == 4` {
		t.Errorf("incorrect rule")
	}
	if v.Rule().Rules["rule2"].Expr != `10<1` {
		t.Errorf("incorrect rule")
	}
	debugLogf(t, "Snapshot after adds:\n%s", snapshot)
	if len(snapshot.Rules["rule1"].Rules) > 0 {
		t.Errorf("snapshot modified")
	}
}

func TestVault_MoveRule(t *testing.T) {
	_, v := setup(t)

	one := &indigo.Rule{ID: "rule1", Expr: `2+2 == 4`}
	b := &indigo.Rule{ID: "b", Expr: `10 > 1`}
	one.Rules = map[string]*indigo.Rule{}
	one.Rules["b"] = b

	two := &indigo.Rule{ID: "rule2", Expr: `1+1 == 2`}
	if err := v.Mutate(indigo.Add(one, "root"), indigo.Add(two, "root")); err != nil {
		t.Fatal(err)
	}
	debugLogf(t, "Before\n%s\n", v.Rule())

	if err := v.Mutate(indigo.Move("b", "rule2")); err != nil {
		t.Fatal(err)
	}

	debugLogf(t, "After\n%s\n", v.Rule())
}

func TestVault_Concurrency(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	schema := indigo.Schema{
		ID: "xx",
		Elements: []indigo.DataElement{
			{Name: "x", Type: indigo.Int{}},
		},
	}
	eng := indigo.NewEngine(cel.NewEvaluator(cel.FixedSchema(&schema)))
	v, err := indigo.NewVault(eng, threeRules())
	if err != nil {
		t.Fatal(err)
	}
	desiredResult := atomic.Bool{}
	desiredResult.Store(true)
	var wg sync.WaitGroup

	// this goroutine will evaluate the rule in the vault continuously
	wg.Go(func() {
		ctx := context.Background()
		errCount := 0
		for i := range 3000 {
			rule := v.Rule()

			debugLogf(t, "Evaluating: \n%s\n", rule)
			res, err := eng.Eval(ctx, rule, map[string]any{})
			if err != nil {
				panic(err)
			}
			expected := desiredResult.Load()
			debugLogf(t, "Result: \n%s\n", res)
			if res.Pass != expected {
				errCount++
				if errCount > 3 {
					t.Errorf("expected %t, got %t, iteration %d", expected, res.Pass, i)
				}
				if errCount == 3 {
					errCount = 1
				}
			}
			// we sleep between iterations to ensure that the
			// rule updates can be observed
			time.Sleep(10 * time.Millisecond)
		}
	})

	wg.Go(func() {
		for i := range 10000 {
			var r *indigo.Rule
			var newResult bool
			if i%2 == 0 {
				r = indigo.NewRule("b2", " 1 != 1 ")
				newResult = false
			} else {
				r = indigo.NewRule("b2", " 1 == 1 ")
				newResult = true

			}
			if err := v.Mutate(indigo.Update(r)); err != nil {
				t.Fatal(err)
			}

			// Move c1 back and forth between a and c
			cr := v.Rule()
			if _, ok := cr.Rules["c"].Rules["c1"]; ok {
				if err := v.Mutate(indigo.Move("c1", "a")); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := v.Mutate(indigo.Move("c1", "c")); err != nil {
					t.Fatal(err)
				}
			}

			debugLogf(t, "After update: \n%s\n", v.Rule())
			desiredResult.Store(newResult)
			time.Sleep(3 * time.Millisecond)
		}
	})

	wg.Wait()
}

func threeRules() *indigo.Rule {
	r := indigo.NewRule("root", "")
	a := indigo.NewRule("a", " 1 == 1")
	a1 := indigo.NewRule("a1", " 1 == 1")
	a2 := indigo.NewRule("a2", " 1 == 1")
	a3 := indigo.NewRule("a3", " 1 == 1")

	a.Rules["a1"] = a1
	a.Rules["a2"] = a2
	a.Rules["a3"] = a3

	b := indigo.NewRule("b", " 1 == 1")
	b1 := indigo.NewRule("b1", " 1 == 1")
	b2 := indigo.NewRule("b2", " 1 == 1")
	b3 := indigo.NewRule("b3", " 1 == 1")

	b.Rules["b1"] = b1
	b.Rules["b2"] = b2
	b.Rules["b3"] = b3

	c := indigo.NewRule("c", " 1 == 1")
	c1 := indigo.NewRule("c1", " 1 == 1")
	c2 := indigo.NewRule("c2", " 1 == 1")
	c3 := indigo.NewRule("c3", " 1 == 1")

	c.Rules["c1"] = c1
	c.Rules["c2"] = c2
	c.Rules["c3"] = c3

	r.Rules["a"] = a
	r.Rules["b"] = b
	r.Rules["c"] = c
	return r
}

func TestVault_AddInvalidInputs(t *testing.T) {
	_, v := setup(t)

	// Add with empty ID
	r := indigo.Rule{ID: "", Expr: "true"}
	err := v.Mutate(indigo.Add(&r, "root"))
	if err == nil {
		t.Error("expected error for empty ID")
	}

	// Add to non-existent parent
	r2 := indigo.Rule{ID: "test", Expr: "true"}
	err = v.Mutate(indigo.Add(&r2, "nonexistent"))
	if err == nil {
		t.Error("expected error for non-existent parent")
	}

	// Add duplicate ID
	r3 := indigo.Rule{ID: "dup", Expr: "true"}
	err = v.Mutate(indigo.Add(&r3, "root"))
	if err != nil {
		t.Fatal(err)
	}
	r4 := indigo.Rule{ID: "dup", Expr: "false"}
	err = v.Mutate(indigo.Add(&r4, "root"))
	if err == nil {
		t.Error("expected error for duplicate ID")
	}
}

func TestVault_UpdateInvalid(t *testing.T) {
	_, v := setup(t)

	// Update non-existent
	r := indigo.Rule{ID: "nonexistent", Expr: "true"}
	err := v.Mutate(indigo.Update(&r))
	if err == nil {
		t.Error("expected error for updating non-existent rule")
	}
}

func TestVault_MoveInvalid(t *testing.T) {
	_, v := setup(t)

	// Setup some rules
	a := indigo.Rule{ID: "a", Expr: "true", Rules: map[string]*indigo.Rule{
		"child": {ID: "child", Expr: "true"},
	}}
	err := v.Mutate(indigo.Add(&a, "root"))
	if err != nil {
		t.Fatal(err)
	}

	// Move to non-existent parent
	err = v.Mutate(indigo.Move("a", "nonexistent"))
	if err == nil {
		t.Error("expected error for moving to non-existent parent")
	}

	// Move non-existent rule
	err = v.Mutate(indigo.Move("nonexistent", "root"))
	if err == nil {
		t.Error("expected error for moving non-existent rule")
	}

	// Move to self
	err = v.Mutate(indigo.Move("a", "a"))
	if err == nil {
		t.Error("expected error for moving to self")
	}

	// Move to descendant
	err = v.Mutate(indigo.Move("a", "child"))
	if err == nil {
		t.Error("expected error for moving to descendant")
	}
}

func TestVault_DeleteInvalid(t *testing.T) {
	_, v := setup(t)

	// Delete root
	err := v.Mutate(indigo.Delete("root"))
	if err == nil {
		t.Error("expected error for deleting root")
	}
}

func TestVault_CompilationError(t *testing.T) {
	_, v := setup(t)

	// Add rule with invalid expression
	r := indigo.Rule{ID: "invalid", Expr: "invalid syntax {{{{"}
	err := v.Mutate(indigo.Add(&r, "root"))
	if err == nil {
		t.Error("expected compilation error")
	}

	// Update to invalid
	valid := indigo.Rule{ID: "valid", Expr: "true"}
	err = v.Mutate(indigo.Add(&valid, "root"))
	if err != nil {
		t.Fatal(err)
	}
	invalidUpdate := indigo.Rule{ID: "valid", Expr: "invalid {{{{"}
	err = v.Mutate(indigo.Update(&invalidUpdate))
	if err == nil {
		t.Error("expected compilation error on update")
	}
}

func TestVault_MultipleMutationsWithError(t *testing.T) {
	_, v := setup(t)

	r1 := indigo.Rule{ID: "r1", Expr: "true"}
	r2 := indigo.Rule{ID: "r2", Expr: "true"}
	invalid := indigo.Rule{ID: "", Expr: "true"} // empty ID

	err := v.Mutate(indigo.Add(&r1, "root"), indigo.Add(&invalid, "root"), indigo.Add(&r2, "root"))
	if err == nil {
		t.Error("expected error in multiple mutations")
	}

	// Check that valid mutations before error were NOT applied
	root := v.Rule()
	if _, ok := root.Rules["r1"]; ok {
		t.Error("r1 was added despite error")
	}
}

// Helper to create a fresh engine + vault for each test
func setup2(t *testing.T) (indigo.Engine, *indigo.Vault) {
	eng := indigo.NewEngine(cel.NewEvaluator())
	r := threeRules()
	c31 := indigo.NewRule("c31", " 1 == 1")
	c32 := indigo.NewRule("c32", " 1 == 1")
	c33 := indigo.NewRule("c33", " 1 == 1")

	c3 := r.Rules["c"].Rules["c3"]
	c3.Rules[c31.ID] = c31
	c3.Rules[c32.ID] = c32
	c3.Rules[c33.ID] = c33

	v, err := indigo.NewVault(eng, r)
	if err != nil {
		t.Fatal(err)
	}
	return eng, v
}

// Helper to create a fresh engine + vault for each test
func setup(t *testing.T) (indigo.Engine, *indigo.Vault) {
	eng := indigo.NewEngine(cel.NewEvaluator())
	v, err := indigo.NewVault(eng, nil)
	if err != nil {
		t.Fatal(err)
	}
	return eng, v
}

func TestVault_ConcurrentMutations_RaceCondition(t *testing.T) {
	// Setup
	eng := indigo.NewEngine(cel.NewEvaluator())
	v, err := indigo.NewVault(eng, indigo.NewRule("root", ""))
	if err != nil {
		t.Fatal(err)
	}

	// Number of concurrent writers
	numWriters := 50
	var wg sync.WaitGroup
	wg.Add(numWriters)

	// Each writer adds a unique rule
	for i := 0; i < numWriters; i++ {
		go func(id int) {
			defer wg.Done()
			ruleID := fmt.Sprintf("rule_%d", id)
			newRule := indigo.NewRule(ruleID, "true")

			if err := v.Mutate(indigo.Add(newRule, "root")); err != nil {
				t.Errorf("Mutate failed for %s: %v", ruleID, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify
	finalRule := v.Rule()
	if len(finalRule.Rules) != numWriters {
		t.Errorf("Race condition detected! Expected %d rules, got %d", numWriters, len(finalRule.Rules))
	}
}

// The Make Safe Path test verifies that mutations on the Vault
// are not seen outside the vault by someone who grabbed to rule before
// mutations happened.
//
// The test operates on this rule tree:
//
// ┌────────────────────────────────────────────────────────┐
// │                                                        │
// │ INDIGO RULES                                           │
// │                                                        │
// ├─────────────────┬────────┬────────────┬────────┬───────┤
// │                 │        │            │ Result │       │
// │ Rule            │ Schema │ Expression │ Type   │ Meta  │
// ├─────────────────┼────────┼────────────┼────────┼───────┤
// │ root            │        │            │ <nil>  │ <nil> │
// │   childA        │        │            │ <nil>  │ <nil> │
// │     grandChildB │        │            │ <nil>  │ <nil> │
// └─────────────────┴────────┴────────────┴────────┴───────┘
func TestVault_MakeSafePath(t *testing.T) {
	// Setup
	originalRoot := indigo.NewRule("root", "")
	originalChildA := indigo.NewRule("childA", "")
	originalGrandChildB := indigo.NewRule("grandChildB", "")
	originalChildA.Rules[originalGrandChildB.ID] = originalGrandChildB
	originalRoot.Rules[originalChildA.ID] = originalChildA
	eng := indigo.NewEngine(cel.NewEvaluator())
	v, err := indigo.NewVault(eng, originalRoot)
	if err != nil {
		t.Fatal(err)
	}

	//----------------------------------------
	// We'll preserve the baseline so we can compare to it
	// after we mutate the vault
	baseline := v.Rule()
	debugLogf(t, "baseline:\n%s\n", baseline)

	if baseline.Rules["childA"] != originalChildA {
		t.Error("childA unexpected poninter value")
	}

	if baseline.Rules["childA"].Rules["grandChildB"] != originalGrandChildB {
		t.Error("grandChildB unexpected poninter value")
	}

	// Now, let's try to update grandChildB with a new rule.
	// This will trigger makeSafePath for the path root -> childA -> grandChildB
	newGrandChildB := indigo.NewRule("grandChildB", "true")
	err = v.Mutate(indigo.Update(newGrandChildB))
	if err != nil {
		// If makeSafePath fails to copy correctly, this Mutate call might return an error
		// like "parent not found for rule after cloning", which would indicate the bug.
		t.Fatalf("Mutating deep rule failed: %v", err)
	}

	//----------------------------------------
	// Verify the new state
	after := v.Rule()

	debugLogf(t, "baseline after grandChildB update:\n%s\n", baseline)
	debugLogf(t, "after grandChildB update:\n%s\n", after)

	// Check if the original tree was untouched (immutable copy-on-write)
	if baseline.Rules["childA"] != originalChildA {
		t.Logf("inVault: %p, original = %p\n", baseline.Rules["childA"], &originalChildA)
		t.Error("Original root's childA should not be modified")
	}
	if originalChildA.Rules["grandChildB"] != originalGrandChildB {
		t.Error("Original childA's grandChildB should not be modified")
	}

	// Verify that the updated grandChildB is in the new tree
	foundGrandChildB, _ := after.FindRule("grandChildB")
	if foundGrandChildB == nil {
		t.Fatal("grandChildB not found in the final rule tree after update")
	}
	if foundGrandChildB.Expr != "true" {
		t.Errorf("grandChildB expr not updated correctly, expected 'true', got '%s'", foundGrandChildB.Expr)
	}
	if foundGrandChildB == newGrandChildB {
		// This is good, it means the new rule was inserted.
		// The key is that its ancestors in the new tree are *copies*, not the originals.
	} else {
		t.Error("foundGrandChildB should be the newGrandChildB instance")
	}

	// Check that the path to grandChildB consists of new copies
	// Parent of grandChildB should be a new instance of childA
	parentOfGrandChildB := after.FindParent("grandChildB")
	if parentOfGrandChildB == nil {
		t.Fatal("Parent of grandChildB not found in final tree")
	}
	if parentOfGrandChildB == originalChildA {
		t.Error("Parent of grandChildB should be a copy, not the original childA")
	}
	if parentOfGrandChildB.ID != "childA" {
		t.Errorf("Parent ID mismatch, expected 'childA', got '%s'", parentOfGrandChildB.ID)
	}

	// Parent of childA (in the new tree) should be a new instance of root
	parentOfChildA := after.FindParent("childA")
	if parentOfChildA == nil {
		t.Fatal("Parent of childA not found in final tree")
	}
	if parentOfChildA == originalRoot {
		t.Error("Parent of childA should be a copy, not the original root")
	}
	if parentOfChildA.ID != "root" {
		t.Errorf("Parent ID mismatch, expected 'root', got '%s'", parentOfChildA.ID)
	}
}
