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

//	func TestVault_DeleteRule(t *testing.T) {
//		_, v := setup(t)
//
//		a := indigo.Rule{ID: "a", Rules: map[string]*indigo.Rule{
//			"child": {ID: "child", Expr: `true`},
//		}}
//
//		err := v.Mutate(indigo.Add(&a, "root"))
//		if err != nil {
//			t.Fatal(err)
//		}
//		// snapshot
//		snapshot := v.ImmutableRule()
//
//		debugLogf(t, "Snapshot before\n%s\n", snapshot)
//		// Delete child
//		if err := v.Mutate(indigo.Delete("child")); err != nil {
//			t.Fatal(err)
//		}
//
//		root := v.ImmutableRule()
//		if root == nil {
//			t.Fatal("root became nil")
//		}
//		child, _ := root.FindRule("child")
//		if child != nil {
//			t.Error("deleted child still present")
//		}
//
//		if snapshot.Expr != "" {
//			t.Error("root expression was updated")
//		}
//
//		// make sure rule is still there in the snapshot
//		child, _ = snapshot.FindRule("child")
//		if child == nil {
//			t.Error("child deleted from snapshot")
//		}
//
//		debugLogf(t, "Snapshot after: \n%s\n", snapshot)
//		debugLogf(t, "Updated after: \n%s\n", root)
//		// try to delete a rule that doesn't exist
//		if err := v.Mutate(indigo.Delete("XXX")); err == nil {
//			t.Fatal("wanted error")
//		}
//	}
//
//	func TestVault_UpdateRule(t *testing.T) {
//		e, v := setup(t)
//
//		old := &indigo.Rule{ID: "rule1", Expr: `2+2 == 3`}
//		if err := v.Mutate(indigo.Add(old, "root")); err != nil {
//			t.Fatal(err)
//		}
//		snapshot := v.ImmutableRule()
//		debugLogf(t, "Snapshot before\n%s\n", snapshot)
//
//		res, err := e.Eval(context.Background(), v.ImmutableRule(), map[string]any{"x": 42})
//		if err != nil {
//			t.Fatal(err)
//		}
//		if res.Pass {
//			t.Error("eval failed")
//		}
//		newRule := &indigo.Rule{ID: "rule1", Expr: `2 + 2 == 4`}
//		if err := v.Mutate(indigo.Update(newRule)); err != nil {
//			t.Fatal(err)
//		}
//
//		res, err = e.Eval(context.Background(), v.ImmutableRule(), map[string]any{"x": 42})
//		if err != nil {
//			t.Fatal(err)
//		}
//		if !res.Pass {
//			t.Error("update did not take effect")
//		}
//
//		debugLogf(t, "After\n%s\n", v.ImmutableRule())
//		if v.ImmutableRule().Rules["rule1"].Expr != `2 + 2 == 4` {
//			t.Errorf("incorrect rule")
//		}
//
//		debugLogf(t, "Snapshot After\n%s\n", snapshot)
//		if e := snapshot.Rules["rule1"].Expr; e != `2+2 == 3` {
//			t.Errorf("snapshot was updated: %s", e)
//		}
//
//		// Update the root (clearing the vault)
//		newRoot := &indigo.Rule{ID: "root", Expr: ` 1 == 1 `}
//		if err := v.Mutate(indigo.Update(newRoot)); err != nil {
//			t.Fatal(err)
//		}
//		if e := snapshot.Rules["rule1"].Expr; e != `2+2 == 3` {
//			t.Errorf("snapshot was updated: %s", e)
//		}
//		after := v.ImmutableRule()
//		if len(after.Rules) > 0 || after.Expr != ` 1 == 1 ` {
//			t.Errorf("root was not replaced")
//		}
//	}
//
//	func TestVault_AddRule(t *testing.T) {
//		e, v := setup(t)
//
//		old := &indigo.Rule{ID: "rule1", Expr: `2+2 == 4`}
//		if err := v.Mutate(indigo.Add(old, "root")); err != nil {
//			t.Fatal(err)
//		}
//
//		res, err := e.Eval(context.Background(), v.ImmutableRule(), map[string]any{"x": 42})
//		if err != nil {
//			t.Fatal(err)
//		}
//		if !res.Pass {
//			t.Error("eval failed")
//		}
//
//		debugLogf(t, "Before adding rule2 to root:\n%s\n", v.ImmutableRule())
//		newRule := &indigo.Rule{ID: "rule2", Expr: `10<1`}
//		if err := v.Mutate(indigo.Add(newRule, "root")); err != nil {
//			t.Fatal(err)
//		}
//		snapshot := v.ImmutableRule()
//		debugLogf(t, "Snapshot before adding rule1.2 to rule1:\n%s\n", snapshot)
//		newRule2 := &indigo.Rule{ID: "rule1.2", Expr: `10<1`}
//		newRule3 := &indigo.Rule{ID: "rule1.3", Expr: `10<1`}
//		if err := v.Mutate(indigo.Add(newRule2, "rule1"), indigo.Add(newRule3, "rule1")); err != nil {
//			t.Fatal(err)
//		}
//
//		debugLogf(t, "After adding rule1.2:\n%s\n", v.ImmutableRule())
//		_, err = e.Eval(context.Background(), v.ImmutableRule(), map[string]any{"x": 42})
//		if err != nil {
//			t.Fatal(err)
//		}
//
//		if v.ImmutableRule().Rules["rule1"].Expr != `2+2 == 4` {
//			t.Errorf("incorrect rule")
//		}
//		if v.ImmutableRule().Rules["rule2"].Expr != `10<1` {
//			t.Errorf("incorrect rule")
//		}
//		debugLogf(t, "Snapshot after adds:\n%s", snapshot)
//		if len(snapshot.Rules["rule1"].Rules) > 0 {
//			t.Errorf("snapshot modified")
//		}
//	}
//
//	func TestVault_MoveRule(t *testing.T) {
//		_, v := setup(t)
//
//		one := &indigo.Rule{ID: "rule1", Expr: `2+2 == 4`}
//		b := &indigo.Rule{ID: "b", Expr: `10 > 1`}
//		one.Rules = map[string]*indigo.Rule{}
//		one.Rules["b"] = b
//
//		two := &indigo.Rule{ID: "rule2", Expr: `1+1 == 2`}
//		if err := v.Mutate(indigo.Add(one, "root"), indigo.Add(two, "root")); err != nil {
//			t.Fatal(err)
//		}
//		baseline := v.ImmutableRule()
//		debugLogf(t, "Before\n%s\n", baseline)
//
//		if err := v.Mutate(indigo.Move("b", "rule2")); err != nil {
//			t.Fatal(err)
//		}
//
//		debugLogf(t, "Baseline after (should not change)\n%s\n", baseline)
//		debugLogf(t, "After\n%s\n", v.ImmutableRule())
//	}
//
// This tests adding a rule to a parent 3 levels deep
func TestVault_NestedAdd(t *testing.T) {
	e, v := setup2(t)
	r := &indigo.Rule{
		ID:   "x1",
		Expr: `11 > 10`,
	}
	e.Compile(r)
	debugLogf(t, "Before add\n%s\n", v.ImmutableRule().Tree())
	t1 := time.Date(2020, 10, 10, 12, 0o0, 0o0, 0o0, time.UTC)
	if err := v.Mutate(indigo.LastUpdate(t1)); err != nil {
		t.Fatal(err)
	}
	t2 := time.Date(2022, 10, 10, 12, 0o0, 0o0, 0o0, time.UTC)
	if err := v.Mutate(indigo.Add(r, "c33"), indigo.LastUpdate(t2)); err != nil {
		t.Fatal(err)
	}
	debugLogf(t, "After add\n%s\n", v.ImmutableRule().Tree())
	lu := v.LastUpdate()
	if !lu.After(t1) {
		t.Fatal("time stamp was not updated")
	}
	res, err := e.Eval(context.Background(), v.ImmutableRule(), map[string]any{"value": 15})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pass {
		t.Error("expected pass")
	}
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
	r := threeRules()

	err := eng.Compile(r)
	if err != nil {
		t.Fatal(err)
	}
	v, err := indigo.NewVault(r)
	if err != nil {
		t.Fatal(err)
	}
	desiredResult := atomic.Bool{}
	desiredResult.Store(true)
	var wg sync.WaitGroup
	rule := v.ImmutableRule()
	_, err = eng.Eval(context.Background(), rule, map[string]any{})
	if err != nil {
		panic(err)
	}

	// this goroutine will evaluate the rule in the vault continuously
	wg.Go(func() {
		ctx := context.Background()
		errCount := 0
		for i := range 3000 {
			rule := v.ImmutableRule()

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
				r = indigo.NewRule("b2", " false ")
				err = eng.Compile(r)
				if err != nil {
					panic(err)
				}
				newResult = false
			} else {
				r = indigo.NewRule("b2", " true ")
				newResult = true
				err = eng.Compile(r)
				if err != nil {
					panic(err)
				}

			}
			if err := v.Mutate(indigo.Update(r)); err != nil {
				t.Fatal(err)
			}

			// Move c1 back and forth between a and c
			cr := v.ImmutableRule()
			if _, ok := cr.Rules["c"].Rules["c1"]; ok {
				if err := v.Mutate(indigo.Move("c1", "a")); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := v.Mutate(indigo.Move("c1", "c")); err != nil {
					t.Fatal(err)
				}
			}

			debugLogf(t, "After update: \n%s\n", v.ImmutableRule())
			desiredResult.Store(newResult)
			time.Sleep(3 * time.Millisecond)
		}
	})

	wg.Wait()
}

func threeRules() *indigo.Rule {
	r := indigo.NewRule("root", "")
	a := indigo.NewRule("a", " true")
	a1 := indigo.NewRule("a1", " true")
	a2 := indigo.NewRule("a2", " true")
	a3 := indigo.NewRule("a3", " true")

	a.Rules["a1"] = a1
	a.Rules["a2"] = a2
	a.Rules["a3"] = a3

	b := indigo.NewRule("b", " true")
	b1 := indigo.NewRule("b1", " true")
	b2 := indigo.NewRule("b2", " true")
	b3 := indigo.NewRule("b3", " true")

	b.Rules["b1"] = b1
	b.Rules["b2"] = b2
	b.Rules["b3"] = b3

	c := indigo.NewRule("c", " true")
	c1 := indigo.NewRule("c1", " true")
	c2 := indigo.NewRule("c2", " true")
	c3 := indigo.NewRule("c3", " true")

	c.Rules["c1"] = c1
	c.Rules["c2"] = c2
	c.Rules["c3"] = c3

	r.Rules["a"] = a
	r.Rules["b"] = b
	r.Rules["c"] = c
	return r
}

func TestVault_AddInvalidInputs(t *testing.T) {
	v := setup(t)

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
	v := setup(t)

	// Update non-existent
	r := indigo.Rule{ID: "nonexistent", Expr: "true"}
	err := v.Mutate(indigo.Update(&r))
	if err == nil {
		t.Error("expected error for updating non-existent rule")
	}
}

func TestVault_MoveInvalid(t *testing.T) {
	v := setup(t)

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
	v := setup(t)

	// Delete root
	err := v.Mutate(indigo.Delete("root"))
	if err == nil {
		t.Error("expected error for deleting root")
	}
}

func TestVault_MultipleMutationsWithError(t *testing.T) {
	v := setup(t)

	r1 := indigo.Rule{ID: "r1", Expr: "true"}
	r2 := indigo.Rule{ID: "r2", Expr: "true"}
	invalid := indigo.Rule{ID: "", Expr: "true"} // empty ID

	err := v.Mutate(indigo.Add(&r1, "root"), indigo.Add(&invalid, "root"), indigo.Add(&r2, "root"))
	if err == nil {
		t.Error("expected error in multiple mutations")
	}

	// Check that valid mutations before error were NOT applied
	root := v.ImmutableRule()
	if _, ok := root.Rules["r1"]; ok {
		t.Error("r1 was added despite error")
	}
}

// Helper to create a fresh engine + vault for each test
func setup2(t *testing.T) (indigo.Engine, *indigo.Vault) {
	eng := indigo.NewEngine(cel.NewEvaluator())
	r := threeRules()
	c31 := indigo.NewRule("c31", " true")
	c32 := indigo.NewRule("c32", " true")
	c33 := indigo.NewRule("c33", " true")

	c3 := r.Rules["c"].Rules["c3"]
	c3.Rules[c31.ID] = c31
	c3.Rules[c32.ID] = c32
	c3.Rules[c33.ID] = c33
	err := eng.Compile(r)
	if err != nil {
		t.Fatal(err)
	}
	v, err := indigo.NewVault(r)
	if err != nil {
		t.Fatal(err)
	}
	return eng, v
}

// Helper to create a fresh engine + vault for each test
func setup(t *testing.T) *indigo.Vault {
	v, err := indigo.NewVault(indigo.NewRule("root", ""))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestVault_ConcurrentMutations_RaceCondition(t *testing.T) {
	// Setup
	v, err := indigo.NewVault(indigo.NewRule("root", ""))
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
	finalRule := v.ImmutableRule()
	if len(finalRule.Rules) != numWriters {
		t.Errorf("Race condition detected! Expected %d rules, got %d", numWriters, len(finalRule.Rules))
	}
}

// This test mutates a rule in a Vault and makes sure that the changes
// are not seen outside the vault by someone who grabbed to rule before
// mutations happened.
func TestVault_Mutations(t *testing.T) {
	setup := func() *indigo.Vault {
		root := indigo.NewRule("root", "")
		root.Add(indigo.NewRule("childA", ""))
		root.Add(indigo.NewRule("childB", ""))
		root.Add(indigo.NewRule("childC", ""))
		root.Rules["childA"].Add(indigo.NewRule("one", ""))
		root.Rules["childA"].Add(indigo.NewRule("two", ""))
		root.Rules["childB"].Add(indigo.NewRule("three", ""))
		v, _ := indigo.NewVault(root)
		return v
	}

	// this is the baseline rule all mutations happen against
	baselineStr := `
┌──────────────────────────────────────────────────┐
│                                                  │
│ INDIGO RULES                                     │
│                                                  │
├───────────┬────────┬────────────┬────────┬───────┤
│           │        │            │ Result │       │
│ Rule      │ Schema │ Expression │ Type   │ Meta  │
├───────────┼────────┼────────────┼────────┼───────┤
│ root      │        │            │ <nil>  │ <nil> │
│   childA  │        │            │ <nil>  │ <nil> │
│     one   │        │            │ <nil>  │ <nil> │
│     two   │        │            │ <nil>  │ <nil> │
│   childB  │        │            │ <nil>  │ <nil> │
│     three │        │            │ <nil>  │ <nil> │
│   childC  │        │            │ <nil>  │ <nil> │
└───────────┴────────┴────────────┴────────┴───────┘       
`

	t.Run("delete", func(t *testing.T) {
		v := setup()
		baseline := v.ImmutableRule()
		debugLogf(t, "baseline:\n%s\n", baseline)

		err := v.Mutate(indigo.Delete("three"))
		if err != nil {
			t.Fatalf("Updating rule failed: %v", err)
		}

		after := v.ImmutableRule()

		debugLogf(t, "baseline after mutation (should not change):\n%s\n", baseline)
		debugLogf(t, "after mutation:\n%s\n", after)
		want := `
┌─────────────────────────────────────────────────┐
│                                                 │
│ INDIGO RULES                                    │
│                                                 │
├──────────┬────────┬────────────┬────────┬───────┤
│          │        │            │ Result │       │
│ Rule     │ Schema │ Expression │ Type   │ Meta  │
├──────────┼────────┼────────────┼────────┼───────┤
│ root     │        │            │ <nil>  │ <nil> │
│   childA │        │            │ <nil>  │ <nil> │
│     one  │        │            │ <nil>  │ <nil> │
│     two  │        │            │ <nil>  │ <nil> │
│   childB │        │            │ <nil>  │ <nil> │
│   childC │        │            │ <nil>  │ <nil> │
└──────────┴────────┴────────────┴────────┴───────┘
`
		assertEqual(want, after.String(), t)
		assertEqual(baselineStr, baseline.String(), t)
		assertPointersDifferent(t, baseline, after, "root", "childB")
		assertPointersEqual(t, baseline, after, "childC", "childA", "one", "two")
	})

	t.Run("update", func(t *testing.T) {
		v := setup()
		baseline := v.ImmutableRule()
		debugLogf(t, "baseline:\n%s\n", baseline)

		err := v.Mutate(indigo.Update(indigo.NewRule("one", "true")))
		if err != nil {
			t.Fatalf("Updating rule failed: %v", err)
		}

		after := v.ImmutableRule()

		debugLogf(t, "baseline after mutation (should not change):\n%s\n", baseline)
		debugLogf(t, "after mutation:\n%s\n", after)
		want := `
┌──────────────────────────────────────────────────┐
│                                                  │
│ INDIGO RULES                                     │
│                                                  │
├───────────┬────────┬────────────┬────────┬───────┤
│           │        │            │ Result │       │
│ Rule      │ Schema │ Expression │ Type   │ Meta  │
├───────────┼────────┼────────────┼────────┼───────┤
│ root      │        │            │ <nil>  │ <nil> │
│   childA  │        │            │ <nil>  │ <nil> │
│     one   │        │ true       │ <nil>  │ <nil> │
│     two   │        │            │ <nil>  │ <nil> │
│   childB  │        │            │ <nil>  │ <nil> │
│     three │        │            │ <nil>  │ <nil> │
│   childC  │        │            │ <nil>  │ <nil> │
└───────────┴────────┴────────────┴────────┴───────┘
`
		assertEqual(want, after.String(), t)
		assertEqual(baselineStr, baseline.String(), t)
		assertPointersDifferent(t, baseline, after, "root", "childA", "one")
		assertPointersEqual(t, baseline, after, "two", "childB", "three", "childC")
	})

	t.Run("add", func(t *testing.T) {
		v := setup()
		baseline := v.ImmutableRule()
		debugLogf(t, "baseline:\n%s\n", baseline)

		err := v.Mutate(indigo.Add(indigo.NewRule("XXX", "false"), "childA"))
		if err != nil {
			t.Fatalf("adding rule failed: %v", err)
		}

		after := v.ImmutableRule()
		debugLogf(t, "baseline after mutation (should not change):\n%s\n", baseline)
		debugLogf(t, "after mutation:\n%s\n", after)
		want := `
┌──────────────────────────────────────────────────┐
│                                                  │
│ INDIGO RULES                                     │
│                                                  │
├───────────┬────────┬────────────┬────────┬───────┤
│           │        │            │ Result │       │
│ Rule      │ Schema │ Expression │ Type   │ Meta  │
├───────────┼────────┼────────────┼────────┼───────┤
│ root      │        │            │ <nil>  │ <nil> │
│   childA  │        │            │ <nil>  │ <nil> │
│     XXX   │        │ false      │ <nil>  │ <nil> │
│     one   │        │            │ <nil>  │ <nil> │
│     two   │        │            │ <nil>  │ <nil> │
│   childB  │        │            │ <nil>  │ <nil> │
│     three │        │            │ <nil>  │ <nil> │
│   childC  │        │            │ <nil>  │ <nil> │
└───────────┴────────┴────────────┴────────┴───────┘
		`
		assertEqual(want, after.String(), t)
		assertEqual(baselineStr, baseline.String(), t)
		assertPointersDifferent(t, baseline, after, "root", "childA")
		assertPointersEqual(t, baseline, after, "childB")
	})

	t.Run("move", func(t *testing.T) {
		v := setup()
		baseline := v.ImmutableRule()
		debugLogf(t, "baseline:\n%s\n", baseline)

		err := v.Mutate(indigo.Move("two", "childC"))
		if err != nil {
			t.Fatalf("moving rule failed: %v", err)
		}

		after := v.ImmutableRule()

		debugLogf(t, "baseline after mutation (should not change):\n%s\n", baseline)
		debugLogf(t, "after mutation:\n%s\n", after)
		want := `
┌──────────────────────────────────────────────────┐
│                                                  │
│ INDIGO RULES                                     │
│                                                  │
├───────────┬────────┬────────────┬────────┬───────┤
│           │        │            │ Result │       │
│ Rule      │ Schema │ Expression │ Type   │ Meta  │
├───────────┼────────┼────────────┼────────┼───────┤
│ root      │        │            │ <nil>  │ <nil> │
│   childA  │        │            │ <nil>  │ <nil> │
│     one   │        │            │ <nil>  │ <nil> │
│   childB  │        │            │ <nil>  │ <nil> │
│     three │        │            │ <nil>  │ <nil> │
│   childC  │        │            │ <nil>  │ <nil> │
│     two   │        │            │ <nil>  │ <nil> │
└───────────┴────────┴────────────┴────────┴───────┘
`
		assertEqual(want, after.String(), t)
		assertEqual(baselineStr, baseline.String(), t)
		assertPointersDifferent(t, baseline, after, "root", "childA", "childC")
		assertPointersEqual(t, baseline, after, "childB", "three")
	})
}

// // This test mutates a rule in a Vault and makes sure that the changes
// // are not seen outside the vault by someone who grabbed to rule before
// // mutations happened.
//
//	func TestVault_MutationsWithShards(t *testing.T) {
//		setup := func() *indigo.Vault {
//			root := indigo.NewRule("root", "")
//			root.Add(indigo.NewRule("one", " type = 'A' "))
//			root.Add(indigo.NewRule("two", " type = 'A' "))
//			root.Add(indigo.NewRule("three", " type = 'B' "))
//			root.Add(indigo.NewRule("four", " type = 'D' "))
//
//			root.Shards = []*indigo.Rule {
//				{
//					ID: "A"
//					Rule: " true ",
//					Meta: func(r *indigo.Rule) {
//
//					}
//				}
//
//
//
//
//
//			}
//
//			v, _ := indigo.NewVault(root)
//			return v
//		}
//
//		// this is the baseline rule all mutations happen against
//		baselineStr := `
//
// ┌──────────────────────────────────────────────────┐
// │                                                  │
// │ INDIGO RULES                                     │
// │                                                  │
// ├───────────┬────────┬────────────┬────────┬───────┤
// │           │        │            │ Result │       │
// │ Rule      │ Schema │ Expression │ Type   │ Meta  │
// ├───────────┼────────┼────────────┼────────┼───────┤
// │ root      │        │            │ <nil>  │ <nil> │
// │   childA  │        │            │ <nil>  │ <nil> │
// │     one   │        │            │ <nil>  │ <nil> │
// │     two   │        │            │ <nil>  │ <nil> │
// │   childB  │        │            │ <nil>  │ <nil> │
// │     three │        │            │ <nil>  │ <nil> │
// │   childC  │        │            │ <nil>  │ <nil> │
// └───────────┴────────┴────────────┴────────┴───────┘
// `
//
//	t.Run("delete", func(t *testing.T) {
//		v := setup()
//		baseline := v.ImmutableRule()
//		debugLogf(t, "baseline:\n%s\n", baseline)
//
//		err := v.Mutate(indigo.Delete("three"))
//		if err != nil {
//			t.Fatalf("Updating rule failed: %v", err)
//		}
//
//		after := v.ImmutableRule()
//
//		debugLogf(t, "baseline after mutation (should not change):\n%s\n", baseline)
//		debugLogf(t, "after mutation:\n%s\n", after)
//		want := `
//
// ┌─────────────────────────────────────────────────┐
// │                                                 │
// │ INDIGO RULES                                    │
// │                                                 │
// ├──────────┬────────┬────────────┬────────┬───────┤
// │          │        │            │ Result │       │
// │ Rule     │ Schema │ Expression │ Type   │ Meta  │
// ├──────────┼────────┼────────────┼────────┼───────┤
// │ root     │        │            │ <nil>  │ <nil> │
// │   childA │        │            │ <nil>  │ <nil> │
// │     one  │        │            │ <nil>  │ <nil> │
// │     two  │        │            │ <nil>  │ <nil> │
// │   childB │        │            │ <nil>  │ <nil> │
// │   childC │        │            │ <nil>  │ <nil> │
// └──────────┴────────┴────────────┴────────┴───────┘
// `
//
//		assertEqual(want, after.String(), t)
//		assertEqual(baselineStr, baseline.String(), t)
//		assertPointersDifferent(t, baseline, after, "root", "childB")
//		assertPointersEqual(t, baseline, after, "childC", "childA", "one", "two")
//	})
//
//	t.Run("update", func(t *testing.T) {
//		v := setup()
//		baseline := v.ImmutableRule()
//		debugLogf(t, "baseline:\n%s\n", baseline)
//
//		err := v.Mutate(indigo.Update(indigo.NewRule("one", "true")))
//		if err != nil {
//			t.Fatalf("Updating rule failed: %v", err)
//		}
//
//		after := v.ImmutableRule()
//
//		debugLogf(t, "baseline after mutation (should not change):\n%s\n", baseline)
//		debugLogf(t, "after mutation:\n%s\n", after)
//		want := `
//
// ┌──────────────────────────────────────────────────┐
// │                                                  │
// │ INDIGO RULES                                     │
// │                                                  │
// ├───────────┬────────┬────────────┬────────┬───────┤
// │           │        │            │ Result │       │
// │ Rule      │ Schema │ Expression │ Type   │ Meta  │
// ├───────────┼────────┼────────────┼────────┼───────┤
// │ root      │        │            │ <nil>  │ <nil> │
// │   childA  │        │            │ <nil>  │ <nil> │
// │     one   │        │ true       │ <nil>  │ <nil> │
// │     two   │        │            │ <nil>  │ <nil> │
// │   childB  │        │            │ <nil>  │ <nil> │
// │     three │        │            │ <nil>  │ <nil> │
// │   childC  │        │            │ <nil>  │ <nil> │
// └───────────┴────────┴────────────┴────────┴───────┘
// `
//
//		assertEqual(want, after.String(), t)
//		assertEqual(baselineStr, baseline.String(), t)
//		assertPointersDifferent(t, baseline, after, "root", "childA", "one")
//		assertPointersEqual(t, baseline, after, "two", "childB", "three", "childC")
//	})
//
//	t.Run("add", func(t *testing.T) {
//		v := setup()
//		baseline := v.ImmutableRule()
//		debugLogf(t, "baseline:\n%s\n", baseline)
//
//		err := v.Mutate(indigo.Add(indigo.NewRule("XXX", "false"), "childA"))
//		if err != nil {
//			t.Fatalf("adding rule failed: %v", err)
//		}
//
//		after := v.ImmutableRule()
//		debugLogf(t, "baseline after mutation (should not change):\n%s\n", baseline)
//		debugLogf(t, "after mutation:\n%s\n", after)
//		want := `
//
// ┌──────────────────────────────────────────────────┐
// │                                                  │
// │ INDIGO RULES                                     │
// │                                                  │
// ├───────────┬────────┬────────────┬────────┬───────┤
// │           │        │            │ Result │       │
// │ Rule      │ Schema │ Expression │ Type   │ Meta  │
// ├───────────┼────────┼────────────┼────────┼───────┤
// │ root      │        │            │ <nil>  │ <nil> │
// │   childA  │        │            │ <nil>  │ <nil> │
// │     XXX   │        │ false      │ <nil>  │ <nil> │
// │     one   │        │            │ <nil>  │ <nil> │
// │     two   │        │            │ <nil>  │ <nil> │
// │   childB  │        │            │ <nil>  │ <nil> │
// │     three │        │            │ <nil>  │ <nil> │
// │   childC  │        │            │ <nil>  │ <nil> │
// └───────────┴────────┴────────────┴────────┴───────┘
//
//		`
//		assertEqual(want, after.String(), t)
//		assertEqual(baselineStr, baseline.String(), t)
//		assertPointersDifferent(t, baseline, after, "root", "childA")
//		assertPointersEqual(t, baseline, after, "childB")
//	})
//
//	t.Run("move", func(t *testing.T) {
//		v := setup()
//		baseline := v.ImmutableRule()
//		debugLogf(t, "baseline:\n%s\n", baseline)
//
//		err := v.Mutate(indigo.Move("two", "childC"))
//		if err != nil {
//			t.Fatalf("moving rule failed: %v", err)
//		}
//
//		after := v.ImmutableRule()
//
//		debugLogf(t, "baseline after mutation (should not change):\n%s\n", baseline)
//		debugLogf(t, "after mutation:\n%s\n", after)
//		want := `
//
// ┌──────────────────────────────────────────────────┐
// │                                                  │
// │ INDIGO RULES                                     │
// │                                                  │
// ├───────────┬────────┬────────────┬────────┬───────┤
// │           │        │            │ Result │       │
// │ Rule      │ Schema │ Expression │ Type   │ Meta  │
// ├───────────┼────────┼────────────┼────────┼───────┤
// │ root      │        │            │ <nil>  │ <nil> │
// │   childA  │        │            │ <nil>  │ <nil> │
// │     one   │        │            │ <nil>  │ <nil> │
// │   childB  │        │            │ <nil>  │ <nil> │
// │     three │        │            │ <nil>  │ <nil> │
// │   childC  │        │            │ <nil>  │ <nil> │
// │     two   │        │            │ <nil>  │ <nil> │
// └───────────┴────────┴────────────┴────────┴───────┘
// `
//
//			assertEqual(want, after.String(), t)
//			assertEqual(baselineStr, baseline.String(), t)
//			assertPointersDifferent(t, baseline, after, "root", "childA", "childC")
//			assertPointersEqual(t, baseline, after, "childB", "three")
//		})
//	}
//
// assertPointersEqual returns an error if any of the named rules point to a different
// area in memory for in and b
func assertPointersEqual(t *testing.T, a *indigo.Rule, b *indigo.Rule, ruleIDs ...string) {
	t.Helper()
	for _, ruleID := range ruleIDs {
		ar, _ := a.FindRule(ruleID)
		br, _ := b.FindRule(ruleID)
		if ar != br && ar != nil && br != nil {
			t.Errorf("rules are different %s (%p) != %s (%p)", ar.ID, ar, br.ID, br)
			return
		}
	}
}

// assertPointersDifferent returns an error if any of the named rules point to the same area in memory
// for a and b
func assertPointersDifferent(t *testing.T, a *indigo.Rule, b *indigo.Rule, ruleIDs ...string) {
	t.Helper()
	for _, ruleID := range ruleIDs {
		var ar *indigo.Rule
		var br *indigo.Rule
		if ruleID == "root" {
			ar = a
			br = b
		}
		ar, _ = a.FindRule(ruleID)
		br, _ = b.FindRule(ruleID)
		if ar == br && ar != nil && br != nil {
			t.Errorf("rules are the same: %s (%p) == %s (%p)", ar.ID, ar, br.ID, br)
			return
		}
	}
}
