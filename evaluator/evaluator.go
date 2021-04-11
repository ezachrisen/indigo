// Package evaluator provides interfaces for compilation and evaluation.
//
// These interfaces are implemented by rule evaluators, such as CEL.
package evaluator

import "github.com/ezachrisen/indigo/schema"

// Evaluator is the interface implemented by types that can evaluate expressions defined in
// the rules.
type Evaluator interface {
	// Evaluate tests the rule against the data.
	// Returns the result of the evaluation and a string containing diagnostic information.
	// Diagnostic information is only returned if explicitly requested.
	Evaluate(data map[string]interface{}, expr string, s schema.Schema, self interface{}, evalData interface{}, returnDiagnostics bool) (schema.Value, string, error)
}
