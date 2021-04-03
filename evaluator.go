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

	// Eval tests the rule expression  against the data.
	// The result is one of Indigo's types.
	// resultType: the type of result value the rule expects
	Eval(data map[string]interface{}, rule *Rule, returnDiagnostics bool) (Value, string, error)
}

type evalOptions struct {
	ReturnDiagnostics bool
}

// EvalOptions specify how the Engine and the Evaluator process rules.
// Pass these to Evaluate to change the behavior of the engine, or
// set options on a rule.
// If specified at the rule level, the option applies to the rule and its child rules.
type evalOption func(f *evalOptions)

// Include diagnostic information with the results.
// To enable this option, you must first turn on diagnostic
// collection at the engine level with the CollectDiagnostics EngineOption.
// Default: off
func ReturnDiagnostics(b bool) evalOption {
	return func(f *evalOptions) {
		f.ReturnDiagnostics = b
	}
}
func applyEvalOptions(o *evalOptions, opts ...evalOption) {
	for _, opt := range opts {
		opt(o)
	}
}
