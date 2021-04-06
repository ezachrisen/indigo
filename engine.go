package indigo

import (
	"fmt"
)

// Engine is the Indigo rules engine.
// Use the engine to first compile the rules, then evaluate them.
type Engine struct {
	// The Evaluator that will be used to evaluate rules in this engine
	evaluator Evaluator
}

// NewEngine initialize a rules engine
func NewEngine(evaluator Evaluator) *Engine {
	engine := Engine{
		evaluator: evaluator,
	}
	return &engine
}

// Compile prepares the rule, and all its children, to be evaluated.
// Engine delegates most of the work to the Evaluator, whose
// Compile method will be called for each rule.
//
// Depending on the Evaluator used, this step will provide rule
// expression error checking.
//
// Compile modifies the rule.Program field, unless the DryRun option is passed.
//
// Once submitted to Compile, you must not make any changes to the rule.
// If you make any changes, you must re-compile before evaluating.
//
// If an error occurs during compilation, rules that had already been
// compiled successfully will have had their rule.Program fields updated.
// Compile does not restore the state of the rules to its pre-Compile
// state in case of errors. To avoid this problem, do a dry run first.
func (e *Engine) Compile(r *Rule, opts ...compileOption) error {

	o := compileOptions{}
	applyCompileOptions(&o, opts...)
	err := e.evaluator.Compile(r, o.CollectDiagnostics, o.DryRun)
	if err != nil {
		return err
	}

	for _, c := range r.Rules {
		err := e.Compile(c, opts...)
		if err != nil {
			return err
		}
	}
	return nil
}

// Evaluate the rule and its children against the input data.
func (e *Engine) Evaluate(data map[string]interface{}, r *Rule, opts ...evalOption) (*Result, error) {

	if r == nil {
		return nil, fmt.Errorf("Evaluate: rule is nil")
	}

	if data == nil {
		return nil, fmt.Errorf("Evaluate: data is nil")
	}

	//	Apply options for this rule evaluation
	o := evalOptions{}
	applyEvalOptions(&o, opts...)

	// If this rule has a reference to a 'self' object, insert it into the data.
	// If it doesn't, we must remove any existing reference to self, so that
	// child rules do not accidentally "inherit" the self object.
	if r.Self != nil {
		data[selfKey] = r.Self
	} else {
		delete(data, selfKey)
	}

	value, diagnostics, err := e.evaluator.Eval(data, r, o.ReturnDiagnostics)
	if err != nil {
		return nil, err
	}

	pr := Result{
		RuleID:         r.ID,
		Meta:           r.Meta,
		Pass:           true,
		RulesEvaluated: 1,
		Value:          value.Val,
		Diagnostics:    diagnostics,
		Results:        make(map[string]*Result, len(r.Rules)), // TODO: consider how large to make it
	}

	// TODO: check that we got the expected value type
	if pass, ok := value.Val.(bool); ok {
		pr.Pass = pass
	}

	if r.Options.StopIfParentNegative && pr.Pass == false {
		return &pr, nil
	}

	r.sortChildKeys()
	for _, c := range r.sortedRules {
		result, err := e.Evaluate(data, c, opts...)
		if err != nil {
			return nil, err
		}

		pr.RulesEvaluated += result.RulesEvaluated

		if (!result.Pass && !r.Options.DiscardFail) ||
			(result.Pass && !r.Options.DiscardPass) {
			pr.Results[c.ID] = result
		}

		if r.Options.StopFirstPositiveChild && result.Pass == true {
			return &pr, nil
		}

		if r.Options.StopFirstNegativeChild && result.Pass == false {
			return &pr, nil
		}
	}
	return &pr, nil
}

type compileOptions struct {
	DryRun             bool
	CollectDiagnostics bool
}

type compileOption func(f *compileOptions)

// Perform all compilation steps, but do not save the results.
// This is to allow a client to check all rules in a rule tree before
// committing the actual compilation results to the rule.
func DryRun(b bool) compileOption {
	return func(f *compileOptions) {
		f.DryRun = b
	}
}

// CollectDiagnostics instructs the engine and its evaluator to save any
// intermediate results of compilation in order to provide good diagnostic
// information after evaluation. Not all evaluators need to have this option set.
// Default: off
func CollectDiagnostics(b bool) compileOption {
	return func(f *compileOptions) {
		f.CollectDiagnostics = b
	}
}

// Given an array of EngineOption functions, apply their effect
// on the engineOptions struct.
func applyCompileOptions(o *compileOptions, opts ...compileOption) {
	for _, opt := range opts {
		opt(o)
	}
}

type evalOptions struct {
	ReturnDiagnostics bool
}

// EvalOptions determine how the engine behaves during the evaluation .
type evalOption func(f *evalOptions)

// Include diagnostic information with the results.
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
