package indigo_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ezachrisen/indigo"
	"github.com/matryer/is"
)

type MockEvaluator struct {
	rules       []string                      // a list of rule IDs in the evaluator
	rulesTested []string                      // a list of rule IDs that were evaluated
	ruleOptions map[string]indigo.EvalOptions // a copy of the evaluation options used for each rule
}

func NewMockEvaluator() *MockEvaluator {
	return &MockEvaluator{
		ruleOptions: make(map[string]indigo.EvalOptions, 10),
	}
}
func (m *MockEvaluator) Compile(rule *indigo.Rule, collectDiagnostics bool, dryRun bool) error {
	m.rules = append(m.rules, rule.ID)
	return nil
}

func (m *MockEvaluator) ResetRulesTested() {
	m.rulesTested = []string{}
}

// The MockEvaluator only knows how to evaluate 1 string: `true`. If the expression is this, the evaluation is true, otherwise false.
func (m *MockEvaluator) Eval(data map[string]interface{}, rule *indigo.Rule, opt indigo.EvalOptions) (indigo.Value, string, error) {
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

	return indigo.Value{
		Val: false,
		Typ: indigo.Bool{},
	}, diagnostics, nil
}

func (m *MockEvaluator) Reset() {
	m.rules = []string{}
}

func (e *MockEvaluator) PrintInternalStructure() {
	for _, v := range e.rules {
		fmt.Println("Rule id", v)
	}
}

func TestAddRules(t *testing.T) {
	is := is.New(t)

	m := NewMockEvaluator()
	e := indigo.NewEngine(m)

	err := e.AddRule([]*indigo.Rule{}...)
	is.NoErr(err) // Calling AddRule with an empty list of rules is OK

	err = e.AddRule(&indigo.Rule{})
	if err == nil {
		is.Fail() // Adding a rule without ID should fail
	}

	err = e.AddRule(&indigo.Rule{ID: "ID/SOMETHING"})
	if err == nil {
		is.Fail() // Adding a rule with a / should fail
	}

	err = e.AddRule(&indigo.Rule{ID: "A"})
	is.NoErr(err) // Only an ID is required for a valid rule

	_, ok := e.Rules()["A"]
	is.True(ok) // The rule should be in the engine

	ruleB := indigo.Rule{
		ID:   "B",
		Meta: "B1",
		Rules: map[string]*indigo.Rule{
			"B": &indigo.Rule{
				ID:   "B",
				Meta: "B2",
			},
			"C": &indigo.Rule{
				ID:   "C",
				Meta: "C2",
			},
		},
	}

	err = e.AddRule(&ruleB)
	is.NoErr(err)

	is.Equal(e.Rules()["B"].Meta, "B1") // B1 should not be overwritten by its child with same ID

}

func makeRuleWithID(id string) *indigo.Rule {
	rule1 := &indigo.Rule{
		ID:   id,
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

func makeRuleWithOptions() *indigo.Rule {
	rule1 := &indigo.Rule{
		ID:   "rule1",
		Expr: `true`,
		Rules: map[string]*indigo.Rule{
			"D": {
				ID:       "D",
				Expr:     `true`,
				EvalOpts: []indigo.EvalOption{indigo.ReturnFail(false)},
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
				EvalOpts: []indigo.EvalOption{indigo.ReturnPass(false)},
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

func inArray(a []string, s string) bool {
	for _, v := range a {
		if s == v {
			return true
		}
	}
	return false
}

// Test that all rules are evaluated in the correct order in the default configuration
func TestRuleAccess(t *testing.T) {
	is := is.New(t)

	m := NewMockEvaluator()
	e := indigo.NewEngine(m)

	e.AddRule(makeRuleWithOptions())

	r1, ok := e.Rule("rule1")
	is.True(ok)
	is.Equal(r1.ID, "rule1")

	_, r11, ok := e.RuleWithPath("rule1")
	is.True(ok)
	is.Equal(r11.ID, "rule1")

	b4, b41, ok := e.RuleWithPath("rule1/B/b4/b4-1")
	is.True(ok)
	is.Equal(b4.ID, "b4")
	is.Equal(b41.ID, "b4-1")

	// separate := makeRuleWithOptions()

	// b41x, ok := separate.FindChild("B/b4/b4-1")
	// is.True(ok)
	// is.Equal(b41x.ID, "b4-1")

}

func TestRuleWithPath(t *testing.T) {
	is := is.New(t)

	m := NewMockEvaluator()
	e := indigo.NewEngine(m)

	e.AddRule(makeRuleWithOptions())

	_, b41, ok := e.RuleWithPath("rule1/B/b4/b4-1")
	is.True(ok)
	is.Equal(b41.ID, "b4-1")
	is.Equal(b41.Expr, "true")
	// b41.Expr = "updated"
	// is.Equal(b41.Expr, "updated")

	// b41x, ok := e.RuleWithPath("rule1/B/b4/b4-1")
	// is.True(ok)
	// is.Equal(b41x.ID, "b4-1")
	// is.Equal(b41x.Expr, "updated")

}

// Test that all rules are evaluated in the correct order in the default configuration
func TestEvaluationTraversalDefault(t *testing.T) {
	is := is.New(t)

	m := NewMockEvaluator()
	e := indigo.NewEngine(m)

	e.AddRule(makeRuleNoOptions())

	expectedResults := map[string]bool{
		"rule1": true,
		"D":     true,
		"d1":    true,
		"d2":    false,
		"d3":    true,
		"B":     false,
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
		"D",
		"d1",
		"d2",
		"d3",
		"E",
		"e1",
		"e2",
		"e3",
	}

	result, err := e.Evaluate(nil, "rule1")
	is.NoErr(err)

	is.Equal(result.RulesEvaluated, len(m.rulesTested))
	is.True(reflect.DeepEqual(expectedOrder, m.rulesTested)) // not all rules were evaluated
	is.True(match(result, expectedResults))
}

func makeRulesAndExpectedResult() (*indigo.Rule, map[string]bool) {
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

	expectedResults := map[string]bool{
		"rule1": true,
		"D":     true,
		"d1":    true,
		"d2":    false,
		"d3":    true,
		"B":     false,
		"E":     false,
		"e1":    true,
		"e2":    false,
		"e3":    true,
	}

	return rule1, expectedResults
}

func TestReplaceRule(t *testing.T) {
	is := is.New(t)

	m := NewMockEvaluator()
	e := indigo.NewEngine(m)

	rule, expectedResults := makeRulesAndExpectedResult()

	e.AddRule(rule)

	result, err := e.Evaluate(nil, "rule1")
	is.NoErr(err)
	is.True(match(result, expectedResults))

	// Now replace the e1 rule
	err = e.ReplaceRule("rule1/E/e1", &indigo.Rule{
		ID:   "e1",
		Expr: `false`,
	})
	is.NoErr(err)

	// Change expected outcome
	expectedResults["e1"] = false

	// Re-evaluate and check that we got the new e1 result
	r2, err := e.Evaluate(nil, "rule1")
	is.NoErr(err)
	is.True(match(r2, expectedResults))

	err = e.ReplaceRule("rule1/D", &indigo.Rule{
		ID:   "D",
		Expr: `false`})
	is.NoErr(err)
	expectedResults["D"] = false
	delete(expectedResults, "d1")
	delete(expectedResults, "d2")
	delete(expectedResults, "d3")

	r2, err = e.Evaluate(nil, "rule1")
	is.NoErr(err)
	// fmt.Println(indigo.SummarizeResults(r2))
	is.True(match(r2, expectedResults))

	// Test with wrong address
	err = e.ReplaceRule("D", &indigo.Rule{
		ID:   "D",
		Expr: `false`})
	is.True(err != nil)

}

// Test that the "stop negative parent" option is respected, and that the rules are evaluated in correct order
func TestEvaluationTraversalStopNegativeParent(t *testing.T) {
	is := is.New(t)

	m := NewMockEvaluator()
	e := indigo.NewEngine(m)

	err := e.AddRule(makeRuleWithOptions())
	is.NoErr(err)

	expectedResults := map[string]bool{
		"rule1": true,
		"D":     true,
		"d1":    true,
		"d2":    false,
		"d3":    true,
		"B":     false,
		// These rules are subjct to these options:
		//   ReturnPass FALSE (set on B)
		//   ReturnFail FALSE default
		// "b1": true,
		// "b2": false,
		// "b3": true,
		// "b4": false,
		// Since B4 isn't returned, neither are its children
		// "b4-1": true,
		// "b4-2": false
		"E": false,
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
		// "e1", excluded because E==false with StopIfParentNegative option
		// "e2",
		// "e3",
	}

	result, err := e.Evaluate(nil, "rule1")
	// fmt.Printf("expected     :%v\n", expectedOrder)
	// fmt.Printf("m.rulesTested:%v\n", m.rulesTested)

	is.NoErr(err)
	is.Equal(result.RulesEvaluated, len(m.rulesTested))
	is.True(reflect.DeepEqual(expectedOrder, m.rulesTested)) // not all rules were evaluated

	is.True(match(result, expectedResults))
}

func match(result *indigo.Result, expected map[string]bool) bool {

	if expected[result.RuleID] != result.Pass {
		// fmt.Println(result.RuleID)
		return false
	}

	for _, r := range result.Results {
		if match(r, expected) != true {
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
			opts: []indigo.EvalOption{indigo.MaxDepth(0)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 0)
			},
		},
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
			opts: []indigo.EvalOption{indigo.ReturnFail(false), indigo.ReturnPass(false)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 0)
			},
		},
		{
			opts: []indigo.EvalOption{indigo.ReturnPass(false)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 2) // should get B and E
				is.True(!r.Results["B"].Pass)
				is.True(!r.Results["E"].Pass)
			},
		},
		{
			opts: []indigo.EvalOption{indigo.ReturnFail(false)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 1)
				is.True(r.Results["D"].Pass)
			},
		},
	}

	m := NewMockEvaluator()
	e := indigo.NewEngine(m)
	err := e.AddRule(makeRuleNoOptions())
	is.NoErr(err)

	for _, c := range cases {
		result, err := e.Evaluate(nil, "rule1", c.opts...)
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
				is.Equal(len(r.Results), 3)                            // B, D and E
				is.Equal(len(r.Results["D"].Results), 2)               // Only want true rules
				is.Equal(len(r.Results["B"].Results), 2)               // Do not want true rules
				is.Equal(len(r.Results["B"].Results["b4"].Results), 1) // Ensure B's opts are inherited
				is.Equal(len(r.Results["E"].Results), 0)               // E is negative, skip child rule

			},
		},
		{
			opts: []indigo.EvalOption{indigo.ReturnPass(true)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 3)                            // B, D and E
				is.Equal(len(r.Results["D"].Results), 2)               // Only want true rules
				is.Equal(len(r.Results["B"].Results), 2)               // Do not want true rules
				is.Equal(len(r.Results["B"].Results["b4"].Results), 1) // Ensure B's opts are inherited
				is.Equal(len(r.Results["E"].Results), 0)               // E is negative, skip child rule

			},
		},
		{
			opts: []indigo.EvalOption{indigo.ReturnPass(false)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 2)                            // B and E only
				is.Equal(len(r.Results["B"].Results), 2)               // Do not want true rules
				is.Equal(len(r.Results["B"].Results["b4"].Results), 1) // Ensure B's opts are inherited
				is.Equal(len(r.Results["E"].Results), 0)               // E is negative, skip child rule

			},
		},
		{
			opts: []indigo.EvalOption{indigo.ReturnFail(true)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 3)                            // B, D and E
				is.Equal(len(r.Results["D"].Results), 2)               // Only want true rules
				is.Equal(len(r.Results["B"].Results), 2)               // Do not want true rules
				is.Equal(len(r.Results["B"].Results["b4"].Results), 1) // Ensure B's opts are inherited
				is.Equal(len(r.Results["E"].Results), 0)               // E is negative, skip child rule

			},
		},
		{
			opts: []indigo.EvalOption{indigo.ReturnFail(false)},
			chk: func(r *indigo.Result) {
				is.Equal(len(r.Results), 1)              // D only
				is.Equal(len(r.Results["D"].Results), 2) // Only want true rules
			},
		},
	}

	m := NewMockEvaluator()
	e := indigo.NewEngine(m)
	err := e.AddRule(makeRuleWithOptions())
	is.NoErr(err)

	for _, c := range cases {
		r, err := e.Evaluate(nil, "rule1", c.opts...)
		is.NoErr(err)
		c.chk(r)
	}
}

func TestDiagnosticOptions(t *testing.T) {

	is := is.New(t)

	// Turn off diagnostic collection, but request it at eval time
	m := NewMockEvaluator()
	engine := indigo.NewEngine(m, indigo.CollectDiagnostics(false))
	err := engine.AddRule(makeRuleNoOptions())
	is.NoErr(err)

	_, err = engine.Evaluate(nil, "rule1", indigo.ReturnDiagnostics(true))
	if err == nil {
		t.Errorf("Wanted error; should require indigo.CollectDiagnostics to be turned on to enable indigo.ReturnDiagnostics")
	}

	// Do not specify diagnostic collection (should be off)
	engine = indigo.NewEngine(m)
	err = engine.AddRule(makeRuleNoOptions())
	is.NoErr(err)

	r, err := engine.Evaluate(nil, "rule1")
	is.NoErr(err)
	is.Equal(r.RulesEvaluated, 10)

	for _, c := range r.Results {
		is.Equal(c.Diagnostics, "")
	}

	// Turn off diagnostic collection
	engine = indigo.NewEngine(m)
	err = engine.AddRule(makeRuleNoOptions())
	is.NoErr(err)

	r, err = engine.Evaluate(nil, "rule1", indigo.ReturnDiagnostics(false))
	is.NoErr(err)
	is.Equal(r.RulesEvaluated, 10)

	for _, c := range r.Results {
		is.Equal(c.Diagnostics, "")
	}

	// Turn on diagnostic collection
	engine = indigo.NewEngine(m, indigo.CollectDiagnostics(true))
	err = engine.AddRule(makeRuleNoOptions())
	is.NoErr(err)

	r, err = engine.Evaluate(nil, "rule1", indigo.ReturnDiagnostics(true))
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

func TestConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	is := is.New(t)
	rand.Seed(time.Now().Unix())

	m := indigo.NoOpEvaluator{}
	e := indigo.NewEngine(m)

	var wg sync.WaitGroup

	for i := 1; i < 50_000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err := e.AddRule(makeRuleWithID(fmt.Sprintf("rule%d", i)))
			is.NoErr(err)
			r, err := e.Evaluate(nil, fmt.Sprintf("rule%d", i), indigo.ReturnDiagnostics(false))
			is.NoErr(err)
			is.Equal(r.RulesEvaluated, 10)
		}(i)
		time.Sleep(time.Duration(rand.Intn(3) * int(time.Millisecond)))
	}

	wg.Wait()
}
