// vault_test.go
package indigo_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
)

// Helper to create a fresh engine + vault for each test
func setup(t *testing.T) (indigo.Engine, *indigo.Vault) {
	eng := indigo.NewEngine(cel.NewEvaluator())
	v, err := indigo.NewVault(eng, nil)
	if err != nil {
		t.Fatal(err)
	}
	return eng, v
}

func TestVault_BasicAddAndEval(t *testing.T) {
	e, v := setup(t)
	r := &indigo.Rule{
		ID:   "a",
		Expr: `11 > 10`,
	}
	t1 := time.Date(2020, 10, 10, 12, 0o0, 0o0, 0o0, time.UTC)
	if err := v.Mutate(indigo.LastUpdate(t1)); err != nil {
		t.Fatal(err)
	}
	t2 := time.Date(2022, 10, 10, 12, 0o0, 0o0, 0o0, time.UTC)
	if err := v.Mutate(indigo.Add(*r, "root"), indigo.LastUpdate(t2)); err != nil {
		t.Fatal(err)
	}
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

	rr := v.Rule()
	if len(rr.Rules) != 1 {
		t.Errorf("missing rule")
	}
	if a, ok := rr.Rules["a"]; !ok || a.Expr != "11 > 10" {
		t.Errorf("missing or incorrect rule")
	}
}

func TestVault_DeleteRule(t *testing.T) {
	_, v := setup(t)

	a := indigo.Rule{ID: "a", Rules: map[string]*indigo.Rule{
		"child": {ID: "child", Expr: `true`},
	}}

	err := v.Mutate(indigo.Add(a, "root"))
	if err != nil {
		t.Fatal(err)
	}
	// snapshot snapshot
	snapshot := v.Rule()

	// t.Logf("Snapshot before\n%s\n", snapshot)
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

	// t.Logf("Snapshot after: \n%s\n", snapshot)
	// t.Logf("Updated after: \n%s\n", root)
	// try to delete a rule that doesn't exist
	if err := v.Mutate(indigo.Delete("XXX")); err == nil {
		t.Fatal("wanted error")
	}
}

func TestVault_UpdateRule(t *testing.T) {
	e, v := setup(t)

	old := &indigo.Rule{ID: "rule1", Expr: `2+2 == 3`}
	if err := v.Mutate(indigo.Add(*old, "root")); err != nil {
		t.Fatal(err)
	}
	snapshot := v.Rule()
	// t.Logf("Snapshot before\n%s\n", snapshot)

	res, err := e.Eval(context.Background(), v.Rule(), map[string]any{"x": 42})
	if err != nil {
		t.Fatal(err)
	}
	if res.Pass {
		t.Error("eval failed")
	}
	newRule := &indigo.Rule{ID: "rule1", Expr: `2 + 2 == 4`}
	if err := v.Mutate(indigo.Update(*newRule)); err != nil {
		t.Fatal(err)
	}

	res, err = e.Eval(context.Background(), v.Rule(), map[string]any{"x": 42})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pass {
		t.Error("update did not take effect")
	}

	// t.Logf("After\n%s\n", v.Rule())
	if v.Rule().Rules["rule1"].Expr != `2 + 2 == 4` {
		t.Errorf("incorrect rule")
	}

	// t.Logf("Snapshot After\n%s\n", snapshot)
	if e := snapshot.Rules["rule1"].Expr; e != `2+2 == 3` {
		t.Errorf("snapshot was updated: %s", e)
	}
}

func TestVault_AddRule(t *testing.T) {
	e, v := setup(t)

	old := &indigo.Rule{ID: "rule1", Expr: `2+2 == 4`}
	if err := v.Mutate(indigo.Add(*old, "root")); err != nil {
		t.Fatal(err)
	}

	res, err := e.Eval(context.Background(), v.Rule(), map[string]any{"x": 42})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pass {
		t.Error("eval failed")
	}

	// t.Logf("Before adding rule2 to root:\n%s\n", v.Rule())
	newRule := &indigo.Rule{ID: "rule2", Expr: `10<1`}
	if err := v.Mutate(indigo.Add(*newRule, "root")); err != nil {
		t.Fatal(err)
	}
	snapshot := v.Rule()
	// t.Logf("Snapshot before adding rule1.2 to rule1:\n%s\n", snapshot)
	newRule2 := &indigo.Rule{ID: "rule1.2", Expr: `10<1`}
	newRule3 := &indigo.Rule{ID: "rule1.3", Expr: `10<1`}
	if err := v.Mutate(indigo.Add(*newRule2, "rule1"), indigo.Add(*newRule3, "rule1")); err != nil {
		t.Fatal(err)
	}

	// t.Logf("After:\n%s\n", v.Rule())
	_, err = e.Eval(context.Background(), v.Rule(), map[string]any{"x": 42})
	if err != nil {
		t.Fatal(err)
	}

	// t.Logf("After\n%s\n", v.Rule())
	if v.Rule().Rules["rule1"].Expr != `2+2 == 4` {
		t.Errorf("incorrect rule")
	}
	if v.Rule().Rules["rule2"].Expr != `10<1` {
		t.Errorf("incorrect rule")
	}
	// t.Logf("Snapshot after adds: %s", snapshot)
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
	if err := v.Mutate(indigo.Add(*one, "root"), indigo.Add(*two, "root")); err != nil {
		t.Fatal(err)
	}
	// t.Logf("Before\n%s\n", v.Rule())

	if err := v.Mutate(indigo.Move("b", "rule2")); err != nil {
		t.Fatal(err)
	}

	// t.Logf("After\n%s\n", v.Rule())
	if v.Rule().Rules["rule2"].Rules["b"].Expr != `10 > 1` {
		t.Errorf("incorrect rule")
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

			// t.Logf("Evaluating: \n%s\n", rule)
			res, err := eng.Eval(ctx, rule, map[string]any{})
			if err != nil {
				panic(err)
			}
			expected := desiredResult.Load()
			// t.Logf("Result: \n%s\n", res)
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
			if err := v.Mutate(indigo.Update(*r)); err != nil {
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

			// t.Logf("After update: \n%s\n", v.Rule())
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
