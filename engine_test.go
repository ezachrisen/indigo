package indigo_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
)

// Test the scenario where the rule has childs and exists (or not) childs with true as their results
func TestTrueIfAnyBehavior(t *testing.T) {

	engine := indigo.NewEngine(cel.NewEvaluator())
	data := map[string]any{}
	ctx := context.Background()

	ruleL2 := &indigo.Rule{ID: "l2", Expr: `false`}
	ruleL1_1 := &indigo.Rule{ID: "l1-1", Expr: `true`}
	ruleL1 := &indigo.Rule{
		Expr: `true`,
		ID:   "l1",
		Rules: map[string]*indigo.Rule{
			ruleL1_1.ID: ruleL1_1,
			"l1-2":      {Expr: `true`, ID: "l1-2"},
		},
	}
	rootRule := &indigo.Rule{
		ID:          "root",
		Expr:        `true`,
		EvalOptions: indigo.EvalOptions{TrueIfAny: true},
		Rules: map[string]*indigo.Rule{
			ruleL1.ID: ruleL1,
			ruleL2.ID: ruleL2,
		},
	}

	expectedResults := map[string]bool{
		rootRule.ID: true, // don't matter this value, since it'll be changed in the loop
		ruleL2.ID:   true,
		ruleL1.ID:   true,
		ruleL1_1.ID: true,
		"l1-2":      true, // if false, the ruleL1 always will be false
	}

	// function to check the changes over the expected rules tree
	check := func(l1, l2 bool) {
		ruleL1_1.Expr = strconv.FormatBool(l1) // assign the result of an Exp for the child rule
		ruleL2.Expr = strconv.FormatBool(l2)

		expectedResults[ruleL2.ID] = l2
		expectedResults[ruleL1_1.ID] = l1       // change the expected value of the leaf
		expectedResults[ruleL1.ID] = l1         // until the root
		expectedResults[rootRule.ID] = l2 || l1 // since the leaf rule will propagate the change

		err := engine.Compile(rootRule)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := engine.Eval(ctx, rootRule, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// verify if matches the expected result
		if err := match(flattenResultsRuleResult(result), expectedResults); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// possible scenarios for the leaf change
	for _, expect := range []bool{false, true} {
		check(expect, false) // if the L2 is false, then we need to check from leaf until the root
		check(expect, true)  // otherwise, the L2 will make the root to be true
	}

}

// Test that all rules are evaluated and yield the correct result in the default configuration
func TestEvaluationTraversalDefault(t *testing.T) {

	e := indigo.NewEngine(newMockEvaluator())
	r := makeRule()

	expectedResults := map[string]bool{
		"rule1": true,
		"D":     true,
		"d1":    true,
		"d2":    false,
		"d3":    true,
		"B":     false,
		"b1":    true,
		"b2":    false,
		"b3":    true,
		"b4":    false,
		"b4-1":  true,
		"b4-2":  false,
		"E":     false,
		"e1":    true,
		"e2":    false,
		"e3":    true,
	}

	err := e.Compile(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := e.Eval(context.Background(), r, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	//	fmt.Println(m.rulesTested)
	//	fmt.Println(result)
	if err := match(flattenResultsExprResult(result), expectedResults); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Ensure that rules are evaluated in the correct order when
// alpha sort is applied
func TestEvaluationTraversalAlphaSort(t *testing.T) {

	e := indigo.NewEngine(newMockEvaluator())
	r := makeRule()

	// Specify the sort order for all rules
	err := indigo.ApplyToRule(r, func(r *indigo.Rule) error {
		r.EvalOptions.SortFunc = indigo.SortRulesAlpha
		r.EvalOptions.StopFirstNegativeChild = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = indigo.ApplyToRule(r, func(r *indigo.Rule) error {
		r.Expr = "true"
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = e.Compile(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	//	fmt.Println(r)
	expectedResults := map[string]bool{
		"rule1": true,
		"D":     true,
		"d1":    true,
		"d2":    true,
		"d3":    true,
		"B":     true,
		"b1":    true,
		"b2":    true,
		"b3":    true,
		"b4":    true,
		"b4-1":  true,
		"b4-2":  true,
		"E":     true,
		"e1":    true,
		"e2":    true,
		"e3":    true,
	}

	// If everything works, the rules were evaluated in this order
	//	(alphabetically)
	expectedOrder := []string{
		"rule1",
		"B",
		"b1",
		"b2",
		"b3",
		"b4",
		"b4-1",
		"b4-2",
		"D",
		"d1",
		"d2",
		"d3",
		"E",
		"e1",
		"e2",
		"e3",
	}

	result, err := e.Eval(context.Background(), r, map[string]any{}, indigo.ReturnDiagnostics(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	//	fmt.Println(m.rulesTested)
	// fmt.Println(result)
	if err := match(flattenResultsExprResult(result), expectedResults); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// fmt.Printf("Expected: %+v\n", expectedOrder)
	// fmt.Printf("Got     : %+v\n", flattenResultsEvaluated(result))
	if !reflect.DeepEqual(expectedOrder, flattenResultsEvaluated(result)) {
		t.Error("not all rules were evaluated")
	}

}

// Test that the engine checks for nil data and rule
func TestNilDataOrRule(t *testing.T) {
	e := indigo.NewEngine(newMockEvaluator())
	r := makeRule()
	err := e.Compile(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = e.Eval(context.Background(), r, nil)
	if err == nil {
		t.Error("should get an error if the data map is nil")
	}
	if !strings.Contains(err.Error(), "data is nil") {
		t.Error("error should contain 'data is nil'")
	}

	_, err = e.Eval(context.Background(), nil, map[string]any{})
	if err == nil {
		t.Error("should get an error if the rule is nil")
	}
	if !strings.Contains(err.Error(), "rule is nil") {
		t.Error("error should contain 'rule is nil'")
	}

	r.Rules["B"].Rules["oops"] = nil
	_, err = e.Eval(context.Background(), r, map[string]any{})
	if err == nil {
		t.Error("should get an error if the rule is nil")
	}
	if !strings.Contains(err.Error(), "rule is nil") {
		t.Error("error should contain 'rule is nil'")
	}

}

// Test the pass/fail of the expression evaluation with various combinations
// of evaluation options
// This tests the result.ExpressionPass field.
func TestEvalOptionsExpressionPassFail(t *testing.T) {

	e := indigo.NewEngine(newMockEvaluator())
	d := map[string]any{"a": "a"} // dummy data, not important
	w := map[string]bool{         // the wanted expression evaluation results with no options in effect
		"rule1": true,
		"D":     true,
		"d1":    true,
		"d2":    false,
		"d3":    true,
		"B":     false,
		"b1":    true,
		"b2":    false,
		"b3":    true,
		"b4":    false,
		"b4-1":  true,
		"b4-2":  false,
		"E":     false,
		"e1":    true,
		"e2":    false,
		"e3":    true,
	}

	cases := map[string]struct {
		prep func(*indigo.Rule)     // a function called to prep the rule before compilation and evaluation
		want func() map[string]bool // a function returning an edited copy of w after options are applied
	}{
		"Default Options": {
			prep: func(r *indigo.Rule) {
			},
			want: func() map[string]bool {
				return copyMap(w)
			},
		},

		"StopIfParentNegative": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.StopIfParentNegative = true
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b1", "b2", "b3", "b4", "b4-1", "b4-2")
			},
		},
		"DiscardPass": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].Rules["b4"].EvalOptions.DiscardPass = true
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b4-1")
			},
		},
		"DiscardFailures": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.DiscardFail = indigo.Discard
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b2", "b4", "b4-1", "b4-2") // b4-1 is elim. because b4 is
			},
		},
		"DiscardPass & DiscardFailures": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.DiscardFail = indigo.Discard
				r.Rules["B"].EvalOptions.DiscardPass = true
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b1", "b2", "b3", "b4", "b4-1", "b4-2")
			},
		},
		"DiscardPass & DiscardFailures on Root": {
			prep: func(r *indigo.Rule) {
				r.EvalOptions.DiscardFail = indigo.Discard
				r.EvalOptions.DiscardPass = true
			},
			want: func() map[string]bool {
				return map[string]bool{"rule1": true}
			},
		},
		"StopFirstPositiveChildX": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.StopFirstPositiveChild = true
				r.Rules["B"].EvalOptions.SortFunc = indigo.SortRulesAlpha
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b2", "b3", "b4", "b4-1", "b4-2")
			},
		},
		"StopFirstNegativeChild": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.StopFirstNegativeChild = true
				r.Rules["B"].EvalOptions.SortFunc = indigo.SortRulesAlpha
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b3", "b4", "b4-1", "b4-2")
			},
		},
		"StopFirstNegativeChild & StopFirstPositiveChild": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.StopFirstNegativeChild = true
				r.Rules["B"].EvalOptions.StopFirstPositiveChild = true
				r.Rules["B"].EvalOptions.SortFunc = indigo.SortRulesAlpha
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b2", "b3", "b4", "b4-1", "b4-2")
			},
		},
		"Multiple Options": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.DiscardPass = true
				r.Rules["B"].Rules["b4"].EvalOptions.DiscardPass = true
				r.Rules["E"].EvalOptions.StopIfParentNegative = true
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b1", "b3", "b4-1", "e1", "e2", "e3")
			},
		},
		"Delete Rule": {
			prep: func(r *indigo.Rule) {
				delete(r.Rules["B"].Rules, "b1")
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b1")
			},
		},
		"Add Rule": {
			prep: func(r *indigo.Rule) {
				x := &indigo.Rule{
					ID:   "x",
					Expr: "true",
				}
				r.Rules[x.ID] = x
			},
			want: func() map[string]bool {
				w := copyMap(w)
				w["x"] = true
				return w
			},
		},
		"Update Rule": { // TODO and forget to compile; stale program?
			prep: func(r *indigo.Rule) {
				r.Rules["B"].Rules["b4"].Expr = "true"
			},
			want: func() map[string]bool {
				w := copyMap(w)
				w["b4"] = true
				return w
			},
		},
	}

	for k, c := range cases {
		c := c
		t.Run(k, func(t *testing.T) {
			r := makeRule()
			c.prep(r)
			err := e.Compile(r)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			u, err := e.Eval(context.Background(), r, d, indigo.ReturnDiagnostics(true))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			//			fmt.Println("got: ", flattenResultsExprResult(u))
			err = match(flattenResultsExprResult(u), c.want())
			if err != nil {
				t.Errorf("%v", err)
			}
		})
	}
}

// Test the pass/fail of the rule  evaluation with various combinations
// of evaluation options
// This tests the result.Pass field
func TestEvalOptionsRulePassFail(t *testing.T) {

	e := indigo.NewEngine(newMockEvaluator())
	d := map[string]any{"a": "a"} // dummy data, not important
	// the wanted rule evaluation results with no options in effect
	// Note that the rule produced by make_rule is manipulated in the loop before
	// running the test cases
	w := map[string]bool{
		"rule1": false,
		"D":     true,
		"d1":    true,
		"d2":    true,
		"d3":    true,
		"B":     false,
		"b1":    true,
		"b2":    false,
		"b3":    true,
		"b4":    false,
		"b4-1":  true,
		"b4-2":  false,
		"E":     false,
		"e1":    true,
		"e2":    false,
		"e3":    true,
	}

	cases := map[string]struct {
		prep func(*indigo.Rule)     // a function called to prep the rule before compilation and evaluation
		want func() map[string]bool // a function returning an edited copy of w after options are applied
	}{
		"Default Options": {
			prep: func(r *indigo.Rule) {
			},
			want: func() map[string]bool {
				return copyMap(w)
			},
		},

		"StopIfParentNegative": {
			prep: func(r *indigo.Rule) {
				r.Rules["E"].EvalOptions.StopIfParentNegative = true
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "e1", "e2", "e3")
			},
		},
		"DiscardPass": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].Rules["b4"].EvalOptions.DiscardPass = true
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b4-1")
			},
		},

		"DiscardOnlyIfExpressionFailed": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].Expr = "true"
				r.Rules["E"].Expr = "false"
				r.EvalOptions.DiscardFail = indigo.Discard
				r.EvalOptions.DiscardFail = indigo.DiscardOnlyIfExpressionFailed
				//				r.EvalOptions.FailAction = indigo.KeepFailures
				r.Rules["B"].EvalOptions.StopIfParentNegative = true
				r.Rules["E"].EvalOptions.StopIfParentNegative = true
				r.Rules["B"].EvalOptions.DiscardFail = indigo.KeepAll
				r.Rules["E"].EvalOptions.DiscardFail = indigo.Discard
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "E", "e1", "e2", "e3")
			},
		},

		"DiscardFailures": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].Expr = "true"
				r.Rules["E"].Expr = "false"
				r.EvalOptions.DiscardFail = indigo.Discard
				r.Rules["B"].EvalOptions.StopIfParentNegative = true
				r.Rules["E"].EvalOptions.StopIfParentNegative = true
				r.Rules["B"].EvalOptions.DiscardFail = indigo.KeepAll
				r.Rules["E"].EvalOptions.DiscardFail = indigo.Discard
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "B", "b2", "b3", "b4", "b4-1", "b4-2", "b1", "E", "e1", "e2", "e3")
			},
		},

		"KeepFailures": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].Expr = "true"
				r.Rules["E"].Expr = "false"
				r.EvalOptions.DiscardFail = indigo.KeepAll
				r.Rules["B"].EvalOptions.StopIfParentNegative = true
				r.Rules["E"].EvalOptions.StopIfParentNegative = true
				r.Rules["B"].EvalOptions.DiscardFail = indigo.KeepAll
				r.Rules["E"].EvalOptions.DiscardFail = indigo.Discard
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "e1", "e2", "e3")
			},
		},

		"DiscardPass & DiscardFailures": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.DiscardFail = indigo.Discard
				r.Rules["B"].EvalOptions.DiscardPass = true
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b1", "b2", "b3", "b4", "b4-1", "b4-2")
			},
		},
		"DiscardPass & DiscardFailures on Root": {
			prep: func(r *indigo.Rule) {
				r.EvalOptions.DiscardFail = indigo.Discard
				r.EvalOptions.DiscardPass = true
			},
			want: func() map[string]bool {
				return map[string]bool{"rule1": false}
			},
		},
		"StopFirstPositiveChild-1": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.StopFirstPositiveChild = true
				r.Rules["B"].EvalOptions.SortFunc = indigo.SortRulesAlpha
			},
			want: func() map[string]bool {
				m := deleteKeys(copyMap(w), "b2", "b3", "b4", "b4-1", "b4-2")
				// B will be true because b1 is true, and we stopped evaluating after the first
				// positive, so we never go to the negatives (b2, b4)
				m["B"] = true
				return m
			},
		},
		"StopFirstNegativeChild": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.StopFirstNegativeChild = true
				r.Rules["B"].EvalOptions.SortFunc = indigo.SortRulesAlpha
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b3", "b4", "b4-1", "b4-2")
			},
		},
		"StopFirstNegativeChild & StopFirstPositiveChild (1)": {
			// This will stop on the first negative child of B
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.StopFirstNegativeChild = true
				r.Rules["B"].EvalOptions.StopFirstPositiveChild = true
				r.Rules["B"].EvalOptions.SortFunc = indigo.SortRulesAlpha
			},
			want: func() map[string]bool {
				m := deleteKeys(copyMap(w), "b2", "b3", "b4", "b4-1", "b4-2")
				// B will be true because b1 is true, and we stopped evaluating after the first
				// positive, so we never go to the negatives (b2, b4)
				m["B"] = true
				return m
			},
		},
		"StopFirstNegativeChild & StopFirstPositiveChild (2)": {
			// This will stop on the first positive child of E
			prep: func(r *indigo.Rule) {
				r.Rules["E"].EvalOptions.StopFirstNegativeChild = true
				r.Rules["E"].EvalOptions.StopFirstPositiveChild = true
				r.Rules["E"].EvalOptions.SortFunc = indigo.SortRulesAlpha
				r.Rules["E"].Rules["e1"].Expr = `false` // make evaluation stop on the first rule
			},
			want: func() map[string]bool {
				m := deleteKeys(copyMap(w), "e2", "e3")
				m["e1"] = false
				return m
			},
		},

		"Multiple Options": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.DiscardPass = true
				r.Rules["B"].Rules["b4"].EvalOptions.DiscardPass = true
				r.Rules["E"].EvalOptions.StopIfParentNegative = true
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b1", "b3", "b4-1", "e1", "e2", "e3")
			},
		},
		"Delete Rule": {
			prep: func(r *indigo.Rule) {
				delete(r.Rules["B"].Rules, "b1")
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b1")
			},
		},
		"Add Rule": {
			prep: func(r *indigo.Rule) {
				x := &indigo.Rule{
					ID:   "x",
					Expr: "true",
				}
				r.Rules[x.ID] = x
			},
			want: func() map[string]bool {
				w := copyMap(w)
				w["x"] = true
				return w
			},
		},
	}

	for k, c := range cases {
		c := c
		t.Run(k, func(t *testing.T) {
			r := makeRule()

			// In the rule created by makeRule, ALL children of rule1 are
			// false, which would make this test not very interesting.
			// We'll manipulate a few of the rules to make them more interesting to test.

			r.Rules["D"].Rules["d2"].Expr = `true` // this will make D true
			r.Rules["B"].Expr = `true`             // this will not make B true, since its children are false

			c.prep(r)
			err := e.Compile(r)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			u, err := e.Eval(context.Background(), r, d, indigo.ReturnDiagnostics(true))

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			err = match(flattenResultsRuleResult(u), c.want())
			if err != nil {
				t.Errorf("%v", err)
				// fmt.Println(r)
				// fmt.Println(u)
			}
		})
	}
}

func TestEvalTrueIfAny(t *testing.T) {

	e := indigo.NewEngine(newMockEvaluator())
	d := map[string]any{"a": "a"} // dummy data, not important
	// the wanted rule evaluation results with no options in effect
	// Note that the rule produced by make_rule is manipulated in the loop before
	// running the test cases
	w := map[string]bool{
		"rule1": true,
		"D":     true, // D itself is true
		"d1":    true,
		"d2":    false, // this is false, bu D has trueifany set
		"d3":    true,
		"B":     false, // B itself is false
		"b1":    true,
		"b2":    false,
		"b3":    true,
		"b4":    false, // b4 itself is false
		"b4-1":  true,
		"b4-2":  false,
		"E":     false,
		"e1":    true,
		"e2":    false,
		"e3":    true,
	}

	cases := map[string]struct {
		prep func(*indigo.Rule)     // a function called to prep the rule before compilation and evaluation
		want func() map[string]bool // a function returning an edited copy of w after options are applied
	}{
		"Default Options": {
			prep: func(r *indigo.Rule) {
				r.EvalOptions.TrueIfAny = true
				r.Rules["B"].EvalOptions.TrueIfAny = true
				r.Rules["D"].EvalOptions.TrueIfAny = true
				r.Rules["E"].EvalOptions.TrueIfAny = true
			},
			want: func() map[string]bool {
				return copyMap(w)
			},
		},

		"StopIfParentNegative": {
			prep: func(r *indigo.Rule) {
				r.EvalOptions.TrueIfAny = true
				r.Rules["B"].EvalOptions.TrueIfAny = true
				r.Rules["D"].EvalOptions.TrueIfAny = true
				r.Rules["E"].EvalOptions.TrueIfAny = true
				r.Rules["B"].EvalOptions.StopIfParentNegative = true
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b1", "b2", "b3", "b4", "b4-1", "b4-2")
			},
		},
		"DiscardPass": {
			prep: func(r *indigo.Rule) {
				r.EvalOptions.TrueIfAny = true
				r.Rules["B"].EvalOptions.TrueIfAny = true
				r.Rules["D"].EvalOptions.TrueIfAny = true
				r.Rules["E"].EvalOptions.TrueIfAny = true
				r.Rules["B"].Rules["b4"].EvalOptions.DiscardPass = true
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b4-1")
			},
		},
		"DiscardFailures": {
			prep: func(r *indigo.Rule) {
				r.EvalOptions.TrueIfAny = true
				r.Rules["B"].EvalOptions.TrueIfAny = true
				r.Rules["D"].EvalOptions.TrueIfAny = true
				r.Rules["E"].EvalOptions.TrueIfAny = true
				r.Rules["B"].EvalOptions.DiscardFail = indigo.Discard
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b2", "b4", "b4-1", "b4-2") // b4-1 is elim. because b4 is
			},
		},
		"DiscardPass & DiscardFailures": {
			prep: func(r *indigo.Rule) {
				r.EvalOptions.TrueIfAny = true
				r.Rules["B"].EvalOptions.TrueIfAny = true
				r.Rules["D"].EvalOptions.TrueIfAny = true
				r.Rules["E"].EvalOptions.TrueIfAny = true

				r.Rules["B"].EvalOptions.DiscardFail = indigo.Discard
				r.Rules["B"].EvalOptions.DiscardPass = true
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b1", "b2", "b3", "b4", "b4-1", "b4-2")
			},
		},
		"DiscardPass & DiscardFailures on Root": {
			prep: func(r *indigo.Rule) {
				r.EvalOptions.TrueIfAny = true
				r.Rules["B"].EvalOptions.TrueIfAny = true
				r.Rules["D"].EvalOptions.TrueIfAny = true
				r.Rules["E"].EvalOptions.TrueIfAny = true

				r.EvalOptions.DiscardFail = indigo.Discard
				r.EvalOptions.DiscardPass = true
			},
			want: func() map[string]bool {
				return map[string]bool{"rule1": true}
			},
		},
		"StopFirstPositiveChild": {
			prep: func(r *indigo.Rule) {
				r.EvalOptions.TrueIfAny = true
				r.Rules["B"].EvalOptions.TrueIfAny = true
				r.Rules["D"].EvalOptions.TrueIfAny = true
				r.Rules["E"].EvalOptions.TrueIfAny = true

				r.Rules["B"].EvalOptions.StopFirstPositiveChild = true
				r.Rules["B"].EvalOptions.SortFunc = indigo.SortRulesAlpha
			},
			want: func() map[string]bool {
				m := deleteKeys(copyMap(w), "b2", "b3", "b4", "b4-1", "b4-2")
				return m
			},
		},
		"StopFirstNegativeChild": {
			prep: func(r *indigo.Rule) {
				r.EvalOptions.TrueIfAny = true
				r.Rules["B"].EvalOptions.TrueIfAny = true
				r.Rules["D"].EvalOptions.TrueIfAny = true
				r.Rules["E"].EvalOptions.TrueIfAny = true

				r.Rules["B"].EvalOptions.StopFirstNegativeChild = true
				r.Rules["B"].EvalOptions.SortFunc = indigo.SortRulesAlpha

				r.Rules["B"].Rules["b1"].Expr = "false"

			},
			want: func() map[string]bool {
				m := deleteKeys(copyMap(w), "b2", "b3", "b4", "b4-1", "b4-2")
				m["B"] = false
				m["b1"] = false
				return m
			},
		},
	}

	for k, c := range cases {
		c := c
		t.Run(k, func(t *testing.T) {
			r := makeRule()

			c.prep(r)
			err := e.Compile(r)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			u, err := e.Eval(context.Background(), r, d, indigo.ReturnDiagnostics(true))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			err = match(flattenResultsRuleResult(u), c.want())
			if err != nil {
				t.Errorf("Error in case %s: %v", k, err)
				fmt.Println(r)
				fmt.Println(u)
			}
		})
	}
}

func TestDiagnosticOptions(t *testing.T) {

	m := newMockEvaluator()
	e := indigo.NewEngine(m)
	d := map[string]any{"a": "a"} // dummy data, not important

	cases := map[string]struct {
		engineDiagnosticCompileRequired bool // whether engine should require compile-time diagnostics
		compileDiagnostics              bool // whether to request diagnostics at compile time
		evalDiagnostics                 bool // whether to request diagnostics at eval time
		wantDiagnostics                 bool // whether we expect to receive diagnostic information
	}{
		"No diagnostics requested at compile or eval time": {
			engineDiagnosticCompileRequired: true,
			compileDiagnostics:              false,
			evalDiagnostics:                 false,
			wantDiagnostics:                 false,
		},
		"No diagnostics requested at compile time, but at eval time": {
			engineDiagnosticCompileRequired: true,
			compileDiagnostics:              false,
			evalDiagnostics:                 true,
			wantDiagnostics:                 false,
		},
		"Diagnostics requested at compile time AND at eval time": {
			engineDiagnosticCompileRequired: true,
			compileDiagnostics:              true,
			evalDiagnostics:                 true,
			wantDiagnostics:                 true,
		},
	}

	for k, c := range cases {
		r := makeRule()

		// Set the mock engine to require that diagnostics must be turned on at compile time,
		// or not. This is a special feature of the mock engine, useful for testing.
		m.diagnosticCompileRequired = c.engineDiagnosticCompileRequired

		err := e.Compile(r, indigo.CollectDiagnostics(c.compileDiagnostics))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		u, err := e.Eval(context.Background(), r, d, indigo.ReturnDiagnostics(c.evalDiagnostics))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// fmt.Println(k)
		//fmt.Println(indigo.DiagnosticsReport(u, nil))
		switch c.wantDiagnostics {
		case true:
			err = allNotEmpty(flattenResultsDiagnostics(u))
			if err != nil {
				t.Errorf("In case '%s', wanted diagnostics: %v", k, err)
			}
			if len(flattenResultsEvaluated(u)) != 16 {
				t.Errorf("In case '%s', wanted list of rules diagnostics", k)
			}

			_ = u.Diagnostics.String()
			indigo.DiagnosticsReport(u, d)

			// Check that calling diagnostics with nils is ok
			indigo.DiagnosticsReport(nil, nil)

		default:
			//fmt.Println(u)
			err = anyNotEmpty(flattenResultsDiagnostics(u))
			if err != nil {
				t.Errorf("In case '%s', wanted no diagnostics, got: %v", k, err)
			}
		}

	}
}

// Test compiling some rules with compile-time collection of diagnostics and some without
func TestPartialDiagnostics(t *testing.T) {

	m := newMockEvaluator()
	e := indigo.NewEngine(m)
	// Set the mock engine to require that diagnostics must be turned on at compile time,
	// or not. This is a special feature of the mock engine, useful for testing.
	m.diagnosticCompileRequired = true
	d := map[string]any{"a": "a"} // dummy data, not important
	r := makeRule()

	// first, compile all the rules WITHOUT diagnostics
	err := e.Compile(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// then, re-compile rule B, WITH diagnostics
	err = e.Compile(r.Rules["B"], indigo.CollectDiagnostics(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, err := e.Eval(context.Background(), r, d, indigo.ReturnDiagnostics(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	f := flattenResultsDiagnostics(u)

	// We expect that only the B rules will have diagnostics
	for k, v := range f {
		if k == "B" || k == "b1" || k == "b2" || k == "b3" || k == "b4" || k == "b4-1" || k == "b4-2" {
			if v == nil {
				t.Error("condition should be true")
			}
		} else {
			if v != nil {
				fmt.Println("V = ", v, k)
			}
			if v != nil {
				t.Error("condition should be true")
			}
		}
	}
}

func TestJSON(t *testing.T) {

	//	e := indigo.NewEngine(newMockEvaluator())
	r := makeRule()

	_, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	//	fmt.Println(string(b))

}

// Test options set at the time eval is called
// (options apply to the entire tree)
func TestGlobalEvalOptions(t *testing.T) {

	cases := []struct {
		prep func(*indigo.Rule)     // Edits to apply to the rule before evaluating
		opts []indigo.EvalOption    // Options to pass to evaluate
		chk  func(r *indigo.Result) // Function to check the results
	}{
		{
			// Check that global (true) overrides default (false)
			opts: []indigo.EvalOption{indigo.StopIfParentNegative(true)},
			chk: func(r *indigo.Result) {
				if len(r.Results) != 3 {
					t.Errorf("expected 3, got %v", len(r.Results))
				}
				if len(r.Results["D"].Results) != 3 {
					t.Errorf("expected 3, got %v", len(r.Results["D"].Results))
				}
				if len(r.Results["E"].Results) != 0 {
					t.Errorf("expected 0, got %v", len(r.Results["E"].Results))
				}
				if len(r.Results["B"].Results) != 0 {
					t.Errorf("expected 0, got %v", len(r.Results["B"].Results))
				}
			},
		},

		// {
		// 	// Check global rollupchildresults (false)
		// 	prep: func(r *indigo.Rule) {
		// 	},
		// 	opts: []indigo.EvalOption{indigo.RollupChildResults(true)},
		// 	chk: func(r *indigo.Result) {
		// 		is.Equal(r.ExpressionPass, false)
		// 	},
		// },

		// {
		// 	// Check global rollupchildresults, should be true
		// 	prep: func(r *indigo.Rule) {
		// 		// only leave true rules
		// 		delete(r.Rules, "B")
		// 		delete(r.Rules, "E")
		// 		r.Rules["D"].Rules["d2"].Expr = `true`
		// 	},
		// 	opts: []indigo.EvalOption{indigo.RollupChildResults(true)},
		// 	chk: func(r *indigo.Result) {
		// 		is.True(r.ExpressionPass)
		// 	},
		// },

		{
			// Check that global (false) overrides local option (true)
			prep: func(r *indigo.Rule) {
				r.EvalOptions.StopIfParentNegative = true
				r.Rules["B"].EvalOptions.StopIfParentNegative = true
				r.Rules["D"].EvalOptions.StopIfParentNegative = true
				r.Rules["E"].EvalOptions.StopIfParentNegative = true
			},
			opts: []indigo.EvalOption{indigo.StopIfParentNegative(false)},
			chk: func(r *indigo.Result) {
				if len(r.Results) != 3 {
					t.Errorf("expected 3, got %v", len(r.Results))
				}
				if len(r.Results["D"].Results) != 3 {
					t.Errorf("expected 3, got %v", len(r.Results["D"].Results))
				}
				if len(r.Results["E"].Results) != 3 {
					t.Errorf("expected 3, got %v", len(r.Results["E"].Results))
				}
				if len(r.Results["B"].Results) != 4 {
					t.Errorf("expected 4, got %v", len(r.Results["B"].Results))
				}
			},
		},
		{
			// Check that the global sort function overrides the rule's sort function
			prep: func(r *indigo.Rule) {
				r.EvalOptions.StopIfParentNegative = true
				r.Rules["B"].EvalOptions.StopFirstPositiveChild = true
				r.Rules["B"].EvalOptions.SortFunc = indigo.SortRulesAlpha
				r.Rules["B"].EvalOptions.DiscardFail = indigo.Discard
				// we'll check that the 2 true rules in B, b1 and b3,
				// come back in the order we expect
				// with indigo.SortRulesAlpha, we expect b1 to be returned,
				// with indigo.SortRulesAlphaDesc we expect b3 to be returned
			},
			opts: []indigo.EvalOption{indigo.SortFunc(indigo.SortRulesAlphaDesc)},
			chk: func(r *indigo.Result) {
				if len(r.Results["B"].Results) != 1 {
					t.Errorf("expected 1, got %v", len(r.Results["B"].Results))
				}
				if x, ok := r.Results["B"].Results["b3"]; !ok {
					t.Errorf("expected b3, got %s", x.Rule.ID)
				}

			},
		},

		{
			// Check that global (true) overrides local option (false)
			opts: []indigo.EvalOption{indigo.StopFirstPositiveChild(true), indigo.SortFunc(indigo.SortRulesAlpha)},
			chk: func(r *indigo.Result) {
				if len(r.Results) != 2 {
					t.Errorf("expected 2, got %v", len(r.Results))
				} // B is false, D is first positive child
				if !r.Results["D"].ExpressionPass {
					t.Error("condition should be true")
				}
				if r.Results["B"].ExpressionPass {
					t.Error("condition should be false")
				}
			},
		},
		{
			// Check that global (false) overrides local option (true)
			prep: func(r *indigo.Rule) {
				r.EvalOptions.StopFirstPositiveChild = true
				r.Rules["B"].EvalOptions.StopFirstPositiveChild = true
				r.Rules["D"].EvalOptions.StopFirstPositiveChild = true
				r.Rules["E"].EvalOptions.StopFirstPositiveChild = true
				r.EvalOptions.SortFunc = indigo.SortRulesAlpha
				r.Rules["B"].EvalOptions.SortFunc = indigo.SortRulesAlpha
				r.Rules["D"].EvalOptions.SortFunc = indigo.SortRulesAlpha
				r.Rules["E"].EvalOptions.SortFunc = indigo.SortRulesAlpha
			},

			opts: []indigo.EvalOption{indigo.StopFirstPositiveChild(false)},
			chk: func(r *indigo.Result) {
				if len(r.Results) != 3 {
					t.Errorf("expected 3, got %v", len(r.Results))
				}
			},
		},

		{
			// Check that global (true) overrides local option (false)
			opts: []indigo.EvalOption{indigo.StopFirstNegativeChild(true), indigo.SortFunc(indigo.SortRulesAlpha)},
			chk: func(r *indigo.Result) {
				if len(r.Results) != 1 {
					t.Errorf("expected 1, got %v", len(r.Results))
				} // B is first, should stop evaluation
				if r.Results["B"].ExpressionPass {
					t.Error("condition should be false")
				}
			},
		},
		{
			// Check that global (true) overrides local option (false)
			opts: []indigo.EvalOption{indigo.StopFirstNegativeChild(true), indigo.StopFirstPositiveChild(true), indigo.SortFunc(indigo.SortRulesAlpha)},
			chk: func(r *indigo.Result) {
				if len(r.Results) != 1 {
					t.Errorf("expected 1, got %v", len(r.Results))
				} // B should stop it
				if r.Results["B"].ExpressionPass {
					t.Error("condition should be false")
				}
			},
		},
		{
			// Check that global (true) overrides local option (false)
			opts: []indigo.EvalOption{indigo.DiscardFail(indigo.Discard), indigo.DiscardPass(true)},
			chk: func(r *indigo.Result) {
				if len(r.Results) != 0 {
					t.Errorf("expected 0, got %v", len(r.Results))
				}
			},
		},
		{
			// Check that global (true) overrides local option (false)
			prep: func(r *indigo.Rule) {
				r.Rules["D"].Rules["d2"].Expr = "true"
			},

			opts: []indigo.EvalOption{indigo.DiscardPass(true)},
			chk: func(r *indigo.Result) {
				if len(r.Results) != 2 {
					t.Errorf("expected 2, got %v", len(r.Results))
				} // should get B and E

				if r.Results["B"].ExpressionPass {
					t.Error("condition should be false")
				}
				if r.Results["E"].ExpressionPass {
					t.Error("condition should be false")
				}
			},
		},
		{
			// Check that global (FALSE) overrides local option (true)
			prep: func(r *indigo.Rule) {
				r.EvalOptions.DiscardPass = true
				r.Rules["B"].EvalOptions.DiscardPass = true
				r.Rules["D"].EvalOptions.DiscardPass = true
				r.Rules["E"].EvalOptions.DiscardPass = true
			},
			opts: []indigo.EvalOption{indigo.DiscardPass(false)},
			chk: func(r *indigo.Result) {
				if len(r.Results) != 3 {
					t.Errorf("expected 3, got %v", len(r.Results))
				}
			},
		},

		{
			prep: func(r *indigo.Rule) {
				r.Rules["D"].Rules["d2"].Expr = "true" // make d2 pass -> D passes
			},
			// Check that global (true) overrides local option (false)
			opts: []indigo.EvalOption{indigo.DiscardFail(indigo.Discard)},
			chk: func(r *indigo.Result) {
				if len(r.Results) != 1 {
					t.Errorf("expected 1, got %v", len(r.Results))
				}
				if !r.Results["D"].Pass {
					t.Error("condition should be true")
				}
			},
		},

		{
			// Check that global (FALSE) overrides local option (true)
			prep: func(r *indigo.Rule) {
				r.EvalOptions.DiscardFail = indigo.Discard
				r.Rules["B"].EvalOptions.DiscardFail = indigo.Discard
				r.Rules["D"].EvalOptions.DiscardFail = indigo.Discard
				r.Rules["E"].EvalOptions.DiscardFail = indigo.Discard
			},
			opts: []indigo.EvalOption{indigo.DiscardFail(indigo.KeepAll)},
			chk: func(r *indigo.Result) {
				if len(r.Results) != 3 {
					t.Errorf("expected 3, got %v", len(r.Results))
				}
			},
		},
	}

	e := indigo.NewEngine(newMockEvaluator())

	for _, c := range cases {
		r := makeRule()
		if c.prep != nil {
			c.prep(r)
		}
		err := e.Compile(r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := e.Eval(context.Background(), r, map[string]any{}, c.opts...)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		c.chk(result)
	}
}

// Test that Indigo stops evaluating rules after a timeout value has been reached
func TestTimeout(t *testing.T) {

	r := makeRule()
	m := newMockEvaluator()
	m.evalDelay = 10 * time.Millisecond
	e := indigo.NewEngine(m)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := e.Eval(ctx, r, map[string]any{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Error("expected context.DeadlineExceeded error")
	}
}

// Test that using both sortFunc and parallel options results in an error
func TestSortFuncAndParallelIncompatible(t *testing.T) {
	e := indigo.NewEngine(newMockEvaluator())
	r := makeRule()

	// Set up a rule with both sortFunc and parallel options
	r.EvalOptions.SortFunc = indigo.SortRulesAlpha

	err := e.Compile(r)
	if err != nil {
		t.Fatalf("unexpected error during compile: %v", err)
	}

	// Try to evaluate with parallel processing - this should fail
	_, err = e.Eval(context.Background(), r, map[string]any{}, indigo.Parallel(10, 2))
	if err == nil {
		t.Fatal("expected error when using both sortFunc and parallel options")
	}

	expectedError := "sortFunc and parallel options are incompatible"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("expected error to contain '%s', got: %v", expectedError, err)
	}

}

var activeEvals int32    // Counter for active mock evaluations
var maxActiveEvals int32 // Maximum number of active evaluations

type trackingMockEvaluator struct {
	evalDelay   time.Duration
	errorRuleID string // if non-empty, this rule ID will produce an error
	panicRuleID string // if non-empty, this rule ID will panic
}

func (m *trackingMockEvaluator) Compile(expr string, s indigo.Schema, resultType indigo.Type, collectDiagnostics, dryRun bool) (any, error) {
	// Simple mock compilation
	return "compiled:" + expr, nil
}

func (m *trackingMockEvaluator) Evaluate(data map[string]any, expr string, s indigo.Schema, self any, evalData any, resultType indigo.Type, returnDiagnostics bool) (any, *indigo.Diagnostics, error) {
	atomic.AddInt32(&activeEvals, 1)
	if activeEvals > maxActiveEvals {
		maxActiveEvals = activeEvals
	}
	defer atomic.AddInt32(&activeEvals, -1)

	if m.panicRuleID != "" && expr == m.panicRuleID {
		panic(fmt.Sprintf("mock panic for rule %s", expr))
	}

	if m.evalDelay > 0 {
		time.Sleep(m.evalDelay)
	}
	if strings.Contains(expr, "Slow") {
		time.Sleep(50 * time.Millisecond)
	}
	if m.errorRuleID != "" && expr == m.errorRuleID {
		return false, nil, errors.New("mock error for rule " + expr)
	}
	return true, nil, nil
}

func TestParallelEvalLeakOnEarlyError(t *testing.T) {
	startGoroutines := runtime.NumGoroutine()
	atomic.StoreInt32(&activeEvals, 0)
	mockEval := &trackingMockEvaluator{
		evalDelay:   100 * time.Millisecond,
		errorRuleID: "ruleError",
	}
	engine := indigo.NewEngine(mockEval)

	rootRule := &indigo.Rule{
		ID: "root",
		Rules: map[string]*indigo.Rule{ // Changed to map to match typical Rule structure
			"ruleSlow1": {ID: "ruleSlow1", Expr: "ruleSlow1"}, // Will be slow
			"ruleSlow2": {ID: "ruleSlow2", Expr: "ruleSlow2"}, // Will be slow
			"ruleSlow3": {ID: "ruleSlow3", Expr: "ruleSlow3"}, // Will be slow
			"ruleSlow4": {ID: "ruleSlow4", Expr: "ruleSlow4"}, // Will be slow
			"ruleError": {ID: "ruleError", Expr: "ruleError"}, // Will error out quickly
			"ruleSlow5": {ID: "ruleSlow5", Expr: "ruleSlow5"}, // Will be slow
			"ruleSlow6": {ID: "ruleSlow6", Expr: "ruleSlow6"}, // Will be slow
			"ruleSlow7": {ID: "ruleSlow7", Expr: "ruleSlow7"}, // Will be slow
			"ruleSlow8": {ID: "ruleSlow8", Expr: "ruleSlow8"}, // Will be slow
		},
		// EvalOptions on the rule itself are used if not overridden by Eval call options
	}
	err := engine.Compile(rootRule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Test timeout
	defer cancel()

	batchSize := 2
	parallel := 2

	// Pass Parallel options directly to Eval
	_, evalErr := engine.Eval(ctx, rootRule, map[string]any{}, indigo.Parallel(batchSize, parallel))
	if evalErr == nil {
		t.Fatal("expected an error but got none")
	}
	if !strings.Contains(evalErr.Error(), "mock error for rule ruleError") {
		t.Errorf("expected error to contain 'mock error for rule ruleError', got: %v", evalErr)
	}
	//fmt.Println("Num goroutines: ", runtime.NumGoroutine())
	// Give some time for other goroutines to potentially get stuck or finish
	time.Sleep(mockEval.evalDelay * 3)

	if active := atomic.LoadInt32(&activeEvals); active != 0 {
		t.Errorf("Expected no active evaluations after early error, got %d", active)
	}

	if runtime.NumGoroutine() > startGoroutines+1 {
		t.Errorf("Expected no goroutines to be created, got %d", runtime.NumGoroutine())
	}
	if maxActiveEvals > int32(parallel) {
		t.Errorf("Expected max active evals to be %d, got %d", parallel, maxActiveEvals)
	}

	// fmt.Println("Num goroutines: ", runtime.NumGoroutine())
	// fmt.Println("Max active evals: ", maxActiveEvals)

}

func TestParallelEvalLeakOnContextCancel(t *testing.T) {
	atomic.StoreInt32(&activeEvals, 0)
	mockEval := &trackingMockEvaluator{
		evalDelay: 200 * time.Millisecond, // Make it longer so cancellation is more likely to interrupt
	}
	engine := indigo.NewEngine(mockEval)

	rootRule := &indigo.Rule{
		ID: "root",
		Rules: map[string]*indigo.Rule{
			"ruleSlow1": {ID: "ruleSlow1", Expr: "ruleSlow1"},
			"ruleSlow2": {ID: "ruleSlow2", Expr: "ruleSlow2"},
			"ruleSlow3": {ID: "ruleSlow3", Expr: "ruleSlow3"},
		},
	}
	err := engine.Compile(rootRule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), mockEval.evalDelay/2)
	defer cancel()

	_, evalErr := engine.Eval(ctx, rootRule, map[string]any{}, indigo.Parallel(2, 3))
	if evalErr == nil {
		t.Fatal("expected an error but got none")
	}
	isContextError := errors.Is(evalErr, context.DeadlineExceeded) || errors.Is(evalErr, context.Canceled)
	if !isContextError {
		// It's possible a rule finished and returned an error just as context was cancelled.
		// The key is that activeEvals should go to 0.
		t.Logf("Eval error was not a direct context error, but: %v", evalErr)
	}

	time.Sleep(mockEval.evalDelay * 2)

	if active := atomic.LoadInt32(&activeEvals); active != 0 {
		t.Errorf("Expected no active evaluations after context cancellation, got %d", active)
	}
}

func TestParallelEvalPanicHandling(t *testing.T) {
	atomic.StoreInt32(&activeEvals, 0)
	mockEval := &trackingMockEvaluator{
		evalDelay:   50 * time.Millisecond,
		panicRuleID: "rulePanic",
	}
	engine := indigo.NewEngine(mockEval)

	rootRule := &indigo.Rule{
		ID: "root",
		Rules: map[string]*indigo.Rule{
			"ruleOK1":   {ID: "ruleOK1", Expr: "ruleOK1"},
			"rulePanic": {ID: "rulePanic", Expr: "rulePanic"}, // This rule will panic
			"ruleOK2":   {ID: "ruleOK2", Expr: "ruleOK2"},
		},
	}
	err := engine.Compile(rootRule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Test timeout
	defer cancel()

	_, evalErr := engine.Eval(ctx, rootRule, map[string]any{}, indigo.Parallel(2, 3))
	if evalErr == nil {
		t.Fatal("expected an error but got none")
	}
	if !strings.Contains(evalErr.Error(), "panic during parallel rule evaluation: mock panic for rule rulePanic") {
		t.Errorf("expected error to contain 'panic during parallel rule evaluation: mock panic for rule rulePanic', got: %v", evalErr)
	}

	time.Sleep(mockEval.evalDelay * 3)
	if active := atomic.LoadInt32(&activeEvals); active != 0 {
		t.Errorf("Expected no active evaluations after panic, got %d", active)
	}
}

// Test that EvalOptions passed to Eval can override the parallel evaluation options
// set on a rule, forcing sequential evaluation instead of parallel evaluation
func TestParallelEvalOptionOverride(t *testing.T) {
	atomic.StoreInt32(&activeEvals, 0)
	atomic.StoreInt32(&maxActiveEvals, 0)

	mockEval := &trackingMockEvaluator{
		evalDelay: 100 * time.Millisecond,
	}
	engine := indigo.NewEngine(mockEval)

	// Create a rule that wants to evaluate its children in parallel
	rootRule := &indigo.Rule{
		ID: "root",
		EvalOptions: indigo.EvalOptions{
			Parallel: indigo.ParallelConfig{
				BatchSize:   2,
				MaxParallel: 3,
			},
		},
		Rules: map[string]*indigo.Rule{
			"rule1": {ID: "rule1", Expr: "true"},
			"rule2": {ID: "rule2", Expr: "true"},
			"rule3": {ID: "rule3", Expr: "true"},
			"rule4": {ID: "rule4", Expr: "true"},
			"rule5": {ID: "rule5", Expr: "true"},
		},
	}

	err := engine.Compile(rootRule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Track initial goroutine count
	initialGoroutines := runtime.NumGoroutine()

	// Override the rule's parallel settings with sequential evaluation
	// (BatchSize=0 and MaxParallel=0 forces sequential evaluation)
	_, evalErr := engine.Eval(ctx, rootRule, map[string]any{}, indigo.Parallel(0, 0))
	if evalErr != nil {
		t.Fatalf("unexpected error: %v", evalErr)
	}

	// Verify that the number of goroutines did not increase significantly
	// (allowing for some variance due to Go runtime)
	finalGoroutines := runtime.NumGoroutine()
	if finalGoroutines > initialGoroutines+2 {
		t.Errorf("Expected sequential evaluation with minimal goroutine increase, initial: %d, final: %d", 
			initialGoroutines, finalGoroutines)
	}

	// Verify that the maximum number of concurrent evaluations was limited
	// In sequential mode, we should never have more than 1 active evaluation at a time
	if maxActiveEvals > 1 {
		t.Errorf("Expected max 1 active evaluation in sequential mode, got %d", maxActiveEvals)
	}
}
