package indigo

import (
	"sort"
	"strings"

	"github.com/ezachrisen/indigo/schema"
)

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
	// The identifier is used identify results of evaluation
	ID string

	// The expression to evaluate (optional)
	// The expression can return a boolean (true or false), or any
	// other value the underlying expression engine can produce.
	// All values are returned in the Results.Value field.
	// Boolean values are also returned in the results as Pass = true  / false
	// If the expression is blank, the result will be true.
	// (This is common if the rule is used as a container for other rules)
	Expr string

	// Program is a compiled representation of the expression.
	// Evaluators may attach a compiled representation of the rule
	// during compilation. Depending on the Evaluator, you may serialize
	// and store the Program between evaluations. This is useful if the time
	// it takes to compile the rule set is so large that restoring a previously
	// compiled version is significantly faster.
	Program interface{}

	// The output type of the expression. Compilers with the ability to check
	// whether an expression produces the desired output should return an error
	// if the expression does not. If you are using an underlying rules engine
	// that does not support type checking, this value is for your information
	// only.
	ResultType schema.Type

	// The schema describing the data provided in the Evaluate input. (optional)
	// Some implementations of Rules require a schema.
	Schema schema.Schema

	// A reference to an object whose values can be used in the rule expression.
	// This is useful to allow dynamic value substitution in the expression.
	// For example, in a rule determining if the temperature is in the specified range,
	// set Self = Thermostat, where Thermostat holds the user's temperature preference.
	// When evaluating the rules, the Thermostat object will be included in the data
	// with the key selfKey (see constants). Rules can then reference the user's temperature preference
	// in the rule expression. This allows rule inputs to be changed without recompiling
	// the rule.
	Self interface{}

	// A set of child rules.
	Rules map[string]*Rule

	// Sorted list of child rules.
	// Child rules will be evaluated in this order.
	sortedRules []*Rule

	// A reference to any object.
	// Useful for external processes to attach an object to the rule.
	// Not used by the rules engine, but returned to you via the Result.
	Meta interface{}

	// Set options for how the engine should evaluate this rule and the child
	// rules. Options will be inherited by all children, but the children can
	// override with options of their own.
	//EvalOpts []EvalOption
	Options RuleOptions
}

// RuleOptions determine how a rule is evaluated, and how its results are
// returned.
// By default all options are turned off.
type RuleOptions struct {
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

// sortChildKeys sorts the IDs of the child rules according to the
// SortFunc set in evaluation options. If none is set, the evaluation
// order is not specified. Sorting child keys slows down evaluation.
func (r *Rule) sortChildKeys() {

	r.sortedRules = make([]*Rule, 0, len(r.Rules))
	for k := range r.Rules {
		r.sortedRules = append(r.sortedRules, r.Rules[k])
	}

	if r.Options.SortFunc != nil {
		sort.Slice(r.sortedRules, func(i, j int) bool {
			return r.Options.SortFunc(r.sortedRules, i, j)
		})
	}
}

// DescribeStructure returns a list of all the rules in hierarchy, with
// child rules sorted in evaluation order.
// This is useful for visualizing a rule hierarchy.
func (r *Rule) Describe() string {
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

// RuleFunc is a function type that can be applied to all rules in a hierarchy
// using the ApplyFunc function
type RuleFunc func(*Rule) error

// ApplyFunc applies the function f to all rules in the rule tree
func (r *Rule) ApplyFunc(f RuleFunc) error {
	err := f(r)
	if err != nil {
		return err
	}
	for _, c := range r.Rules {
		err := c.ApplyFunc(f)
		if err != nil {
			return err
		}
	}
	return nil
}
