package indigo_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/ezachrisen/indigo"
)

// -------------------------------------------------- MOCK EVALUATOR
// mockEvaluator is used for testing
// It provides minimal evaluation of rules and captures
// information about which rules were processed, etc.
type mockEvaluator struct {
	rules []string // a list of rule IDs in the evaluator
	// if set, diagnostic information is only returned if the flag was
	// set during compilation
	diagnosticCompileRequired bool
	// Introduce an artificial delay in evaluating the expression.
	// Used for testing the engine's context cancelation functionality.
	evalDelay time.Duration
}

type program struct {
	compiledDiagnostics bool
}

func newMockEvaluator() *mockEvaluator {
	return &mockEvaluator{}
}

func (m *mockEvaluator) Compile(_ string, _ indigo.Schema, _ indigo.Type, collectDiagnostics, _ bool) (any, error) {

	p := program{}
	if collectDiagnostics {
		p.compiledDiagnostics = true
	}

	return p, nil
}

// The mockEvaluator only knows how to evaluate 1 string: `true`. If the expression is this, the evaluation is true, otherwise false.
func (m *mockEvaluator) Evaluate(data map[string]any, expr string, s indigo.Schema, self any, prog any, resultType indigo.Type, returnDiagnostics bool) (any, *indigo.Diagnostics, error) {
	//	m.rulesTested = append(m.rulesTested, r.ID)
	time.Sleep(m.evalDelay)
	prg := program{}

	p, ok := prog.(program)
	if m.diagnosticCompileRequired {
		if !ok {
			return false, nil, errors.New("compiled data type assertion failed")
		}
		prg = p
	}

	var diagnostics *indigo.Diagnostics

	if returnDiagnostics && ((m.diagnosticCompileRequired && prg.compiledDiagnostics) || !m.diagnosticCompileRequired) {
		diagnostics = &indigo.Diagnostics{}
	}

	if expr == `true` {
		// return indigo.Value{
		// 	Val:  true,
		// 	Type: indigo.Bool{},
		// }, diagnostics, nil
		return true, diagnostics, nil
	}

	// return indigo.Value{
	// 	Val:  false,
	// 	Type: indigo.Bool{},
	// }, diagnostics, nil
	return false, diagnostics, nil
}

func (m *mockEvaluator) Reset() {
	m.rules = []string{}
}

func (m *mockEvaluator) PrintInternalStructure() {
	for _, v := range m.rules {
		fmt.Println("Rule id", v)
	}
}
