package indigo

// Evaluator is the interface implemented by types that can evaluate expressions defined in
// the rules.
type Evaluator interface {
	// Evaluate tests the rule expression against the data.
	// Returns the result of the evaluation and a string containing diagnostic information.
	// Diagnostic information is only returned if explicitly requested.
	// Evaluate should check the result against the expected resultType and return an error if the
	// result does not match.
	Evaluate(data map[string]interface{}, expr string, s Schema,
		self interface{}, evalData interface{}, resultType Type, returnDiagnostics bool) (Value, *Diagnostics, error)

	// Compile pre-processes the expression, returning a compiled version.
	// The Indigo engine will store the compiled version, later providing it back to the
	// evaluator.
	//
	// collectDiagnostics instructs the compiler to generate additional information
	// to help provide diagnostic information on the evaluation later.
	// dryRun performs the compilation, but doesn't store the results, mainly
	// for the purpose of checking rule correctness.
	Compile(expr string, s Schema, resultType Type, collectDiagnostics, dryRun bool) (interface{}, error)
}
