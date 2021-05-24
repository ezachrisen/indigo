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
	"github.com/matryer/is"
)

// Test that all rules are evaluated and yield the correct result in the default configuration
func TestEvaluationTraversalDefault(t *testing.T) {
	is := is.New(t)

	e := indigo.NewEngine(newMockEvaluator())
	r := makeRule()

	//fmt.Println(r)
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
	//	is.True(err != nil)

	result, err := e.Eval(context.Background(), r, map[string]interface{}{})
	is.NoErr(err)
	//fmt.Println(m.rulesTested)
	//fmt.Println(result)
	is.NoErr(match(flattenResults(result), expectedResults))
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
		return nil
	})

	if err != nil {
		t.Error(err)
	}

	err = e.Compile(r)
	is.NoErr(err)

	//	fmt.Println(r)
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
	is.NoErr(match(flattenResults(result), expectedResults))
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
	is.Equal(result.Results["D"].Value.(int), 22)           // D should return 'self', which is 22
	is.Equal(result.Results["D"].Results["d1"].Pass, false) // d1 should not inherit D's self
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

// Test the rule evaluation with various options set on the rules
func TestEvalOptions(t *testing.T) {
	is := is.New(t)

	e := indigo.NewEngine(newMockEvaluator())
	d := map[string]interface{}{"a": "a"} // dummy data, not important
	w := map[string]bool{                 // the wanted rule evaluation results with no options in effect
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
				return deleteKeys(copyMap(w), "b2", "b4", "b4-1", "b4-2")
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

		u, err := e.Eval(context.Background(), r, d)
		is.NoErr(err)
		//		fmt.Println(r)
		//		fmt.Println(u)
		err = match(flattenResults(u), c.want())
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
		wantDiagnostics                 bool // whether we expect to received diagnostic information
	}{
		"No diagnostics requested at compile or eval time": {
			engineDiagnosticCompileRequired: true,
			compileDiagnostics:              false,
			evalDiagnostics:                 false,
			wantDiagnostics:                 false,
		},
		"No diagnostics requested at compile time, but NOT at eval time": {
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

		switch c.wantDiagnostics {
		case true:
			err = allNotEmpty(flattenResultsDiagnostics(u))
			if err != nil {
				t.Errorf("In case '%s', wanted diagnostics: %v", k, err)
			}
			if len(flattenResultsEvaluated(u)) != 16 {
				t.Errorf("In case '%s', wanted list of rules diagnostics", k)
			}

		default:
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

// // Test options set at the time eval is called
// // (options apply to the entire tree)
// func TestGlobalEvalOptions(t *testing.T) {
// 	is := is.New(t)

// 	cases := []struct {
// 		opts []indigo.EvalOption  // Options to pass to evaluate
// 		chk  func(*indigo.Result) // Function to check the results
// 	}{
// 		{
// 			opts: []indigo.EvalOption{indigo.StopIfParentNegative(true)},
// 			chk: func(r *indigo.Result) {
// 				is.Equal(len(r.Results), 3)
// 				is.Equal(len(r.Results["D"].Results), 3)
// 				is.Equal(len(r.Results["E"].Results), 0)
// 				is.Equal(len(r.Results["B"].Results), 0)
// 			},
// 		},
// 		{
// 			opts: []indigo.EvalOption{indigo.StopIfParentNegative(false)},
// 			chk: func(r *indigo.Result) {
// 				is.Equal(len(r.Results), 3)
// 				is.Equal(len(r.Results["D"].Results), 3)
// 				is.Equal(len(r.Results["E"].Results), 3)
// 				is.Equal(len(r.Results["B"].Results), 4)
// 			},
// 		},
// 		{
// 			opts: []indigo.EvalOption{indigo.StopFirstPositiveChild(true), indigo.SortFunc(sortRulesAlpha)},
// 			chk: func(r *indigo.Result) {
// 				is.Equal(len(r.Results), 2) // B is false, D is first positive child
// 				is.True(r.Results["D"].Pass)
// 				is.True(!r.Results["B"].Pass)
// 			},
// 		},
// 		{
// 			opts: []indigo.EvalOption{indigo.StopFirstNegativeChild(true), indigo.SortFunc(sortRulesAlpha)},
// 			chk: func(r *indigo.Result) {
// 				is.Equal(len(r.Results), 1) // B is first, should stop evaluation
// 				is.True(!r.Results["B"].Pass)
// 			},
// 		},
// 		{
// 			opts: []indigo.EvalOption{indigo.StopFirstNegativeChild(true), indigo.StopFirstPositiveChild(true), indigo.SortFunc(sortRulesAlpha)},
// 			chk: func(r *indigo.Result) {
// 				is.Equal(len(r.Results), 1) // B should stop it
// 				is.True(!r.Results["B"].Pass)
// 			},
// 		},
// 		{
// 			opts: []indigo.EvalOption{indigo.DiscardFail(true), indigo.DiscardPass(true)},
// 			chk: func(r *indigo.Result) {
// 				is.Equal(len(r.Results), 0)
// 			},
// 		},
// 		{
// 			opts: []indigo.EvalOption{indigo.DiscardPass(true)},
// 			chk: func(r *indigo.Result) {
// 				is.Equal(len(r.Results), 2) // should get B and E
// 				is.True(!r.Results["B"].Pass)
// 				is.True(!r.Results["E"].Pass)
// 			},
// 		},
// 		{
// 			opts: []indigo.EvalOption{indigo.DiscardFail(true)},
// 			chk: func(r *indigo.Result) {
// 				is.Equal(len(r.Results), 1)
// 				is.True(r.Results["D"].Pass)
// 			},
// 		},
// 	}

// 	e := indigo.NewEngine(newMockEvaluator())
// 	r := makeRule()

// 	for _, c := range cases {
// 		result, err := e.Eval(context.Background(), r, map[string]interface{}{}, c.opts...)
// 		is.NoErr(err)
// 		c.chk(result)
// 	}
// }

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
				is.True(r.Results["D"].Pass)
				is.True(!r.Results["B"].Pass)
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
				is.True(!r.Results["B"].Pass)
			},
		},
		{
			// Check that global (true) overrides local option (false)
			opts: []indigo.EvalOption{indigo.StopFirstNegativeChild(true), indigo.StopFirstPositiveChild(true), indigo.SortFunc(sortRulesAlpha)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 1) // B should stop it
				is.True(!r.Results["B"].Pass)
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
			opts: []indigo.EvalOption{indigo.DiscardPass(true)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 2) // should get B and E
				is.True(!r.Results["B"].Pass)
				is.True(!r.Results["E"].Pass)
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
