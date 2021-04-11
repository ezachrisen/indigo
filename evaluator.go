package indigo

// Evaluator is the interface implemented by types that can evaluate expressions defined in
// the rules.
type Evaluator interface {
	// Evaluate tests the rule against the data.
	// Returns the result of the evaluation and a string containing diagnostic information.
	// Diagnostic information is only returned if explicitly requested.
	Evaluate(data map[string]interface{}, r *Rule, evalData interface{}, returnDiagnostics bool) (Value, string, error)
}
