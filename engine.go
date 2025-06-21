package indigo

import (
	"context"
	"fmt"
)

// Compiler is the interface that wraps the Compile method.
// Compile pre-processes the rule recursively using an ExpressionCompiler, which
// is applied to each rule.
type Compiler interface {
	Compile(r *Rule, opts ...CompilationOption) error
}

// Evaluator is the interface that wraps the Evaluate method.
// Evaluate tests the rule recursively against the input data using an ExpressionEvaluator,
// which is applied to each rule.
type Evaluator interface {
	Eval(ctx context.Context, r *Rule, d map[string]interface{}, opts ...EvalOption) (*Result, error)
}

// Engine is the interface that groups the Compiler and Evaluator interfaces.
// An Engine is used to compile and evaluate rules.
type Engine interface {
	Compiler
	Evaluator
}

// DefaultEngine provides an implementation of the Indigo Engine interface
// to evaluate rules locally.
type DefaultEngine struct {
	e ExpressionCompilerEvaluator
}

// NewEngine initializes and returns a DefaultEngine.
func NewEngine(e ExpressionCompilerEvaluator) *DefaultEngine {
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

	if err := validateEvalArguments(r, e, d); err != nil {
		return nil, err
	}

	o := r.EvalOptions
	applyEvaluatorOptions(&o, opts...)
	setSelfKey(r, d)

	//	fmt.Println("Rule ID", r.ID, "return diags?", o.ReturnDiagnostics)

	val, diagnostics, err := e.e.Evaluate(d, r.Expr, r.Schema, r.Self, r.Program, defaultResultType(r), o.ReturnDiagnostics)
	if err != nil {
		return nil, fmt.Errorf("rule %s: %w", r.ID, err)
	}

	//	fmt.Println("Rule ID", r.ID, "diagnostics: ", diagnostics)

	u := &Result{
		Rule:           r,
		ExpressionPass: true,                                   // default boolean result
		Results:        make(map[string]*Result, len(r.Rules)), // TODO: consider how large to make it
		Value:          val,
		Diagnostics:    diagnostics,
		EvalOptions:    o,
	}

	// If the evaluation returned a boolean, set the Result's value,
	// otherwise keep the default, true
	if pass, ok := val.(bool); ok {
		u.ExpressionPass = pass
	}

	// By default, the rule's pass/fail is determined by the pass/fail of the
	// expression. If the rule has child rules, we'll iterate through them next
	// and change the rule's pass/fail (but not expresion pass/fail) if any child
	// rules are negative.
	u.Pass = u.ExpressionPass

	// We've been asked not to evaluate child rules if this rule failed.
	if o.StopIfParentNegative && !u.ExpressionPass {
		return u, nil
	}

	// count the number of failed and passed children
	var failCount int
	var passCount int

done: // break out of inner switch
	for _, cr := range r.sortChildKeys(o) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if o.ReturnDiagnostics {
				u.RulesEvaluated = append(u.RulesEvaluated, cr)
			}

			result, err := e.Eval(ctx, cr, d, opts...)
			if err != nil {
				return nil, err
			}

			// If the child rule failed, either due to its own expression evaluation
			// or its children, we have encountered a failure, and we'll count it
			switch result.Pass {
			case true:
				passCount++
			case false:
				failCount++
			}

			// Decide if we should return the result or not.
			if (!result.Pass && !o.DiscardFail) ||
				(result.Pass && !o.DiscardPass) {
				u.Results[cr.ID] = result
			}

			if o.StopFirstPositiveChild && result.Pass {
				break done
			}

			if o.StopFirstNegativeChild && !result.Pass {
				break done
			}
		}
	}

	// Based on the results of the child rules, determine the result of the parent rule
	switch r.EvalOptions.TrueIfAny {
	case true:
		// If any of the child rules passed AND the parent's expression passed, the rule passes
		if u.ExpressionPass && passCount > 0 {
			u.Pass = true
		}
	case false:
		// If one or more of child rules failed, we will fail also, regardless of the parent rule's result
		if failCount > 0 {
			u.Pass = false
		}
	}

	return u, nil
}

// Compile uses the Evaluator's compile method to check the rule and its children,
// returning any validation errors. Stores a compiled version of the rule in the
// rule.Program field (if the compiler returns a program).
func (e *DefaultEngine) Compile(r *Rule, opts ...CompilationOption) error {
	if err := validateCompileArguments(r, e); err != nil {
		return err
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

	// TrueIfAny makes a parent rule Pass = true if any of its child rules are true.
	// The default behavior is that a rule is only true if all of its child rules are true, and
	// the parent rule itself is true.
	// Setting TrueIfAny changes this behvior so that the parent rule is true if at least one of its child rules
	// are true, and the parent rule itself is true.
	TrueIfAny bool `json:"true_if_any"`

	// StopIfParentNegative prevents the evaluation of child rules if the parent's expression is false.
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
// Sorting is only performed if the evaluation order of the child rules is important (i.e.,
// if an option such as StopFirstNegativeChild is set).
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

// validateEvalArguments checks the input parameters to engine.Eval
func validateEvalArguments(r *Rule, e *DefaultEngine, d map[string]interface{}) error {

	switch {
	case r == nil:
		return fmt.Errorf("rule is nil")
	case e == nil:
		return fmt.Errorf("engine is nil")
	case e.e == nil:
		return fmt.Errorf("evaluator is nil")
	case d == nil:
		return fmt.Errorf("data is nil")
	default:
		return nil
	}
}

func setSelfKey(r *Rule, d map[string]interface{}) {
	if d == nil {
		return
	}
	// If this rule has a reference to a 'self' object, insert it into the d.
	// If it doesn't, we must remove any existing reference to self, so that
	// child rules do not accidentally "inherit" the self object.
	if r.Self != nil {
		d[selfKey] = r.Self
	} else {
		delete(d, selfKey)
	}
}

// Default the result type to boolean
// This is the result type passed to the evaluator. The evaluator may use it to
// inspect / validate the result it generates.
func defaultResultType(r *Rule) Type {

	switch r.ResultType {
	case nil:
		return Bool{}
	default:
		return r.ResultType
	}

}

// validateEvalArguments checks the input parameters to engine.Eval
func validateCompileArguments(r *Rule, e *DefaultEngine) error {

	switch {
	case r == nil:
		return fmt.Errorf("rule is nil")
	case e == nil:
		return fmt.Errorf("engine is nil")
	case e.e == nil:
		return fmt.Errorf("evaluator is nil")
	default:
		return nil
	}
}
