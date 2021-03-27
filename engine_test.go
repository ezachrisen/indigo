package indigo_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/ezachrisen/indigo"
	"github.com/matryer/is"
)

// -------------------------------------------------- MOCK EVALUATOR
// mockEvaluator is used for testing
// It provides minimal evaluation of rules and captures
// information about which rules were processed, etc.
type mockEvaluator struct {
	rules       []string                      // a list of rule IDs in the evaluator
	rulesTested []string                      // a list of rule IDs that were evaluated
	ruleOptions map[string]indigo.EvalOptions // a copy of the evaluation options used for each rule
}

func newMockEvaluator() *mockEvaluator {
	return &mockEvaluator{
		ruleOptions: make(map[string]indigo.EvalOptions, 10),
	}
}
func (m *mockEvaluator) Compile(rule *indigo.Rule, collectDiagnostics bool, dryRun bool) error {
	m.rules = append(m.rules, rule.ID)
	return nil
}

func (m *mockEvaluator) ResetRulesTested() {
	m.rulesTested = []string{}
}

// The mockEvaluator only knows how to evaluate 1 string: `true`. If the expression is this, the evaluation is true, otherwise false.
func (m *mockEvaluator) Eval(data map[string]interface{}, rule *indigo.Rule, opt indigo.EvalOptions) (indigo.Value, string, error) {
	m.rulesTested = append(m.rulesTested, rule.ID)

	diagnostics := ""

	if opt.ReturnDiagnostics {
		diagnostics = "diagnostics here"
	}

	if rule.Expr == `true` {
		return indigo.Value{
			Val: true,
			Typ: indigo.Bool{},
		}, diagnostics, nil
	}

	if rule.Expr == `self` && rule.Self != nil {
		return indigo.Value{
			Val: rule.Self.(int),
			Typ: indigo.Int{},
		}, diagnostics, nil
	}

	return indigo.Value{
		Val: false,
		Typ: indigo.Bool{},
	}, diagnostics, nil
}

func (m *mockEvaluator) Reset() {
	m.rules = []string{}
}

func (e *mockEvaluator) PrintInternalStructure() {
	for _, v := range e.rules {
		fmt.Println("Rule id", v)
	}
}

// -------------------------------------------------- RULE CREATION HELPERS
// Make a rule that incldues a reference to a "self" value
func makeRuleWithSelf(id string) *indigo.Rule {

	return &indigo.Rule{
		ID:   id,
		Expr: `true`,
		Rules: map[string]*indigo.Rule{
			"a": &indigo.Rule{
				ID:   "a",
				Expr: `self`,
				Self: 22,
				Rules: map[string]*indigo.Rule{
					"a1": &indigo.Rule{
						ID:   "a1",
						Expr: `self`,
					},
				},
			},
		},
	}
}

// Make a nested rule tree where the rules
// do not have any evaluation options set locally
func makeRuleNoOptions() *indigo.Rule {
	rule1 := &indigo.Rule{
		ID:   "rule1",
		Expr: `true`,
		Rules: map[string]*indigo.Rule{
			"D": &indigo.Rule{
				ID:   "D",
				Expr: `true`,
				Rules: map[string]*indigo.Rule{
					"d1": {
						ID:   "d1",
						Expr: `true`,
					},
					"d2": {
						ID:   "d2",
						Expr: `false`,
					},
					"d3": {
						ID:   "d3",
						Expr: `true`,
					},
				},
			},
			"B": {
				ID:   "B",
				Expr: `false`,
				Rules: map[string]*indigo.Rule{
					"b1": {
						ID:   "b1",
						Expr: `true`,
					},
					"b2": {
						ID:   "b2",
						Expr: `false`,
					},
					"b3": {
						ID:   "b3",
						Expr: `true`,
					},
					"b4": {
						ID:   "b4",
						Expr: `false`,
						Rules: map[string]*indigo.Rule{
							"b4-1": {
								ID:   "b4-1",
								Expr: `true`,
							},
							"b4-2": {
								ID:   "b4-2",
								Expr: `false`,
							},
						},
					},
				},
			},
			"E": {
				ID:   "E",
				Expr: `false`,
				Rules: map[string]*indigo.Rule{
					"e1": {
						ID:   "e1",
						Expr: `true`,
					},
					"e2": {
						ID:   "e2",
						Expr: `false`,
					},
					"e3": {
						ID:   "e3",
						Expr: `true`,
					},
				},
			},
		},
	}
	return rule1
}

// Make a nested rule tree where some rules have local
// evaluation options set
func makeRuleWithOptions() *indigo.Rule {
	rule1 := &indigo.Rule{
		ID:   "rule1",
		Expr: `true`,
		Rules: map[string]*indigo.Rule{
			"D": {
				ID:   "D",
				Expr: `true`,
				Rules: map[string]*indigo.Rule{
					"d1": {
						ID:   "d1",
						Expr: `true`,
					},
					"d2": {
						ID:   "d2",
						Expr: `false`,
					},
					"d3": {
						ID:   "d3",
						Expr: `true`,
					},
				},
			},
			"B": {
				ID:       "B",
				Expr:     `false`,
				EvalOpts: []indigo.EvalOption{indigo.DiscardPass(true)},
				Rules: map[string]*indigo.Rule{
					"b1": {
						ID:   "b1",
						Expr: `true`,
					},
					"b2": {
						ID:   "b2",
						Expr: `false`,
					},
					"b3": {
						ID:   "b3",
						Expr: `true`,
					},
					"b4": {
						ID:   "b4",
						Expr: `false`,
						Rules: map[string]*indigo.Rule{
							"b4-1": {
								ID:   "b4-1",
								Expr: `true`,
							},
							"b4-2": {
								ID:   "b4-2",
								Expr: `false`,
							},
						},
					},
				},
			},
			"E": {
				ID:       "E",
				Expr:     `false`,
				EvalOpts: []indigo.EvalOption{indigo.StopIfParentNegative(true)},
				Rules: map[string]*indigo.Rule{
					"e1": {
						ID:   "e1",
						Expr: `true`,
					},
					"e2": {
						ID:   "e2",
						Expr: `false`,
					},
					"e3": {
						ID:   "e3",
						Expr: `true`,
					},
				},
			},
		},
	}
	return rule1
}

// Test that all rules are evaluated in the correct order in the default configuration
func TestEvaluationTraversalDefault(t *testing.T) {
	is := is.New(t)

	m := newMockEvaluator()
	e := indigo.NewEngine(m)

	rule1 := makeRuleNoOptions()
	err := e.Compile(rule1)
	is.NoErr(err)

	// fmt.Println(rule1.DescribeStructure())
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

	// If everything works, the rules were evaluated in this order
	// (alphabetically)
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

	result, err := e.Evaluate(nil, rule1)
	is.NoErr(err)
	// fmt.Println(m.rulesTested)
	// fmt.Println(indigo.SummarizeResults(result))
	is.Equal(result.RulesEvaluated, len(m.rulesTested))
	is.True(reflect.DeepEqual(expectedOrder, m.rulesTested)) // not all rules were evaluated
	is.True(match(flattenResults(result), expectedResults))
}

// Test that a self reference is passed through compilation, evaluation
// and finally returned in results
func TestSelf(t *testing.T) {
	is := is.New(t)

	m := newMockEvaluator()
	e := indigo.NewEngine(m)

	rule1 := makeRuleWithSelf("rule1")
	err := e.Compile(rule1)
	is.NoErr(err)

	result, err := e.Evaluate(nil, rule1)
	is.True(err != nil) // should get an error if the data map is nil and we try to use 'self'

	result, err = e.Evaluate(map[string]interface{}{"anything": "anything"}, rule1)
	is.NoErr(err)
	is.Equal(result.RulesEvaluated, 3)                      // rule1, a and a1
	is.Equal(result.Results["a"].Value.(int), 22)           // a should return 'self'
	is.Equal(result.Results["a"].Results["a1"].Pass, false) // a1 should not inherit a's self
}

// Test that the "stop negative parent" option is respected, and that the rules are evaluated in correct order
func TestEvaluationTraversalStopNegativeParent(t *testing.T) {
	is := is.New(t)

	m := newMockEvaluator()
	e := indigo.NewEngine(m)
	rule1 := makeRuleWithOptions()

	err := e.Compile(rule1)
	is.NoErr(err)

	expectedResults := map[string]bool{
		"rule1": true,
		"D":     true,
		"d1":    true,
		"d2":    false,
		"d3":    true,
		"B":     false, // DiscardPass: true
		// "b1": true,  // discard, since it's true
		"b2": false,
		// "b3": true,
		//		"b4": false,
		// Since B4 isn't returned, neither are its children
		// "b4-1": true,
		"b4-2": false,
		"E":    false, // StopIfParentNegative
		// "e1" : true // not returned since E is negative
		// "e2" : true // not returned since E is negative
		// "e3" : true // not returned since E is negative
	}

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
		// "e1", not evaluated because E==false with StopIfParentNegative option
		// "e2",
		// "e3",
	}

	result, err := e.Evaluate(nil, rule1)
	// fmt.Printf("expected     :%v\n", expectedOrder)
	// fmt.Printf("m.rulesTested:%v\n", m.rulesTested)

	fmt.Printf(indigo.SummarizeResults(result))
	is.NoErr(err)
	is.Equal(result.RulesEvaluated, len(m.rulesTested))
	is.True(reflect.DeepEqual(expectedOrder, m.rulesTested)) // not all rules were evaluated
	is.True(match(flattenResults(result), expectedResults))

}

func flattenResults(result *indigo.Result) map[string]bool {
	m := map[string]bool{}
	m[result.RuleID] = result.Pass

	for _, r := range result.Results {
		mc := flattenResults(r)
		for k, mcc := range mc {
			m[k] = mcc
		}
	}
	return m
}

func match(result map[string]bool, expected map[string]bool) bool {

	for k, v := range result {
		ok, ev := expected[k]
		if !ok {
			fmt.Println("received result for rule ", k, "; no result was expected")
			return false
		}

		if v != ev {
			fmt.Println("result mismatch: rule ", k, ", got ", v, ", expected ", ev)
			return false
		}
	}

	for k := range expected {
		if ok, _ := result[k]; !ok {
			fmt.Println("expected result for rule ", k, "; no result found")
			return false
		}
	}

	return true
}

// Test options set at the time eval is called
// (options apply to the entire tree)
func TestGlobalEvalOptions(t *testing.T) {
	is := is.New(t)

	cases := []struct {
		opts []indigo.EvalOption  // Options to pass to evaluate
		chk  func(*indigo.Result) // Function to check the results
	}{
		{
			opts: []indigo.EvalOption{indigo.StopIfParentNegative(true)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 3)
				is.Equal(len(r.Results["D"].Results), 3)
				is.Equal(len(r.Results["E"].Results), 0)
				is.Equal(len(r.Results["B"].Results), 0)
			},
		},
		{
			opts: []indigo.EvalOption{indigo.StopIfParentNegative(false)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 3)
				is.Equal(len(r.Results["D"].Results), 3)
				is.Equal(len(r.Results["E"].Results), 3)
				is.Equal(len(r.Results["B"].Results), 0)
			},
		},
		{
			opts: []indigo.EvalOption{indigo.StopFirstPositiveChild(true)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 2) // B is false, D is first positive child
				is.True(r.Results["D"].Pass)
				is.True(!r.Results["B"].Pass)
			},
		},
		{
			opts: []indigo.EvalOption{indigo.StopFirstNegativeChild(true)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 1) // B is first, should stop evaluation
				is.True(!r.Results["B"].Pass)
			},
		},
		{
			opts: []indigo.EvalOption{indigo.StopFirstNegativeChild(true), indigo.StopFirstPositiveChild(true)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 1) // B should stop it
				is.True(!r.Results["B"].Pass)
			},
		},
		{
			opts: []indigo.EvalOption{indigo.DiscardFail(true), indigo.DiscardPass(true)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 0)
			},
		},
		{
			opts: []indigo.EvalOption{indigo.DiscardPass(true)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 2)
			},
		},
		{
			opts: []indigo.EvalOption{indigo.DiscardFail(true), indigo.DiscardPass(false)},
			chk: func(r *indigo.Result) {

				is.Equal(len(r.Results), 1)
				is.True(r.Results["D"].Pass)
			},
		},
	}

	m := newMockEvaluator()
	e := indigo.NewEngine(m)
	r1 := makeRuleNoOptions()
	err := e.Compile(r1)
	is.NoErr(err)

	for _, c := range cases {
		result, err := e.Evaluate(nil, r1, c.opts...)
		is.NoErr(err)
		c.chk(result)
	}
}

// Test that eval options passed in via eval do not override the options
// set at the rule level.
func TestLocalEvalOptions(t *testing.T) {
	is := is.New(t)

	cases := []struct {
		opts []indigo.EvalOption  // Options to pass to evaluate
		chk  func(*indigo.Result) // Function to check the results
	}{
		{
			opts: []indigo.EvalOption{},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 3)                            // All top-level rules B, D and E
				is.Equal(len(r.Results["D"].Results), 3)               // D has no local restriction; want all children
				is.Equal(len(r.Results["B"].Results), 2)               // B discards PASS, want 2
				is.Equal(len(r.Results["B"].Results["b4"].Results), 1) // Ensure B's opts are inherited, only want b4-2 (false)
				is.Equal(len(r.Results["E"].Results), 0)               // E is negative, skip child rule

			},
		},
		{
			opts: []indigo.EvalOption{indigo.DiscardPass(true)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 2)                            // B and E only, since they're false
				is.Equal(len(r.Results["B"].Results), 2)               // Do not want true rules
				is.Equal(len(r.Results["B"].Results["b4"].Results), 1) // Ensure B's opts are inherited
				is.Equal(len(r.Results["E"].Results), 0)               // E is negative, skip child rule

			},
		},
		{
			opts: []indigo.EvalOption{indigo.DiscardPass(false)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 3)                            // B and E only
				is.Equal(len(r.Results["B"].Results["b4"].Results), 1) // Ensure B's opts are inherited
				is.Equal(len(r.Results["E"].Results), 0)               // E is negative, skip child rule

			},
		},
		{
			opts: []indigo.EvalOption{indigo.DiscardFail(false)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 3)                            // B, D and E
				is.Equal(len(r.Results["D"].Results), 3)               // Only want true rules
				is.Equal(len(r.Results["B"].Results), 2)               // Do not want true rules
				is.Equal(len(r.Results["B"].Results["b4"].Results), 1) // Ensure B's opts are inherited
				is.Equal(len(r.Results["E"].Results), 0)               // E is negative, skip child rule

			},
		},
		{
			opts: []indigo.EvalOption{indigo.DiscardFail(true)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 1)              // D only
				is.Equal(len(r.Results["D"].Results), 2) // Only want true rules
			},
		},
	}

	m := newMockEvaluator()
	e := indigo.NewEngine(m)
	r1 := makeRuleWithOptions()
	err := e.Compile(r1)
	is.NoErr(err)

	for _, c := range cases {
		r, err := e.Evaluate(nil, r1, c.opts...)
		is.NoErr(err)
		c.chk(r)
	}
}

func TestDiagnosticOptions(t *testing.T) {

	is := is.New(t)

	// Turn off diagnostic collection, but request it at eval time
	m := newMockEvaluator()
	engine := indigo.NewEngine(m, indigo.CollectDiagnostics(false))
	r1 := makeRuleNoOptions()
	err := engine.Compile(r1)
	is.NoErr(err)

	_, err = engine.Evaluate(nil, r1, indigo.ReturnDiagnostics(true))
	if err == nil {
		t.Errorf("Wanted error; should require indigo.CollectDiagnostics to be turned on to enable indigo.ReturnDiagnostics")
	}

	// Do not specify diagnostic collection (should be off)
	engine = indigo.NewEngine(m)
	r2 := makeRuleNoOptions()

	err = engine.Compile(r2)
	is.NoErr(err)

	r, err := engine.Evaluate(nil, r2)
	is.NoErr(err)
	is.Equal(r.RulesEvaluated, 10)

	for _, c := range r.Results {
		is.Equal(c.Diagnostics, "")
	}

	// Turn off diagnostic collection
	engine = indigo.NewEngine(m)
	r3 := makeRuleNoOptions()
	err = engine.Compile(r3)
	is.NoErr(err)

	r, err = engine.Evaluate(nil, r3, indigo.ReturnDiagnostics(false))
	is.NoErr(err)
	is.Equal(r.RulesEvaluated, 10)

	for _, c := range r.Results {
		is.Equal(c.Diagnostics, "")
	}

	// Turn on diagnostic collection
	engine = indigo.NewEngine(m, indigo.CollectDiagnostics(true))
	r4 := makeRuleNoOptions()
	err = engine.Compile(r4)
	is.NoErr(err)

	r, err = engine.Evaluate(nil, r4, indigo.ReturnDiagnostics(true))
	is.NoErr(err)
	is.Equal(r.RulesEvaluated, 10)
	is.Equal(len(r.Results), 3)
	is.Equal(len(r.Results["D"].Results), 3)
	is.Equal(len(r.Results["B"].Results), 0)
	is.Equal(len(r.Results["E"].Results), 3) // E is negative, skip child rule
	is.Equal(r.Results["B"].Pass, false)
	is.Equal(r.Results["D"].Pass, true)
	is.Equal(r.Results["E"].Pass, false)
	is.Equal(r.Results["E"].Results["e1"].Pass, true)
	is.Equal(r.Results["E"].Results["e2"].Pass, false)
	is.Equal(r.Results["E"].Results["e3"].Pass, true)

	for _, c := range r.Results {
		is.Equal(c.Diagnostics, "diagnostics here")
	}

}

// func TestConcurrency(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping test in short mode.")
// 	}

// 	is := is.New(t)
// 	rand.Seed(time.Now().Unix())

// 	m := indigo.NoOpEvaluator{}
// 	e := indigo.NewEngine(m)

// 	var wg sync.WaitGroup

// 	for i := 1; i < 50_000; i++ {
// 		wg.Add(1)
// 		go func(i int) {
// 			defer wg.Done()
// 			r :=
// 			err := e.AddRule("/", makeRuleWithID(fmt.Sprintf("rule%d", i)))
// 			is.NoErr(err)
// 			r, err := e.Evaluate(nil, fmt.Sprintf("rule%d", i), indigo.ReturnDiagnostics(false))
// 			is.NoErr(err)
// 			is.Equal(r.RulesEvaluated, 10)
// 		}(i)
// 		time.Sleep(time.Duration(rand.Intn(3) * int(time.Millisecond)))
// 	}

// 	wg.Wait()
// }
