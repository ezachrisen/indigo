package indigo_test

import (
	"fmt"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/schema"
)

// -------------------------------------------------- MOCK EVALUATOR
// mockEvaluator is used for testing
// It provides minimal evaluation of rules and captures
// information about which rules were processed, etc.
type mockEvaluator struct {
	rules       []string // a list of rule IDs in the evaluator
	rulesTested []string // a list of rule IDs that were evaluated
	//	ruleOptions map[string]indigo.evalOption // a copy of the evaluation options used for each rule
}

func newMockEvaluator() *mockEvaluator {
	return &mockEvaluator{
		//		ruleOptions: make(map[string]indigo.EvalOption, 10),
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
func (m *mockEvaluator) Eval(data map[string]interface{}, rule *indigo.Rule, returnDiagnostics bool) (indigo.Value, string, error) {
	m.rulesTested = append(m.rulesTested, rule.ID)

	diagnostics := ""

	if returnDiagnostics {
		diagnostics = "diagnostics here"
	}

	if rule.Expr == `true` {
		return indigo.Value{
			Val: true,
			Typ: schema.Bool{},
		}, diagnostics, nil
	}

	if rule.Expr == `self` && rule.Self != nil {
		return indigo.Value{
			Val: rule.Self.(int),
			Typ: schema.Int{},
		}, diagnostics, nil
	}

	return indigo.Value{
		Val: false,
		Typ: schema.Bool{},
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

// --------------------------------------------------
// Functions to manipulate and compare rule evaluation results
// and expected results

// flattenResults takes a hierarchy of Result objects and flattens it
// to a map of rule ID to pass/fail. This is so that it's easy to
// compare the results to expected.
func flattenResults(result *indigo.Result) map[string]bool {
	m := map[string]bool{}
	m[result.RuleID] = result.Pass
	for k := range result.Results {
		r := result.Results[k]
		mc := flattenResults(r)
		for k := range mc {
			m[k] = mc[k]
		}
	}
	return m
}

// Match compares the results to expected results. Returns false
// if any difference is found and prints the reason why.
// Call flattenResults on the *indigo.Result first.
func match(result map[string]bool, expected map[string]bool) bool {

	// fmt.Printf("Expected: %+v\n\n", expected)
	// fmt.Printf("Got: %+v\n\n", result)

	for k, v := range result {
		ev, ok := expected[k]
		if !ok {
			fmt.Println("received result for rule ", k, "(", v, "); no result was expected")
			return false
		}

		if v != ev {
			fmt.Println("result mismatch: rule ", k, ", got ", v, ", expected ", ev)
			return false
		}
	}

	for k := range expected {
		if _, ok := result[k]; !ok {
			fmt.Println("expected result for rule ", k, "; no result found")
			return false
		}
	}

	return true
}
