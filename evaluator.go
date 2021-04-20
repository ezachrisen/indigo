package indigo

import "github.com/ezachrisen/indigo/schema"

// Evaluator is the interface implemented by types that can evaluate expressions defined in
// the rules.
type Evaluator interface {
	// Evaluate tests the rule expression against the data.
	// Returns the result of the evaluation and a string containing diagnostic information.
	// Diagnostic information is only returned if explicitly requested.
	Evaluate(data map[string]interface{}, expr string, s schema.Schema, self interface{}, evalData interface{}, returnDiagnostics bool) (schema.Value, string, error)

	// Compile pre-processes the expression, returning a compiled version.
	// The Indigo engine will store the compiled version, later providing it back to the
	// evaluator.
	//
	// collectDiagnostics instructs the compiler to generate additional information
	// to help provide diagnostic information on the evaluation later.
	// dryRun performs the compilation, but doesn't store the results, mainly
	// for the purpose of checking rule correctness.
	Compile(expr string, s schema.Schema, resultType schema.Type, collectDiagnostics, dryRun bool) (interface{}, error)
}
