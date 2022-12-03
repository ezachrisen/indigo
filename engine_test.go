package indigo_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	"github.com/matryer/is"
)

// Test the scenario where the rule has childs and exists (or not) childs with true as their results
func TestTrueIfAnyBehavior(t *testing.T) {
	is := is.New(t)

	engine := indigo.NewEngine(cel.NewEvaluator())
	data := map[string]interface{}{}
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
		ruleL1_1.Expr = fmt.Sprintf("%t", l1) // assign the result of an Exp for the child rule
		ruleL2.Expr = fmt.Sprintf("%t", l2)

		expectedResults[ruleL2.ID] = l2
		expectedResults[ruleL1_1.ID] = l1       // change the expected value of the leaf
		expectedResults[ruleL1.ID] = l1         // until the root
		expectedResults[rootRule.ID] = l2 || l1 // since the leaf rule will propagate the change

		err := engine.Compile(rootRule)
		is.NoErr(err)

		result, err := engine.Eval(ctx, rootRule, data)
		is.NoErr(err)

		// verify if matches the expected result
		is.NoErr(match(flattenResultsRuleResult(result), expectedResults))
	}

	// possible scenarios for the leaf change
	for _, expect := range []bool{false, true} {
		check(expect, false) // if the L2 is false, then we need to check from leaf until the root
		check(expect, true)  // otherwise, the L2 will make the root to be true
	}

}

// Test that all rules are evaluated and yield the correct result in the default configuration
func TestEvaluationTraversalDefault(t *testing.T) {
	is := is.New(t)

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
	is.NoErr(err)

	result, err := e.Eval(context.Background(), r, map[string]interface{}{})
	is.NoErr(err)
	//	fmt.Println(m.rulesTested)
	fmt.Println(result)
	fmt.Println(result.SummaryCompact())
	js, err := json.MarshalIndent(result.SummaryCompact(), "", "  ")
	if err == nil {
		fmt.Println(string(js))
	}
	is.NoErr(match(flattenResultsExprResult(result), expectedResults))
}

// Ensure that rules are evaluated in the correct order when
// alpha sort is applied
func TestEvaluationTraversalAlphaSort(t *testing.T) {
	is := is.New(t)

	e := indigo.NewEngine(newMockEvaluator())
	r := makeRule()

	// Specify the sort order for all rules
	err := indigo.ApplyToRule(r, func(r *indigo.Rule) error {
		r.EvalOptions.SortFunc = sortRulesAlpha
		r.EvalOptions.StopFirstNegativeChild = true
		return nil
	})
	is.NoErr(err)
	err = indigo.ApplyToRule(r, func(r *indigo.Rule) error {
		r.Expr = "true"
		return nil
	})
	is.NoErr(err)

	err = e.Compile(r)
	is.NoErr(err)

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

	//If everything works, the rules were evaluated in this order
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

	result, err := e.Eval(context.Background(), r, map[string]interface{}{}, indigo.ReturnDiagnostics(true))
	is.NoErr(err)
	//	fmt.Println(m.rulesTested)
	//fmt.Println(result)
	is.NoErr(match(flattenResultsExprResult(result), expectedResults))
	// fmt.Printf("Expected: %+v\n", expectkedOrder)
	// fmt.Printf("Got     : %+v\n", flattenResultsEvaluated(result))
	is.True(reflect.DeepEqual(expectedOrder, flattenResultsEvaluated(result))) // not all rules were evaluated
}

// Test that a self reference is passed through compilation, evaluation
// and finally returned in results
func TestSelf(t *testing.T) {
	is := is.New(t)

	e := indigo.NewEngine(newMockEvaluator())
	r := makeRule()
	err := e.Compile(r)
	is.NoErr(err)

	// Set the self reference on D
	D := r.Rules["D"]
	D.Self = 22
	D.Expr = "self"
	d1 := D.Rules["d1"]
	// Give d1 a self expression, but no self value
	d1.Expr = "self"
	_, err = e.Eval(context.Background(), r, nil)
	is.True(err != nil) // should get an error if the data map is nil and we try to use 'self'

	result, err := e.Eval(context.Background(), r, map[string]interface{}{"anything": "anything"})
	is.NoErr(err)
	is.Equal(result.Results["D"].Value.(int), 22)                     // D should return 'self', which is 22
	is.Equal(result.Results["D"].Results["d1"].ExpressionPass, false) // d1 should not inherit D's self
}

// Test that the engine checks for nil data and rule
func TestNilDataOrRule(t *testing.T) {
	is := is.New(t)
	e := indigo.NewEngine(newMockEvaluator())
	r := makeRule()

	_, err := e.Eval(context.Background(), r, nil)
	is.True(err != nil) // should get an error if the data map is nil
	is.True(strings.Contains(err.Error(), "data is nil"))

	_, err = e.Eval(context.Background(), nil, map[string]interface{}{})
	is.True(err != nil) // should get an error if the rule is nil
	is.True(strings.Contains(err.Error(), "rule is nil"))

	r.Rules["B"].Rules["oops"] = nil
	_, err = e.Eval(context.Background(), r, map[string]interface{}{})
	is.True(err != nil) // should get an error if the rule is nil
	is.True(strings.Contains(err.Error(), "rule is nil"))

}

// Test the pass/fail of the expression evaluation with various combinations
// of evaluation options
// This tests the result.ExpressionPass field.
func TestEvalOptionsExpressionPassFail(t *testing.T) {
	is := is.New(t)

	e := indigo.NewEngine(newMockEvaluator())
	d := map[string]interface{}{"a": "a"} // dummy data, not important
	w := map[string]bool{                 // the wanted expression evaluation results with no options in effect
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
		"DiscardFail": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.DiscardFail = true
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b2", "b4", "b4-1", "b4-2") // b4-1 is elim. because b4 is
			},
		},
		"DiscardPass & DiscardFail": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.DiscardFail = true
				r.Rules["B"].EvalOptions.DiscardPass = true
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b1", "b2", "b3", "b4", "b4-1", "b4-2")
			},
		},
		"DiscardPass & DiscardFail on Root": {
			prep: func(r *indigo.Rule) {
				r.EvalOptions.DiscardFail = true
				r.EvalOptions.DiscardPass = true
			},
			want: func() map[string]bool {
				return map[string]bool{"rule1": true}
			},
		},
		"StopFirstPositiveChild": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.StopFirstPositiveChild = true
				r.Rules["B"].EvalOptions.SortFunc = sortRulesAlpha
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b2", "b3", "b4", "b4-1", "b4-2")
			},
		},
		"StopFirstNegativeChild": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.StopFirstNegativeChild = true
				r.Rules["B"].EvalOptions.SortFunc = sortRulesAlpha
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b3", "b4", "b4-1", "b4-2")
			},
		},
		"StopFirstNegativeChild & StopFirstPositiveChild": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.StopFirstNegativeChild = true
				r.Rules["B"].EvalOptions.StopFirstPositiveChild = true
				r.Rules["B"].EvalOptions.SortFunc = sortRulesAlpha
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
		r := makeRule()
		c.prep(r)

		u, err := e.Eval(context.Background(), r, d, indigo.ReturnDiagnostics(true))
		is.NoErr(err)
		err = match(flattenResultsExprResult(u), c.want())
		if err != nil {
			t.Errorf("Error in case %s: %v", k, err)
		}
	}
}

// Test the pass/fail of the rule  evaluation with various combinations
// of evaluation options
// This tests the result.Pass field
func TestEvalOptionsRulePassFail(t *testing.T) {
	is := is.New(t)

	e := indigo.NewEngine(newMockEvaluator())
	d := map[string]interface{}{"a": "a"} // dummy data, not important
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
		"DiscardFail": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.DiscardFail = true
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b2", "b4", "b4-1", "b4-2") // b4-1 is elim. because b4 is
			},
		},
		"DiscardPass & DiscardFail": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.DiscardFail = true
				r.Rules["B"].EvalOptions.DiscardPass = true
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b1", "b2", "b3", "b4", "b4-1", "b4-2")
			},
		},
		"DiscardPass & DiscardFail on Root": {
			prep: func(r *indigo.Rule) {
				r.EvalOptions.DiscardFail = true
				r.EvalOptions.DiscardPass = true
			},
			want: func() map[string]bool {
				return map[string]bool{"rule1": false}
			},
		},
		"StopFirstPositiveChild": {
			prep: func(r *indigo.Rule) {
				r.Rules["B"].EvalOptions.StopFirstPositiveChild = true
				r.Rules["B"].EvalOptions.SortFunc = sortRulesAlpha
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
				r.Rules["B"].EvalOptions.SortFunc = sortRulesAlpha
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
				r.Rules["B"].EvalOptions.SortFunc = sortRulesAlpha
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
				r.Rules["E"].EvalOptions.SortFunc = sortRulesAlpha
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

	// r := makeRule()
	// r.Rules["D"].Rules["d2"].Expr = `true` // this will make D true
	// r.Rules["B"].Expr = `true`             // this will not make B true, since its children are false
	// u, _ := e.Eval(context.Background(), r, d, indigo.ReturnDiagnostics(true))
	// fmt.Println(u)

	for k, c := range cases {
		r := makeRule()

		// In the rule created by makeRule, ALL children of rule1 are
		// false, which would make this test not very interesting.
		// We'll manipulate a few of the rules to make them more interesting to test.

		r.Rules["D"].Rules["d2"].Expr = `true` // this will make D true
		r.Rules["B"].Expr = `true`             // this will not make B true, since its children are false

		// Modified results
		// ┌───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
		// │                                                                                                                       │
		// │ INDIGO RESULT SUMMARY                                                                                                 │
		// │                                                                                                                       │
		// ├────────────┬───────┬───────┬───────┬────────┬─────────────┬─────────────┬────────────┬────────────┬─────────┬─────────┤
		// │            │ Pass/ │ Expr. │ Chil- │ Output │ Diagnostics │ Stop If     │ Stop First │ Stop First │ Discard │ Discard │
		// │ Rule       │ Fail  │ Pass/ │ dren  │ Value  │ Available?  │ Parent Neg. │ Pos. Child │ Neg. Child │ Pass    │ Fail    │
		// │            │       │ Fail  │       │        │             │             │            │            │         │         │
		// ├────────────┼───────┼───────┼───────┼────────┼─────────────┼─────────────┼────────────┼────────────┼─────────┼─────────┤
		// │ rule1      │ FAIL  │ PASS  │ 3     │ true   │ yes         │             │            │            │         │         │
		// │   D        │ PASS  │ PASS  │ 3     │ true   │ yes         │             │            │            │         │         │
		// │     d1     │ PASS  │ PASS  │ 0     │ true   │ yes         │             │            │            │         │         │
		// │     d2     │ PASS  │ PASS  │ 0     │ true   │ yes         │             │            │            │         │         │
		// │     d3     │ PASS  │ PASS  │ 0     │ true   │ yes         │             │            │            │         │         │
		// │   B        │ FAIL  │ PASS  │ 4     │ true   │ yes         │             │            │            │         │         │
		// │     b1     │ PASS  │ PASS  │ 0     │ true   │ yes         │             │            │            │         │         │
		// │     b2     │ FAIL  │ FAIL  │ 0     │ false  │ yes         │             │            │            │         │         │
		// │     b3     │ PASS  │ PASS  │ 0     │ true   │ yes         │             │            │            │         │         │
		// │     b4     │ FAIL  │ FAIL  │ 2     │ false  │ yes         │             │            │            │         │         │
		// │       b4-1 │ PASS  │ PASS  │ 0     │ true   │ yes         │             │            │            │         │         │
		// │       b4-2 │ FAIL  │ FAIL  │ 0     │ false  │ yes         │             │            │            │         │         │
		// │   E        │ FAIL  │ FAIL  │ 3     │ false  │ yes         │             │            │            │         │         │
		// │     e3     │ PASS  │ PASS  │ 0     │ true   │ yes         │             │            │            │         │         │
		// │     e1     │ PASS  │ PASS  │ 0     │ true   │ yes         │             │            │            │         │         │
		// │     e2     │ FAIL  │ FAIL  │ 0     │ false  │ yes         │             │            │            │         │         │
		// └────────────┴───────┴───────┴───────┴────────┴─────────────┴─────────────┴────────────┴────────────┴─────────┴─────────┘

		c.prep(r)

		u, err := e.Eval(context.Background(), r, d, indigo.ReturnDiagnostics(true))

		is.NoErr(err)
		//		fmt.Println(u)

		// if k == "StopFirstPositiveChild" {
		// 	fmt.Println(indigo.DiagnosticsReport(u, nil))
		// }
		err = match(flattenResultsRuleResult(u), c.want())
		if err != nil {
			t.Errorf("Error in case %s: %v", k, err)
			// fmt.Println(r)
			// fmt.Println(u)
		}
	}
}

func TestEvalTrueIfAny(t *testing.T) {
	is := is.New(t)

	e := indigo.NewEngine(newMockEvaluator())
	d := map[string]interface{}{"a": "a"} // dummy data, not important
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
		"DiscardFail": {
			prep: func(r *indigo.Rule) {
				r.EvalOptions.TrueIfAny = true
				r.Rules["B"].EvalOptions.TrueIfAny = true
				r.Rules["D"].EvalOptions.TrueIfAny = true
				r.Rules["E"].EvalOptions.TrueIfAny = true
				r.Rules["B"].EvalOptions.DiscardFail = true
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b2", "b4", "b4-1", "b4-2") // b4-1 is elim. because b4 is
			},
		},
		"DiscardPass & DiscardFail": {
			prep: func(r *indigo.Rule) {
				r.EvalOptions.TrueIfAny = true
				r.Rules["B"].EvalOptions.TrueIfAny = true
				r.Rules["D"].EvalOptions.TrueIfAny = true
				r.Rules["E"].EvalOptions.TrueIfAny = true

				r.Rules["B"].EvalOptions.DiscardFail = true
				r.Rules["B"].EvalOptions.DiscardPass = true
			},
			want: func() map[string]bool {
				return deleteKeys(copyMap(w), "b1", "b2", "b3", "b4", "b4-1", "b4-2")
			},
		},
		"DiscardPass & DiscardFail on Root": {
			prep: func(r *indigo.Rule) {
				r.EvalOptions.TrueIfAny = true
				r.Rules["B"].EvalOptions.TrueIfAny = true
				r.Rules["D"].EvalOptions.TrueIfAny = true
				r.Rules["E"].EvalOptions.TrueIfAny = true

				r.EvalOptions.DiscardFail = true
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
				r.Rules["B"].EvalOptions.SortFunc = sortRulesAlpha
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
				r.Rules["B"].EvalOptions.SortFunc = sortRulesAlpha

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
		r := makeRule()

		c.prep(r)

		u, err := e.Eval(context.Background(), r, d, indigo.ReturnDiagnostics(true))
		is.NoErr(err)
		err = match(flattenResultsRuleResult(u), c.want())
		if err != nil {
			t.Errorf("Error in case %s: %v", k, err)
			fmt.Println(r)
			fmt.Println(u)
		}
	}
}

func TestDiagnosticOptions(t *testing.T) {

	is := is.New(t)
	m := newMockEvaluator()
	e := indigo.NewEngine(m)
	d := map[string]interface{}{"a": "a"} // dummy data, not important

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
		is.NoErr(err)

		u, err := e.Eval(context.Background(), r, d, indigo.ReturnDiagnostics(c.evalDiagnostics))
		is.NoErr(err)
		//fmt.Println(k)
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

	is := is.New(t)
	m := newMockEvaluator()
	e := indigo.NewEngine(m)
	// Set the mock engine to require that diagnostics must be turned on at compile time,
	// or not. This is a special feature of the mock engine, useful for testing.
	m.diagnosticCompileRequired = true
	d := map[string]interface{}{"a": "a"} // dummy data, not important
	r := makeRule()

	// first, compile all the rules WITHOUT diagnostics
	err := e.Compile(r)
	is.NoErr(err)

	// then, re-compile rule B, WITH diagnostics
	err = e.Compile(r.Rules["B"], indigo.CollectDiagnostics(true))
	is.NoErr(err)

	u, err := e.Eval(context.Background(), r, d, indigo.ReturnDiagnostics(true))
	is.NoErr(err)

	f := flattenResultsDiagnostics(u)

	// We expect that only the B rules will have diagnostics
	for k, v := range f {
		if k == "B" || k == "b1" || k == "b2" || k == "b3" || k == "b4" || k == "b4-1" || k == "b4-2" {
			is.True(v != nil)
		} else {
			if v != nil {
				fmt.Println("V = ", v, k)
			}
			is.True(v == nil)
		}
	}
}

func TestJSON(t *testing.T) {
	is := is.New(t)

	//	e := indigo.NewEngine(newMockEvaluator())
	r := makeRule()

	_, err := json.Marshal(r)
	is.NoErr(err)
	//	fmt.Println(string(b))

}

// Test options set at the time eval is called
// (options apply to the entire tree)
func TestGlobalEvalOptions(t *testing.T) {
	is := is.New(t)

	cases := []struct {
		prep func(*indigo.Rule)     // Edits to apply to the rule before evaluating
		opts []indigo.EvalOption    // Options to pass to evaluate
		chk  func(r *indigo.Result) // Function to check the results
	}{
		{
			// Check that global (true) overrides default (false)
			opts: []indigo.EvalOption{indigo.StopIfParentNegative(true)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 3)
				is.Equal(len(r.Results["D"].Results), 3)
				is.Equal(len(r.Results["E"].Results), 0)
				is.Equal(len(r.Results["B"].Results), 0)
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
				is.Equal(len(r.Results), 3)
				is.Equal(len(r.Results["D"].Results), 3)
				is.Equal(len(r.Results["E"].Results), 3)
				is.Equal(len(r.Results["B"].Results), 4)
			},
		},
		{
			// Check that global (true) overrides local option (false)
			opts: []indigo.EvalOption{indigo.StopFirstPositiveChild(true), indigo.SortFunc(sortRulesAlpha)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 2) // B is false, D is first positive child
				is.True(r.Results["D"].ExpressionPass)
				is.True(!r.Results["B"].ExpressionPass)
			},
		},
		{
			// Check that global (false) overrides local option (true)
			prep: func(r *indigo.Rule) {
				r.EvalOptions.StopFirstPositiveChild = true
				r.Rules["B"].EvalOptions.StopFirstPositiveChild = true
				r.Rules["D"].EvalOptions.StopFirstPositiveChild = true
				r.Rules["E"].EvalOptions.StopFirstPositiveChild = true
				r.EvalOptions.SortFunc = sortRulesAlpha
				r.Rules["B"].EvalOptions.SortFunc = sortRulesAlpha
				r.Rules["D"].EvalOptions.SortFunc = sortRulesAlpha
				r.Rules["E"].EvalOptions.SortFunc = sortRulesAlpha
			},

			opts: []indigo.EvalOption{indigo.StopFirstPositiveChild(false)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 3)
			},
		},

		{
			// Check that global (true) overrides local option (false)
			opts: []indigo.EvalOption{indigo.StopFirstNegativeChild(true), indigo.SortFunc(sortRulesAlpha)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 1) // B is first, should stop evaluation
				is.True(!r.Results["B"].ExpressionPass)
			},
		},
		{
			// Check that global (true) overrides local option (false)
			opts: []indigo.EvalOption{indigo.StopFirstNegativeChild(true), indigo.StopFirstPositiveChild(true), indigo.SortFunc(sortRulesAlpha)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 1) // B should stop it
				is.True(!r.Results["B"].ExpressionPass)
			},
		},
		{
			// Check that global (true) overrides local option (false)
			opts: []indigo.EvalOption{indigo.DiscardFail(true), indigo.DiscardPass(true)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 0)
			},
		},
		{
			// Check that global (true) overrides local option (false)
			prep: func(r *indigo.Rule) {
				r.Rules["D"].Rules["d2"].Expr = "true"
			},

			opts: []indigo.EvalOption{indigo.DiscardPass(true)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 2) // should get B and E

				is.True(!r.Results["B"].ExpressionPass)
				is.True(!r.Results["E"].ExpressionPass)
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
				is.Equal(len(r.Results), 3)
			},
		},

		{
			prep: func(r *indigo.Rule) {
				r.Rules["D"].Rules["d2"].Expr = "true" // make d2 pass -> D passes
			},
			// Check that global (true) overrides local option (false)
			opts: []indigo.EvalOption{indigo.DiscardFail(true)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 1)
				is.True(r.Results["D"].Pass)
			},
		},

		{
			// Check that global (FALSE) overrides local option (true)
			prep: func(r *indigo.Rule) {
				r.EvalOptions.DiscardFail = true
				r.Rules["B"].EvalOptions.DiscardFail = true
				r.Rules["D"].EvalOptions.DiscardFail = true
				r.Rules["E"].EvalOptions.DiscardFail = true
			},
			opts: []indigo.EvalOption{indigo.DiscardFail(false)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 3)
			},
		},
	}

	e := indigo.NewEngine(newMockEvaluator())

	for _, c := range cases {
		r := makeRule()
		if c.prep != nil {
			c.prep(r)
		}
		result, err := e.Eval(context.Background(), r, map[string]interface{}{}, c.opts...)
		is.NoErr(err)
		c.chk(result)
	}
}

// Test that Indigo stops evaluating rules after a timeout value has been reached
func TestTimeout(t *testing.T) {
	is := is.New(t)

	r := makeRule()
	m := newMockEvaluator()
	m.evalDelay = 10 * time.Millisecond
	e := indigo.NewEngine(m)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := e.Eval(ctx, r, map[string]interface{}{})
	is.True(errors.Is(err, context.DeadlineExceeded))
}
