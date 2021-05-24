package indigo

import (
	"context"
	"fmt"
)

// The Engine interface defines the behavior of the rules engine.
type Engine interface {

	// Compile pre-processes the rule, returning a compiled version of the rule.
	// The rule will store the compiled version, later providing it back to the
	// evaluator.
	Compile(r *Rule, opts ...CompilationOption) error

	// Evaluate tests the rule and its child rules against the data.
	// Returns the result of the evaluation.
	Eval(ctx context.Context, r *Rule, d map[string]interface{}, opts ...EvalOption) (*Result, error)
}

// DefaultEngine provides an implementation of the Indigo Engine interface
// to evaluate rules locally.
type DefaultEngine struct {
	e Evaluator
}

// NewEngine initializes and returns a DefaultEngine.
func NewEngine(e Evaluator) *DefaultEngine {
	return &DefaultEngine{
		e: e,
	}
}

// Eval evaluates the expression of the rule and its children. It uses the evaluation
// options of each rule to determine what to do with the results, and whether to proceed
// evaluating. Options passed to this function will override the options set on the rules.
// Eval uses the Evaluator provided to the engine to perform the expression evaluation.
func (e *DefaultEngine) Eval(ctx context.Context, r *Rule,
	d map[string]interface{}, opts ...EvalOption) (*Result, error) {
	if r == nil {
		return nil, fmt.Errorf("rule is nil")
	}

	if e == nil {
		return nil, fmt.Errorf("engine is nil")
	}

	if e.e == nil {
		return nil, fmt.Errorf("evaluator is nil")
	}

	if d == nil {
		return nil, fmt.Errorf("data is nil")
	}

	o := r.EvalOptions
	applyEvaluatorOptions(&o, opts...)

	// If this rule has a reference to a 'self' object, insert it into the d.
	// If it doesn't, we must remove any existing reference to self, so that
	// child rules do not accidentally "inherit" the self object.
	if r.Self != nil {
		d[selfKey] = r.Self
	} else {
		delete(d, selfKey)
	}

	pr := &Result{
		Rule:    r,
		Pass:    true,
		Results: make(map[string]*Result, len(r.Rules)), // TODO: consider how large to make it
	}

	// Default the result type to boolean
	resultType := r.ResultType
	if resultType == nil {
		resultType = Bool{}
	}

	val, diagnostics, err := e.e.Evaluate(d, r.Expr, r.Schema, r.Self, r.Program, resultType, o.ReturnDiagnostics)
	if err != nil {
		return nil, fmt.Errorf("rule %s: %w", r.ID, err)

	}

	pr.Value = val.Val
	pr.Diagnostics = diagnostics
	// if pr.Diagnostics != nil {
	// 	pr.Diagnostics.Rule = r
	// 	pr.Diagnostics.Data = d
	// }
	pr.EvalOptions = o

	if pass, ok := val.Val.(bool); ok {
		pr.Pass = pass
	}

	if o.StopIfParentNegative && !pr.Pass {
		return pr, nil
	}

	for _, cr := range r.sortChildKeys(o) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if o.ReturnDiagnostics {
				pr.RulesEvaluated = append(pr.RulesEvaluated, cr)
			}
			result, err := e.Eval(ctx, cr, d, opts...)
			if err != nil {
				return nil, err
			}

			if (!result.Pass && !o.DiscardFail) ||
				(result.Pass && !o.DiscardPass) {
				pr.Results[cr.ID] = result
			}

			if o.StopFirstPositiveChild && result.Pass {
				return pr, nil
			}

			if o.StopFirstNegativeChild && !result.Pass {
				return pr, nil
			}
		}
	}
	return pr, nil
}

// Compile uses the Evaluator's compile method to check the rule and its children,
// returning any validation errors. Stores a compiled version of the rule in the
// rule.Program field (if the compiler returns a program).
func (e *DefaultEngine) Compile(r *Rule, opts ...CompilationOption) error {

	if r == nil {
		return fmt.Errorf("rule is nil")
	}

	if e == nil {
		return fmt.Errorf("engine is nil")
	}

	if e.e == nil {
		return fmt.Errorf("evaluator is nil")
	}

	o := compileOptions{}
	applyCompilerOptions(&o, opts...)

	resultType := r.ResultType
	if resultType == nil {
		resultType = Bool{}
	}

	prg, err := e.e.Compile(r.Expr, r.Schema, resultType, o.collectDiagnostics, o.dryRun)
	if err != nil {
		return fmt.Errorf("rule %s: %w", r.ID, err)
	}

	if !o.dryRun {
		r.Program = prg
	}

	for _, cr := range r.Rules {
		err := e.Compile(cr, opts...)
		if err != nil {
			return err
		}
	}
	return nil
}

type compileOptions struct {
	dryRun             bool
	collectDiagnostics bool
}

// CompilationOption is a functional option to specify compilation behavior.
type CompilationOption func(f *compileOptions)

// DryRun specifies to perform all compilation steps, but do not save the results.
// This is to allow a client to check all rules in a rule tree before
// committing the actual compilation results to the rule.
func DryRun(b bool) CompilationOption {
	return func(f *compileOptions) {
		f.dryRun = b
	}
}

// CollectDiagnostics instructs the engine and its evaluator to save any
// intermediate results of compilation in order to provide good diagnostic
// information after evaluation. Not all evaluators need to have this option set.
func CollectDiagnostics(b bool) CompilationOption {
	return func(f *compileOptions) {
		f.collectDiagnostics = b
	}
}

// Given an array of EngineOption functions, apply their effect
// on the engineOptions struct.
func applyCompilerOptions(o *compileOptions, opts ...CompilationOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// EvalOptions determines how the engine should treat the results of evaluating a rule.
type EvalOptions struct {

	// StopIfParentNegative  not evaluate child rules if the parent's expression is false.
	// Use case: apply a "global" rule to all the child rules.
	StopIfParentNegative bool `json:"stop_if_parent_negative"`

	// Stops the evaluation of child rules when the first positive child is encountered.
	// Results will be partial. Only the child rules that were evaluated will be in the results.
	// By default rules are evaluated in alphabetical order.
	// Use case: role-based access; allow action if any child rule (permission rule) allows it.
	StopFirstPositiveChild bool `json:"stop_first_positive_child"`

	// Stops the evaluation of child rules when the first negative child is encountered.
	// Results will be partial. Only the child rules that were evaluated will be in the results.
	// By default rules are evaluated in alphabetical order.
	// Use case: you require ALL child rules to be satisfied.
	StopFirstNegativeChild bool `json:"stop_first_negative_child"`

	// Do not return rules that passed
	// Default: all rules are returned
	DiscardPass bool `json:"discard_pass"`

	// Do not return rules that failed
	// Default: all rules are returned
	DiscardFail bool `json:"discard_fail"`

	// Include diagnostic information with the results.
	// To enable this option, you must first turn on diagnostic
	// collection at the engine level with the CollectDiagnostics EngineOption.
	ReturnDiagnostics bool `json:"return_diagnostics"`

	// Specify the function used to sort the child rules before evaluation.
	// Useful in scenarios where you are asking the engine to stop evaluating
	// after either the first negative or first positive child.
	// See the provided SortAlpha function as an example.
	// Default: No sort
	SortFunc func(rules []*Rule, i, j int) bool `json:"-"`
}

// EvalOption is a functional option for specifying how evaluations behave.
type EvalOption func(f *EvalOptions)

// ReturnDiagnostics specifies that diagnostics should be returned
// from this evaluation. You must first turn on diagnostic collectionat the
// engine level when compiling the rule.
func ReturnDiagnostics(b bool) EvalOption {
	return func(f *EvalOptions) {
		f.ReturnDiagnostics = b
	}
}

// SortFunc specifies the function used to sort child rules before evaluation.
func SortFunc(x func(rules []*Rule, i, j int) bool) EvalOption {
	return func(f *EvalOptions) {
		f.SortFunc = x
	}
}

// DiscardFail specifies whether to omit failed rules from the results.
func DiscardFail(b bool) EvalOption {
	return func(f *EvalOptions) {
		f.DiscardFail = b
	}
}

// DiscardPass specifies whether to omit passed rules from the results.
func DiscardPass(b bool) EvalOption {
	return func(f *EvalOptions) {
		f.DiscardPass = b
	}
}

// StopIfParentNegative prevents the evaluation of child rules if the
// parent rule itself is negative.
func StopIfParentNegative(b bool) EvalOption {
	return func(f *EvalOptions) {
		f.StopIfParentNegative = b
	}
}

// StopFirstNegativeChild stops the evaluation of child rules once the first
// negative child has been found.
func StopFirstNegativeChild(b bool) EvalOption {
	return func(f *EvalOptions) {
		f.StopFirstNegativeChild = b
	}
}

// StopFirstPositiveChild stops the evaluation of child rules once the first
// positive child has been found.
func StopFirstPositiveChild(b bool) EvalOption {
	return func(f *EvalOptions) {
		f.StopFirstPositiveChild = b
	}
}

// See the EvalOptions struct for documentation.
func applyEvaluatorOptions(o *EvalOptions, opts ...EvalOption) {
	for _, opt := range opts {
		opt(o)
	}
}
