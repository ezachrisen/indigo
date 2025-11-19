// vault_test.go
package indigo_test

import (
	"context"
	"fmt"
	"testing"

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
	fmt.Println(v.CurrentRoot())
	r := &indigo.Rule{
		ID:   "a",
		Expr: `11 > 10`,
	}

	if err := v.ApplyMutations([]indigo.RuleMutation{{ID: "a", Rule: r, Parent: "root"}}); err != nil {
		t.Fatal(err)
	}
	res, err := e.Eval(context.Background(), v.CurrentRoot(), map[string]any{"value": 15})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pass {
		t.Error("expected pass")
	}

	rr := v.CurrentRoot()
	if len(rr.Rules) != 1 {
		t.Errorf("missing rule")
	}
	if a, ok := rr.Rules["a"]; !ok || a.Expr != "11 > 10" {
		t.Errorf("missing or incorrect rule")
	}
}

func TestVault_DeleteRule(t *testing.T) {
	_, v := setup(t)

	err := v.ApplyMutations([]indigo.RuleMutation{
		{ID: "a", Parent: "root", Rule: &indigo.Rule{ID: "a", Rules: map[string]*indigo.Rule{
			"child": {ID: "child", Expr: `true`},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Delete child
	if err := v.ApplyMutations([]indigo.RuleMutation{{ID: "child", Rule: nil}}); err != nil {
		t.Fatal(err)
	}

	root := v.CurrentRoot()
	if root == nil {
		t.Fatal("root became nil")
	}
	child := indigo.FindRule(v.CurrentRoot(), "child")
	if child != nil {
		t.Error("deleted child still present")
	}

	// try to delete a rule that doesn't exist
	if err := v.ApplyMutations([]indigo.RuleMutation{{ID: "XXX", Rule: nil}}); err == nil {
		t.Fatal("wanted error")
	}
}

func TestVault_UpdateRule(t *testing.T) {
	e, v := setup(t)

	old := &indigo.Rule{ID: "rule1", Expr: `2+2 == 3`}
	if err := v.ApplyMutations([]indigo.RuleMutation{{ID: "rule1", Rule: old, Parent: "root"}}); err != nil {
		t.Fatal(err)
	}
	// t.Logf("Before\n%s\n", v.CurrentRoot())

	res, err := e.Eval(context.Background(), v.CurrentRoot(), map[string]any{"x": 42})
	if err != nil {
		t.Fatal(err)
	}
	if res.Pass {
		t.Error("eval failed")
	}

	newRule := &indigo.Rule{ID: "rule1", Expr: `2 + 2 == 4`}
	if err := v.ApplyMutations([]indigo.RuleMutation{{ID: "rule1", Rule: newRule}}); err != nil {
		t.Fatal(err)
	}

	res, err = e.Eval(context.Background(), v.CurrentRoot(), map[string]any{"x": 42})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pass {
		t.Error("update did not take effect")
	}

	// t.Logf("After\n%s\n", v.CurrentRoot())
	if v.CurrentRoot().Rules["rule1"].Expr != `2 + 2 == 4` {
		t.Errorf("incorrect rule")
	}
}

func TestVault_AddRule(t *testing.T) {
	e, v := setup(t)

	old := &indigo.Rule{ID: "rule1", Expr: `2+2 == 4`}
	if err := v.ApplyMutations([]indigo.RuleMutation{{ID: "rule1", Rule: old, Parent: "root"}}); err != nil {
		t.Fatal(err)
	}
	// t.Logf("Before\n%s\n", v.CurrentRoot())

	res, err := e.Eval(context.Background(), v.CurrentRoot(), map[string]any{"x": 42})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pass {
		t.Error("eval failed")
	}

	newRule := &indigo.Rule{ID: "rule2", Expr: `10<1`}
	if err := v.ApplyMutations([]indigo.RuleMutation{{ID: "rule2", Rule: newRule, Parent: "root"}}); err != nil {
		t.Fatal(err)
	}

	res, err = e.Eval(context.Background(), v.CurrentRoot(), map[string]any{"x": 42})
	if err != nil {
		t.Fatal(err)
	}
	if res.Pass {
		t.Error("update did not take effect")
	}

	// t.Logf("After\n%s\n", v.CurrentRoot())
	if v.CurrentRoot().Rules["rule1"].Expr != `2+2 == 4` {
		t.Errorf("incorrect rule")
	}
}

func TestVault_MoveRule(t *testing.T) {
	_, v := setup(t)

	one := &indigo.Rule{ID: "rule1", Expr: `2+2 == 4`}
	b := &indigo.Rule{ID: "b", Expr: `10 > 1`}
	one.Rules = map[string]*indigo.Rule{}
	one.Rules["b"] = b

	two := &indigo.Rule{ID: "rule2", Expr: `1+1 == 2`}

	if err := v.ApplyMutations([]indigo.RuleMutation{
		{ID: "rule1", Rule: one, Parent: "root"},
		{ID: "rule2", Rule: two, Parent: "root"},
	}); err != nil {
		t.Fatal(err)
	}
	// t.Logf("Before\n%s\n", v.CurrentRoot())

	if err := v.ApplyMutations([]indigo.RuleMutation{{ID: "b", NewParent: "rule2"}}); err != nil {
		t.Fatal(err)
	}

	//	t.Logf("After\n%s\n", v.CurrentRoot())
	if v.CurrentRoot().Rules["rule2"].Rules["b"].Expr != `10 > 1` {
		t.Errorf("incorrect rule")
	}
}
