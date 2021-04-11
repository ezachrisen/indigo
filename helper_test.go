package indigo_test

import (
	"fmt"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/schema"
)

// -------------------------------------------------- NO-OP EVALUATOR
type noOpEvaluator struct{}

func (n *noOpEvaluator) Evaluate(map[string]interface{}, *indigo.Rule, bool) (indigo.Value, string, error) {
	return indigo.Value{
		Val: false,
		Typ: schema.Bool{},
	}, "", nil
}

// -------------------------------------------------- MOCK EVALUATOR
// mockEvaluator is used for testing
// It provides minimal evaluation of rules and captures
// information about which rules were processed, etc.
type mockEvaluator struct {
	rules       []string // a list of rule IDs in the evaluator
	rulesTested []string // a list of rule IDs that were evaluated
	// if set, diagnostic information is only returned if the flag was
	// set during compilation
	diagnosticCompileRequired bool
}

type program struct {
	compiledDiagnostics bool
}

func newMockEvaluator() *mockEvaluator {
	return &mockEvaluator{}
}

func (m *mockEvaluator) Compile(r *indigo.Rule, collectDiagnostics, dryRun bool) (interface{}, error) {

	p := program{}
	if collectDiagnostics {
		p.compiledDiagnostics = true
	}

	return p, nil
}

// func CompileWithDiagnostics(rule *indigo.Rule) error {
// 	p := program{}
// 	p.compiledDiagnostics = true
// 	m.programs[r]=p
// 	return nil
// }

func (m *mockEvaluator) ResetRulesTested() {
	m.rulesTested = []string{}
}

// The mockEvaluator only knows how to evaluate 1 string: `true`. If the expression is this, the evaluation is true, otherwise false.
func (m *mockEvaluator) Evaluate(data map[string]interface{}, r *indigo.Rule, prog interface{}, returnDiagnostics bool) (indigo.Value, string, error) {
	m.rulesTested = append(m.rulesTested, r.ID)
	prg := program{}

	p, ok := prog.(program)
	if m.diagnosticCompileRequired {
		if !ok {
			return indigo.Value{
				Val: false,
				Typ: schema.Bool{},
			}, "", fmt.Errorf("compiled data type assertion failed")
		} else {
			prg = p
		}
	}

	var diagnostics string

	if returnDiagnostics && ((m.diagnosticCompileRequired && prg.compiledDiagnostics) || !m.diagnosticCompileRequired) {
		diagnostics = "diagnostics here"
	}

	if r.Expr == `true` {
		return indigo.Value{
			Val: true,
			Typ: schema.Bool{},
		}, diagnostics, nil
	}

	if r.Expr == `self` && r.Self != nil {
		return indigo.Value{
			Val: r.Self,
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
	m[result.Rule.ID] = result.Pass
	for k := range result.Results {
		r := result.Results[k]
		mc := flattenResults(r)
		for k := range mc {
			m[k] = mc[k]
		}
	}
	return m
}

// flattenResults takes a hierarchy of Result objects and flattens it
// to a map of rule ID to pass/fail. This is so that it's easy to
// compare the results to expected.
func flattenResultsDiagnostics(result *indigo.Result) map[string]string {
	m := map[string]string{}
	m[result.Rule.ID] = result.Diagnostics
	for k := range result.Results {
		r := result.Results[k]
		mc := flattenResultsDiagnostics(r)
		for k := range mc {
			m[k] = mc[k]
		}
	}
	return m
}

func anyNotEmpty(m map[string]string) error {
	for k, v := range m {
		if v != "" {
			return fmt.Errorf("diagnostics for rule '%s' is not empty: '%s'", k, v)
		}
	}
	return nil
}

func allNotEmpty(m map[string]string) error {
	for k, v := range m {
		if v != "diagnostics here" {
			return fmt.Errorf("diagnostics missing for rule '%s'", k)
		}
	}
	return nil
}

// Match compares the results to expected results.
// Call flattenResults on the *indigo.Result first.
func match(result map[string]bool, expected map[string]bool) error {

	// fmt.Printf("Expected: %+v\n\n", expected)
	// fmt.Printf("Got: %+v\n\n", result)

	for k, v := range result {
		ev, ok := expected[k]
		if !ok {
			return fmt.Errorf("received result for rule %s ( %v ); no result was expected", k, v)
		}

		if v != ev {
			return fmt.Errorf("result mismatch: rule %s: got %v, wanted %v", k, v, ev)
		}
	}

	for k := range expected {
		if _, ok := result[k]; !ok {
			return fmt.Errorf("expected result for rule %s: no result found", k)
		}
	}

	return nil
}

func copyMap(m map[string]bool) map[string]bool {

	n := make(map[string]bool, len(m))

	for k, v := range m {
		n[k] = v
	}
	return n
}

func deleteKeys(m map[string]bool, keys ...string) map[string]bool {

	for _, k := range keys {
		delete(m, k)
	}
	return m
}

func apply(r *indigo.Rule, f func(r *indigo.Rule) error) error {
	err := f(r)
	if err != nil {
		return err
	}
	for _, c := range r.Rules {
		err := apply(c, f)
		if err != nil {
			return err
		}
	}
	return nil
}
