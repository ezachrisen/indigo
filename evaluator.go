package indigo

// Evaluator is the interface implemented by types that can evaluate expressions defined in
// the rules. The interface includes a Compile step that the Engine calls before evaluation.
// This gives the Evaluator the option of pre-processing the rule for efficiency. At a minimum
// the Compile step should give the user feedback on whether the provided rule conforms to its
// specificiations.
type Evaluator interface {
	// Compile a rule, checking for correctness and preparing the rule to be
	// evaluated later.
	Compile(rule *Rule, collectDiagnostics bool, dryRun bool) error

	// Eval tests the rule expression against the data.
	// The result is one of Indigo's types.
	Eval(data map[string]interface{}, rule *Rule, returnDiagnostics bool) (Value, string, error)
}
