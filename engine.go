package indigo

import (
	"fmt"
	"strings"
)

// Engine is the Indigo rules engine.
// Use the engine to first compile the rules, then evaluate them.
type Engine struct {

	// The Evaluator that will be used to evaluate rules in this engine
	evaluator Evaluator

	// Options used by the engine during compilation and evaluation
	opts EngineOptions
}

// NewEngine initialize a rules engine with evaluator and the options provided
func NewEngine(evaluator Evaluator, opts ...EngineOption) *Engine {
	engine := Engine{
		evaluator: evaluator,
	}
	applyEngineOptions(&engine.opts, opts...)
	return &engine
}

// Compile prepares the rule, and all its children, to be evaluated.
// Engine delegates most of the work to the Evaluator, whose
// Compile method will be called for each rule.
//
// Depending on the Evaluator used, this step will provide rule
// expression error checking. The result of error checking will be returned
// as an error.
//
// Compile modifies the rule.Program field.
//
// Once submitted to Compile, you must not make any changes to the rule.
// If you make any changes, you must re-compile before evaluating.
//
// If an error occurs during compilation, rules that had already been
// compiled successfully will have had their rule.Program fields updated.
// Compile does not restore the state of the rules to its pre-Compile
// state in case of errors.
//
func (e *Engine) Compile(r *Rule) error {

	if len(strings.Trim(r.ID, " ")) == 0 {
		return fmt.Errorf("missing rule ID (expr '%s')", r.Expr)
	}

	// // We make sure that the child keys are sorted per the rule's options
	// // The evaluator's compiler may rely on the rule's sort order.
	var o EvalOptions
	applyEvalOptions(&o, r.EvalOpts...)
	r.sortChildKeys(o)

	err := e.evaluator.Compile(r, e.opts.CollectDiagnostics, e.opts.DryRun)
	if err != nil {
		return err
	}

	for _, k := range r.sortedKeys {
		child := r.Rules[k]
		err := e.Compile(child)
		if err != nil {
			return err
		}
	}

	return nil
}

// Evaluate the rule against the input data.
// All rules and child rules  will be evaluated.
// Set EvalOptions to control which rules are evaluated, and what results are returned.
// A nil data map is allowed, unless a rule uses the Self data map feature.
//
// Evaluation options passed in will be used during evaluation, but an individual rule may
// override the settings passed.
func (e *Engine) Evaluate(data map[string]interface{}, r *Rule, opts ...EvalOption) (*Result, error) {

	// Capture the options passed to Evaluate
	// The options will be used for evaluation, unless
	// overriden by individual rules
	var o EvalOptions
	applyEvalOptions(&o, opts...)

	// You can specify at the rule engine that you want to force collect diagnostics
	// even if a rule does not have it turned on
	if e.opts.ForceDiagnosticsAllRules {
		o.ReturnDiagnostics = true
	}

	if o.ReturnDiagnostics && !e.opts.CollectDiagnostics {
		return nil, fmt.Errorf("option set to return diagnostic, but engine does not have CollectDiagnostics option set")
	}

	return e.eval(data, r, o)
}

// Recursively evaluate the rule and its children
func (e *Engine) eval(data map[string]interface{}, r *Rule, o EvalOptions) (*Result, error) {

	if r == nil {
		return nil, fmt.Errorf("eval: rule is nil")
	}

	//	Apply options for this rule evaluation
	applyEvalOptions(&o, r.EvalOpts...)

	// If this rule has a reference to a 'self' object, insert it into the data.
	// If it doesn't, we must remove any existing reference to self, so that
	// child rules do not accidentally "inherit" the self object.
	if r.Self != nil {
		if data == nil {
			return nil, fmt.Errorf("rule references 'self', but data map is nil")
		}
		data[selfKey] = r.Self
	} else {
		delete(data, selfKey)
	}

	value, diagnostics, err := e.evaluator.Eval(data, r, o)
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

	if o.StopIfParentNegative && pr.Pass == false {
		return &pr, nil
	}

	for _, k := range r.sortedKeys {
		c, ok := r.Rules[k]
		if !ok {
			return nil, fmt.Errorf("eval: rule with id '%s' not found in parent '%s'", k, r.ID)
		}

		result, err := e.eval(data, c, o)
		if err != nil {
			return nil, err
		}
		if result != nil {
			if (!result.Pass && !o.DiscardFail) ||
				(result.Pass && !o.DiscardPass) {
				pr.Results[k] = result
			}
			pr.RulesEvaluated += result.RulesEvaluated
		}

		if o.StopFirstPositiveChild && result.Pass == true {
			return &pr, nil
		}

		if o.StopFirstNegativeChild && result.Pass == false {
			return &pr, nil
		}
	}
	return &pr, nil
}

// EngineOptions holds the settings used by the engine during compilation and evaluation.
// See the functional definitions below for the meaning.
type EngineOptions struct {
	CollectDiagnostics       bool
	ForceDiagnosticsAllRules bool
	DryRun                   bool
}

// EngineOption is a functional option type
type EngineOption func(f *EngineOptions)

// Given an array of EngineOption functions, apply their effect
// on the EngineOptions struct.
func applyEngineOptions(o *EngineOptions, opts ...EngineOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// Collect diagnostic information from the engine.
// Default: off
func CollectDiagnostics(b bool) EngineOption {
	return func(f *EngineOptions) {
		f.CollectDiagnostics = b
	}
}

// Force the return of diagnostic information for all rules, regardless of
// the setting on each rule. If you don't set this option, you can enable
// the return of diagnostic information for individual rules by setting an option
// on the rule itself.
// Default: off
func ForceDiagnosticsAllRules(b bool) EngineOption {
	return func(f *EngineOptions) {
		f.ForceDiagnosticsAllRules = b
	}
}

// Run through all iterations and logic, but do not
// - compile
// - evaluate
// Default: off
func DryRun(b bool) EngineOption {
	return func(f *EngineOptions) {
		f.DryRun = b
	}
}
