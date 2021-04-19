package indigo

import (
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
	Eval(r *Rule, d map[string]interface{}, opts ...EvalOption) (*Result, error)
}

// DefaultEngine provides an implementation of the Engine interface
// to evaluate rules.
type DefaultEngine struct {
	e Evaluator
}

func NewEngine(e Evaluator) *DefaultEngine {
	return &DefaultEngine{
		e: e,
	}
}

func (e *DefaultEngine) Eval(r *Rule, d map[string]interface{}, opts ...EvalOption) (*Result, error) {
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

	o := evalOptions{}
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

	val, diagnostics, err := e.e.Evaluate(d, r.Expr, r.Schema, r.Self, r.Program, o.returnDiagnostics)
	if err != nil {
		return nil, err
	}

	pr.Value = val.Val
	pr.Diagnostics = diagnostics

	// TODO: check that we got the expected value type
	if pass, ok := val.Val.(bool); ok {
		pr.Pass = pass
	}

	if r.StopIfParentNegative && pr.Pass == false {
		return pr, nil
	}

	for _, cr := range r.sortChildKeys() {
		if o.returnDiagnostics {
			pr.RulesEvaluated = append(pr.RulesEvaluated, cr)
		}
		result, err := e.Eval(cr, d, opts...)
		if err != nil {
			return nil, err
		}

		if (!result.Pass && !r.DiscardFail) ||
			(result.Pass && !r.DiscardPass) {
			pr.Results[cr.ID] = result
		}

		if r.StopFirstPositiveChild && result.Pass == true {
			return pr, nil
		}

		if r.StopFirstNegativeChild && result.Pass == false {
			return pr, nil
		}
	}
	return pr, nil
}

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

	prg, err := e.e.Compile(r.Expr, r.Schema, r.ResultType, o.collectDiagnostics, o.dryRun)
	if err != nil {
		return err
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

// CompilationOptions determines how compilation behaves.
type CompilationOption func(f *compileOptions)

// Perform all compilation steps, but do not save the results.
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

type evalOptions struct {
	returnDiagnostics bool
}

// EvalOptions determine how evaluation behaves.
type EvalOption func(f *evalOptions)

// Include diagnostic information with the results.
// Default: off
func ReturnDiagnostics(b bool) EvalOption {
	return func(f *evalOptions) {
		f.returnDiagnostics = b
	}
}

func applyEvaluatorOptions(o *evalOptions, opts ...EvalOption) {
	for _, opt := range opts {
		opt(o)
	}
}
