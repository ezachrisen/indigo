package indigo

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ezachrisen/indigo/schema"
)

// A Rule defines logic that can be evaluated by an Evaluator.
// The logic for evaluation is specified by an expression.
// A rule can have child rules. Rule options specify to the Evaluator
// how child rules should be handled. Child rules can in turn have children,
// enabling you to create a hierarchy of rules.
//
// Example Rule Structures
//
// A hierchy of parent/child rules, combined with evaluation options
// give many different ways of using the rules engine.
//  Rule with expression, no child rules:
//   Parent rule expression is evaluated and result returned.
//
//  Rule with expression and child rules:
//   No options specified
//   - Parent rule xpression is evaluated, and so are all the child rules.
//   - All children and their evaluation results are returned
//
//  Rule with expression and child rules
//  Option set: StopIfParentNegative
//  - Parent rule expression is evaluated
//  - If the parent rule is a boolean, and it returns FALSE,
//    the children are NOT evaluated
//  - If the parent rule returns TRUE, or if it's not a
//    boolean, all the children and their resulsts are returned
//
type Rule struct {
	// A rule identifer. (required)
	ID string

	// The expression to evaluate (optional)
	// The expression can return a boolean (true or false), or any
	// other value the underlying expression engine can produce.
	// All values are returned in the Results.Value field.
	// Boolean values are also returned in the results as Pass = true  / false
	// If the expression is blank, the result will be true.
	Expr string

	// The output type of the expression. Evaluators with the ability to check
	// whether an expression produces the desired output should return an error
	// if the expression does not.
	ResultType schema.Type

	// The schema describing the data provided in the Evaluate input. (optional)
	// Some implementations of Evaluator require a schema.
	Schema schema.Schema

	// A reference to an object whose values can be used in the rule expression.
	// Add the corresponding object in the data with the reserved key name selfKey
	// (see constants).
	// See example for usage. TODO: example
	Self interface{}

	// A set of child rules.
	Rules map[string]*Rule

	// Reference to intermediate compilation / evaluation data.
	// Set
	program interface{}

	// A reference to any object.
	// Not used by the rules engine.
	Meta interface{}

	// StopIfParentNegative  not evaluate child rules if the parent's expression is false.
	// Use case: apply a "global" rule to all the child rules.
	StopIfParentNegative bool

	// Stops the evaluation of child rules when the first positive child is encountered.
	// Results will be partial. Only the child rules that were evaluated will be in the results.
	// By default rules are evaluated in alphabetical order.
	// Use case: role-based access; allow action if any child rule (permission rule) allows it.
	StopFirstPositiveChild bool

	// Stops the evaluation of child rules when the first negative child is encountered.
	// Results will be partial. Only the child rules that were evaluated will be in the results.
	// By default rules are evaluated in alphabetical order.
	// Use case: you require ALL child rules to be satisifed.
	StopFirstNegativeChild bool

	// Do not return rules that passed
	// Default: all rules are returned
	DiscardPass bool

	// Do not return rules that failed
	// Default: all rules are returned
	DiscardFail bool

	// Include diagnostic information with the results.
	// To enable this option, you must first turn on diagnostic
	// collection at the engine level with the CollectDiagnostics EngineOption.
	ReturnDiagnostics bool

	// Specify the function used to sort the child rules before evaluation.
	// Useful in scenarios where you are asking the engine to stop evaluating
	// after either the first negative or first positive child.
	// See the provided SortAlpha function as an example.
	// Default: No sort
	SortFunc func(rules []*Rule, i, j int) bool
}

const (
	// If the rule includes a Self object, it will be made available in the input
	// data with this key name.
	selfKey = "self"
)

// NewRule initializes a rule with the given ID
func NewRule(id string) *Rule {
	return &Rule{
		ID:    id,
		Rules: map[string]*Rule{},
	}
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
func (r *Rule) Compile(c Compiler, opts ...CompilerOption) error {

	if c == nil {
		return fmt.Errorf("Compile: Compiler is nil")
	}

	o := compileOptions{}
	applyCompilerOptions(&o, opts...)

	prg, err := c.Compile(r, o.collectDiagnostics, o.dryRun)
	if err != nil {
		return err
	}

	if !o.dryRun {
		r.program = prg
	}

	for _, cr := range r.Rules {
		err := cr.Compile(c, opts...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Rule) Evaluate(e Evaluator, d map[string]interface{}, opts ...EvaluatorOption) (*Result, error) {

	if r == nil {
		return nil, fmt.Errorf("Evaluate: rule is nil")
	}

	if e == nil {
		return nil, fmt.Errorf("Evaluate: evaluator is nil")
	}

	if d == nil {
		return nil, fmt.Errorf("Evaluate: data is nil")
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

	val, diagnostics, err := e.Evaluate(d, r, r.program, o.returnDiagnostics)
	if err != nil {
		return nil, err
	}

	pr := &Result{
		Rule:        r,
		Pass:        true,
		Value:       val.Val,
		Diagnostics: diagnostics,
		Results:     make(map[string]*Result, len(r.Rules)), // TODO: consider how large to make it
	}

	// TODO: check that we got the expected value type
	if pass, ok := val.Val.(bool); ok {
		pr.Pass = pass
	}

	if r.StopIfParentNegative && pr.Pass == false {
		return pr, nil
	}

	for _, cr := range r.sortChildKeys() {
		result, err := cr.Evaluate(e, d, opts...)
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

type compileOptions struct {
	dryRun             bool
	collectDiagnostics bool
}

type CompilerOption func(f *compileOptions)

// Perform all compilation steps, but do not save the results.
// This is to allow a client to check all rules in a rule tree before
// committing the actual compilation results to the rule.
func DryRun(b bool) CompilerOption {
	return func(f *compileOptions) {
		f.dryRun = b
	}
}

// CollectDiagnostics instructs the engine and its evaluator to save any
// intermediate results of compilation in order to provide good diagnostic
// information after evaluation. Not all evaluators need to have this option set.
// Default: off
func CollectDiagnostics(b bool) CompilerOption {
	return func(f *compileOptions) {
		f.collectDiagnostics = b
	}
}

// Given an array of EngineOption functions, apply their effect
// on the engineOptions struct.
func applyCompilerOptions(o *compileOptions, opts ...CompilerOption) {
	for _, opt := range opts {
		opt(o)
	}
}

type evalOptions struct {
	returnDiagnostics bool
}

// EvaluatorOptions determine how the engine behaves during the evaluation .
type EvaluatorOption func(f *evalOptions)

// Include diagnostic information with the results.
// Default: off
func ReturnDiagnostics(b bool) EvaluatorOption {
	return func(f *evalOptions) {
		f.returnDiagnostics = b
	}
}

func applyEvaluatorOptions(o *evalOptions, opts ...EvaluatorOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// String returns a list of all the rules in hierarchy, with
// child rules sorted in evaluation order.
func (r *Rule) String() string {
	return r.describe(0)
}

func (r *Rule) describe(n int) string {
	s := strings.Builder{}
	s.WriteString(strings.Repeat("  ", n)) // indent
	s.WriteString(r.ID)
	s.WriteString("\n")
	for _, c := range r.Rules {
		s.WriteString(c.describe(n + 1))
	}
	return s.String()
}

// sortChildKeys sorts the IDs of the child rules according to the
// SortFunc set in evaluation options. If no SortFunc is set, the evaluation
// order is not specified.
func (r *Rule) sortChildKeys() []*Rule {
	keys := make([]*Rule, 0, len(r.Rules))
	for k := range r.Rules {
		keys = append(keys, r.Rules[k])
	}

	if r.SortFunc != nil {
		sort.Slice(keys, func(i, j int) bool {
			return r.SortFunc(keys, i, j)
		})
	}
	return keys
}
