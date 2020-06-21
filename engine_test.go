package indigo_test

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

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
func (m *MockEvaluator) Compile(ruleID string, expr string, resultType indigo.Type, s indigo.Schema, collectDiagnostics bool) error {
	m.rules = append(m.rules, ruleID)
	return nil
}

// The MockEvaluator only knows how to evaluate 1 string: `true`. If the expression is this, the evaluation is true, otherwise false.
func (m *MockEvaluator) Eval(data map[string]interface{}, ruleID string, expr string, resultType indigo.Type, opt indigo.EvalOptions) (indigo.Value, string, error) {
	m.rulesTested = append(m.rulesTested, ruleID)
	if expr == `true` {
		return indigo.Value{
			Val: true,
			Typ: indigo.Bool{},
		}, "", nil
	}

	return indigo.Value{
		Val: false,
		Typ: indigo.Bool{},
	}, "", nil
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

func TestEvaluationTraversalDefault(t *testing.T) {
	is := is.New(t)

	m := NewMockEvaluator()
	e := indigo.NewEngine(m)

	e.AddRule(makeRuleNoOptions())

	ruleIDs := []string{"rule1", "rule1/D", "rule1/D/d1", "rule1/D/d2", "rule1/D/d3", "rule1/B", "rule1/E", "rule1/E/e1", "rule1/E/e2", "rule1/E/e3"}

	expected := map[string]bool{
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

	result, err := e.Evaluate(nil, "rule1")

	sort.Strings(ruleIDs)
	sort.Strings(m.rulesTested)
	// fmt.Printf("A:%v\nB:%v\n", ruleIDs, m.rulesTested)
	// fmt.Printf("C:%v\n", e.Rules()["rule1"].Rules)
	is.NoErr(err)
	is.Equal(result.RulesEvaluated, len(m.rulesTested))
	is.True(reflect.DeepEqual(ruleIDs, m.rulesTested)) // not all rules were evaluated
	is.True(match(result, expected))
}

func TestEvaluationTraversalStopNegativeParent(t *testing.T) {
	is := is.New(t)

	m := NewMockEvaluator()
	e := indigo.NewEngine(m)

	err := e.AddRule(makeRuleWithOptions())
	is.NoErr(err)

	ruleIDs := []string{"rule1", "rule1/D", "rule1/D/d1", "rule1/D/d2", "rule1/D/d3", "rule1/B", "rule1/E"}

	expected := map[string]bool{
		"rule1": true,
		"D":     true,
		"d1":    true,
		"d2":    false,
		"d3":    true,
		"B":     false,
		"E":     false,
	}

	result, err := e.Evaluate(nil, "rule1")

	sort.Strings(ruleIDs)
	sort.Strings(m.rulesTested)
	//fmt.Printf("%v\n%v\n", ruleIDs, m.rulesTested)

	is.NoErr(err)
	is.Equal(result.RulesEvaluated, len(m.rulesTested))
	is.True(reflect.DeepEqual(ruleIDs, m.rulesTested)) // not all rules were evaluated
	is.True(match(result, expected))
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

func TestEvalOptions(t *testing.T) {
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
