// Package Indigo provides a rules engine process that uses an instance of the
// Evaluator interface to perform evaluation of rules.
//
// Indigo itself does not specify a language for rules, relying instead on the Evaluator's
// rule language.
//
//
package indigo

import (
	"fmt"
	"strings"
)

const (
	// Default recursive depth for evaluate.
	// Override with options.
	defaultDepth = 100

	// If the rule includes a Self object, it will be made available in the input
	// data with this key name.
	selfKey = "self"

	// Path separator character when creating child rule IDs for use by Evaluator
	idPathSeparator = "/"

	// These characters are not allowed in rule IDs
	bannedIDCharacters = "/"
)

// Evaluator is the interface implemented by types that can evaluate expressions defined in
// the rules. The interface includes a Compile step that the Engine calls before evaluation.
// This gives the Evaluator the option of pre-processing the rule for efficiency. At a minimum
// the Compile step should give the user feedback on whether the provided rule conforms to its
// specificiations.
type Evaluator interface {
	// Compile a rule, checking for correctness and preparing the rule to be
	// evaluated later.
	Compile(ruleID string, expr string, resultType Type, s Schema, collectDiagnostics bool) error

	// Eval tests the rule expression  against the data.
	// The result is one of Indigo's types.
	// resultType: the type of result value the rule expects
	Eval(data map[string]interface{}, ruleID string, expr string, resultType Type, opt EvalOptions) (Value, string, error)
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
	SortFunc               func(i, j int) bool
}

// A Rule defines an expression that can be evaluated by the
// Engine. The language used in the expression is dependent
// on the implementation of Engine.
//
// A Rule can also have child rules, so that a Rule can serve the
// purpose of being a container for other rules. Options can be
// set via EvalOptions on each rule to determine how to process
// the child rules.
//
// Example Rule Structures
//
//   Rule with expression, no child rules
//   - Parent rule expression is evaluated and result returned
//
//  Rule with expression and child rules
//  No options specified
//  - Parent rule xpression is evaluated, and so are all the child rules.
//  - All children and their evaluation results are returned
//
//  Rule with expression and child rules
//  Option set: StopIfParentNegative
//  - Parent rule expression is evaluated
//  - If the parent rule is a boolean, and it returns FALSE,
//    the children are NOT evaluated
//  - If the parent rule returns TRUE, or if it's not a
//    boolean, all the children and their resulsts are returned
type Rule struct {
	// A rule identifer. (required)
	// No two root rules can have the same identifier.
	ID string

	// The expression to evaluate (optional)
	// The expression can return a boolean (true or false), or any
	// other value the underlying expression engine can produce.
	// All values are returned in the Results.Value field.
	// Boolean values are also returned in the results as Pass = true  / false
	// If the expression is blank, the result will be true.
	// (This is common if the rule is used as a container for other rules)
	Expr string

	// The output type of the expression. Compilers with the ability to check
	// whether an expression produces the desired output should return an error
	// if the expression does not. If you are using an underlying rules engine
	// that does not support type checking, this value is for your information
	// only.
	ResultType Type

	// The schema describing the data provided in the Evaluate input. (optional)
	// Some implementations of Rules require a schema.
	Schema Schema

	// A reference to an object whose values can be used in the rule expression.
	// This is useful to allow dynamic value substitution in the expression.
	// For example, in a rule determining if the temperature is in the specified range,
	// set Self = Thermostat, where Thermostat holds the user's temperature preference.
	// When evaluating the rules, the Thermostat object will be included in the data
	// with the key rules.selfKey. Rules can then reference the user's temperaturE preference
	// in the rule expression. This allows rule inputs to be changed without recompiling
	// the rule.
	Self interface{}

	// A set of child rules.
	Rules map[string]*Rule

	// Sorted list of child rule keys.
	// Child rules will be evaluated in this order.
	sortedKeys []string

	// A reference to any object.
	Meta interface{}

	// Set options for how the engine should evaluate this rule and the child
	// rules. Options will be inherited by all children, but the children can
	// override with options of their own.
	EvalOpts []EvalOption
}

func (r *Rule) AddChild(c *Rule) {
	r.Rules[c.ID] = c
}

// Doer performs an action as a result of a rule qualifying.
// Unused at this time
type Doer interface {
	Do(map[string]interface{}) error
}

// Result of evaluating a rule.
type Result struct {
	// The Rule that was evaluated
	RuleID string

	// Reference to a value set when the rule was added to the engine.
	Meta interface{}

	// Whether the rule yielded a TRUE logical value.
	// The default is FALSE
	// This is the result of evaluating THIS rule only.
	// The result will not be affected by the results of the child rules.
	// If no rule expression is supplied for a rule, the result will be TRUE.
	Pass bool

	// The result of the evaluation. Boolean for logical expressions.
	// Calculations or string manipulations will return the appropriate type.
	Value interface{}

	// Results of evaluating the child rules.
	Results map[string]*Result

	// Unused at this time
	Action      Doer
	AsyncAction Doer

	// Diagnostic data
	Diagnostics    string
	RulesEvaluated int
}

type Value struct {
	Val interface{}
	Typ Type
}

// SummarizeResults produces a list of rules (including child rules) executed and the result of the evaluation.
// n[0] is the indent level, passed as a variadic solely to allow callers to omit it
func SummarizeResults(r *Result, n ...int) string {
	s := strings.Builder{}

	if len(n) == 0 {
		s.WriteString("\n---------- Result Diagnostic --------------------------\n")
		s.WriteString("                                         Pass Chil-\n")
		s.WriteString("Rule                                     Fail dren Value\n")
		s.WriteString("--------------------------------------------------------\n")
		n = append(n, 0)
	}
	indent := strings.Repeat(" ", (n[0]))
	boolString := "PASS"
	if !r.Pass {
		boolString = "FAIL"
	}
	s.WriteString(fmt.Sprintf("%-40s %-4s %4d %v\n", fmt.Sprintf("%s%s", indent, r.RuleID), boolString, len(r.Results), r.Value))
	for _, c := range r.Results {
		s.WriteString(SummarizeResults(c, n[0]+1))
	}
	return s.String()
}
