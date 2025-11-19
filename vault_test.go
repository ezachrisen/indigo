// vault_test.go
package indigo_test

import (
	"context"
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

	// Delete child
	if err := v.Mutate(indigo.Delete("child")); err != nil {
		t.Fatal(err)
	}

	root := v.Rule()
	if root == nil {
		t.Fatal("root became nil")
	}
	child := indigo.FindRule(v.Rule(), "child")
	if child != nil {
		t.Error("deleted child still present")
	}

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
	// t.Logf("Before\n%s\n", v.CurrentRoot())

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

	// t.Logf("After\n%s\n", v.CurrentRoot())
	if v.Rule().Rules["rule1"].Expr != `2 + 2 == 4` {
		t.Errorf("incorrect rule")
	}
}

func TestVault_AddRule(t *testing.T) {
	e, v := setup(t)

	old := &indigo.Rule{ID: "rule1", Expr: `2+2 == 4`}
	if err := v.Mutate(indigo.Add(*old, "root")); err != nil {
		t.Fatal(err)
	}
	// t.Logf("Before\n%s\n", v.CurrentRoot())

	res, err := e.Eval(context.Background(), v.Rule(), map[string]any{"x": 42})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pass {
		t.Error("eval failed")
	}

	newRule := &indigo.Rule{ID: "rule2", Expr: `10<1`}
	if err := v.Mutate(indigo.Add(*newRule, "root")); err != nil {
		t.Fatal(err)
	}

	res, err = e.Eval(context.Background(), v.Rule(), map[string]any{"x": 42})
	if err != nil {
		t.Fatal(err)
	}

	// t.Logf("After\n%s\n", v.CurrentRoot())
	if v.Rule().Rules["rule1"].Expr != `2+2 == 4` {
		t.Errorf("incorrect rule")
	}
	if v.Rule().Rules["rule2"].Expr != `10<1` {
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
	if err := v.Mutate(indigo.Add(*one, "root"), indigo.Add(*two, "root")); err != nil {
		t.Fatal(err)
	}
	// t.Logf("Before\n%s\n", v.CurrentRoot())

	if err := v.Mutate(indigo.Move("b", "rule2")); err != nil {
		t.Fatal(err)
	}

	// t.Logf("After\n%s\n", v.CurrentRoot())
	if v.Rule().Rules["rule2"].Rules["b"].Expr != `10 > 1` {
		t.Errorf("incorrect rule")
	}
}
