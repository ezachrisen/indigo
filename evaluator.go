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
	Eval(data map[string]interface{}, rule *Rule, opt EvalOptions) (Value, string, error)
}

// EvalOptions specify how the Engine and the Evaluator process rules.
// Pass these to Evaluate to change the behavior of the engine, or
// set options on a rule.
// If specified at the rule level, the option applies to the rule and its child rules.
type EvalOption func(f *EvalOptions)

// Determines how far to recurse through child rules.
// The default is rules.defaultDepth.
func MaxDepth(d int) EvalOption {
	return func(f *EvalOptions) {
		f.MaxDepth = d
	}
}

// Does not evaluate child rules if the parent's expression is false.
// Use case: apply a "global" rule to all the child rules.
func StopIfParentNegative(b bool) EvalOption {
	return func(f *EvalOptions) {
		f.StopIfParentNegative = b
	}
}

// Stops the evaluation of child rules when the first positive child is encountered.
// Results will be partial. Only the child rules that were evaluated will be in the results.
// By default rules are evaluated in alphabetical order.
// Use case: role-based access; allow action if any child rule (permission rule) allows it.
func StopFirstPositiveChild(b bool) EvalOption {
	return func(f *EvalOptions) {
		f.StopFirstPositiveChild = b
	}
}

// Stops the evaluation of child rules when the first negative child is encountered.
// Results will be partial. Only the child rules that were evaluated will be in the results.
// By default rules are evaluated in alphabetical order.
// Use case: you require ALL child rules to be satisifed.
func StopFirstNegativeChild(b bool) EvalOption {
	return func(f *EvalOptions) {
		f.StopFirstNegativeChild = b
	}
}

// Return rules that passed.
// By default this is on.
func ReturnPass(b bool) EvalOption {
	return func(f *EvalOptions) {
		f.ReturnPass = b
	}
}

// Return rules that did not pass.
// By default this is on.
func ReturnFail(b bool) EvalOption {
	return func(f *EvalOptions) {
		f.ReturnFail = b
	}
}

// Include diagnostic information with the results.
// To enable this option, you must first turn on diagnostic
// collection at the engine level with the CollectDiagnostics EngineOption.
// Default: off
func ReturnDiagnostics(b bool) EvalOption {
	return func(f *EvalOptions) {
		f.ReturnDiagnostics = b
	}
}

// Specify the function used to sort the child rules before evaluation.
// Useful in scenarios where you are asking the engine to stop evaluating
// after either the first negative or first positive child.
// Default: sort by the key name
func SortFunc(s func(i, j int) bool) EvalOption {
	return func(f *EvalOptions) {
		f.SortFunc = s
	}
}

func applyEvalOptions(o *EvalOptions, opts ...EvalOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// EvalOptions holds the result of applying the functional options
// to a rule. This struct is passed to the Evaluator's Eval function.
// See the corresponding functions for documentation.
type EvalOptions struct {
	MaxDepth               int
	StopIfParentNegative   bool // TODO: add StopIfParentPositive
	StopFirstPositiveChild bool
	StopFirstNegativeChild bool
	ReturnPass             bool
	ReturnFail             bool
	ReturnDiagnostics      bool
	DryRun                 bool
	SortFunc               func(i, j int) bool
}
