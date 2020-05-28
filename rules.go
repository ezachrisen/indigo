package rules

import (
	"fmt"
	"strings"
)

// --------------------------------------------------
// Constants

const (
	// Default recursive depth for evaluate.
	// Override with options.
	DefaultDepth = 100

	// If the rule includes a Self object, it will be made available in the input
	// data with the key SelfKey.
	SelfKey = "self"
)

// --------------------------------------------------
// Rules Engine

// The Engine interface represents a rules engine capable of evaluating rules
// against supplied data.

type Engine interface {

	// Add the rule(s) to the engine.
	// The rule must have an ID that is unique among the root rules.
	// An existing rule with the same ID will be replaced.
	AddRule(...Rule) error

	// Return the rule with the ID
	Rule(id string) (Rule, bool)

	// Remove the rule with the ID
	RemoveRule(id string)

	// Number of (top level) rules
	RuleCount() int

	// Evaluate a rule agains the the data.
	// See the list of options available.
	// Note that the options can be overriden by individual rules.
	// Will recursively evaluate child rules up to the default limit (see const DefaultDepth)
	// or the option. No error is returned if the limit is reached.
	Evaluate(data map[string]interface{}, id string, opts ...Option) (*Result, error)

	// Calculate the expression and return the result.
	Calculate(data map[string]interface{}, expr string, schema Schema) (float64, error)
}

// Rule Evaluation Options
// Pass these to Evaluate to change the behavior of the engine.
// All of these options can also be specified at the individual rule level. TODO: HOW?
// If specified at the rule level, the option applies to the rule and its child rules.

// Determines how far to recurse through child rules.
// The default is rules.DefaultDepth.
func MaxDepth(d int) Option {
	return func(f *EvalOptions) {
		f.MaxDepth = d
	}
}

// Does not evaluate child rules if the parent's expression is false.
// Use case: apply a "global" rule to all the child rules.
func StopIfParentNegative(b bool) Option {
	return func(f *EvalOptions) {
		f.StopIfParentNegative = b
	}
}

// Stops the evaluation of child rules when the first positive child is encountered.
// Results will be partial. Only the child rules that were evaluated will be in the results.
// Use case: role-based access; allow action if any child rule (permission rule) allows it.
func StopFirstPositiveChild(b bool) Option {
	return func(f *EvalOptions) {
		f.StopFirstPositiveChild = b
	}
}

// Stops the evaluation of child rules when the first negative child is encountered.
// Results will be partial. Only the child rules that were evaluated will be in the results.
// Use case: you require ALL child rules to be satisifed.
func StopFirstNegativeChild(b bool) Option {
	return func(f *EvalOptions) {
		f.StopFirstNegativeChild = b
	}
}

// Return rules that passed.
// By default this is on.
func ReturnPass(b bool) Option {
	return func(f *EvalOptions) {
		f.ReturnPass = b
	}
}

// Return rules that did not pass.
// By default this is on.
func ReturnFail(b bool) Option {
	return func(f *EvalOptions) {
		f.ReturnFail = b
	}
}

// Internal use
func ApplyOptions(o *EvalOptions, opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

// Internal use
type EvalOptions struct {
	MaxDepth               int
	StopIfParentNegative   bool
	StopFirstPositiveChild bool
	StopFirstNegativeChild bool
	ReturnPass             bool
	ReturnFail             bool
}

type Option func(f *EvalOptions)

// --------------------------------------------------
// A Rule defines an expression that can be evaluated by the
// Engine. The language used in the expression is dependent
// on the implementation of Engine.
//
// A Rule also has child rules, so that a Rule can serve the
// purpose of being a container for other rules.
//
// Example Structures
//
//   Rule with expression, no child rules
//   - Expression is evaluated and result returned
//
//  Rule with expression and child rules
//  - Expression is evaluated, and so are the child
//    rules.
//  - User can use the expression result to determine what
//    to do with the results of the children

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

	// The schema describing the data provided in the Evaluate input. (optional)
	// Some implementations of Rules require a schema.
	Schema Schema

	// A reference to an object whose values can be used in the rule expression.
	// This is useful to allow dynamic value substitution in the expression.
	// For example, in a rule determining if the temperature is in the specified range,
	// set Self = Thermostat, where Thermostat holds the user's temperature preference.
	// When evaluating the rules, the Thermostat object will be included in the data
	// with the key rules.SelfKey. Rules can then reference the user's temperature preference
	// in the rule expression. This allows rule inputs to be changed without recompiling
	// the rule.
	Self interface{}

	// Actions to execute when this rule qualifies and the client specifies that
	// actions should be executed.
	Action       Doer
	AsynchAction Doer

	// A set of child rules.
	Rules map[string]Rule

	// A reference to a custom object. The same reference will be returned in the results.
	Meta interface{}

	// Options
	// Set options for how the engine should evaluate this rule and the child
	// rules. Options will be inherited by all children, but the children can
	// override with options of their own.
	Opts []Option
}

// Doer performs an action as a result of a rule qualifying.
type Doer interface {
	Do(map[string]interface{}) error
}

// Result of evaluating a rule.
type Result struct {
	// The ID of the rule that was evaluated
	RuleID string

	// Whether the rule yielded a TRUE logical value.
	// The default is FALSE
	// This is the result of evaluating THIS rule only.
	// The result will not be affected by the results of the child rules.
	// If no rule expression is supplied for a rule, the result will be TRUE.
	Pass bool

	// User-supplied reference. This reference is returned in the results
	// along with the outcome of the rule.
	Meta interface{}

	Action Doer

	// The result of the evaluation. Boolean for logical expressions.
	// Calculations or string manipulations will return the appropriate type.
	Value interface{}

	// Results of evaluating the child rules.
	Results map[string]Result

	// Recursion depth
	// How far down in the rule / child rule hierarchy this result is from.
	// The first level is 0.
	Depth int
}

func PrintResults(r *Result, n ...int) {

	if len(n) == 0 {
		fmt.Println("---------- Result Diagnostic ----------")
		n = append(n, 0)
	}
	indent := strings.Repeat(" ", (n[0] * 3))
	boolString := "PASS"
	if !r.Pass {
		boolString = "FAIL"
	}
	fmt.Printf("%-60s %-10s\n", fmt.Sprintf("%s%s", indent, r.RuleID), boolString)
	for _, c := range r.Results {
		PrintResults(&c, n[0]+1)
	}

}

// --------------------------------------------------
// Schema

// Schema defines the keys (variable names) and their data types used in a
// rule expression. The same keys and types must be supplied in the data map
// when rules are evaluated.
type Schema struct {
	ID       string
	Elements []DataElement
}

// DataElement defines a named variable in a schema
type DataElement struct {
	// Short, user-friendly name of the variable. This is the name
	// that will be used in rules to refer to data passed in.
	//
	// RESERVED NAMES:
	//   SelfKey (see const)
	Name string

	// One of the Type interface defined.
	Type Type

	// Optional description of the type.
	Description string
}

// --------------------------------------------------
// Data Types in a Schema
// Not all implementations of Engine support all types.

// Type of data element represented in a schema
type Type interface {
	TypeName()
}

type String struct{}
type Int struct{}
type Float struct{}
type Any struct{}
type Bool struct{}
type Duration struct{}
type Timestamp struct{}

type Proto struct {
	Protoname string
	Message   interface{}
}

// List is an array of items
type List struct {
	ValueType Type
}

// Map is a map of items. Maps can be nested.
type Map struct {
	KeyType   Type
	ValueType Type
}

// Dummy implementations to satisfy the Type interface
func (t Int) TypeName()       {}
func (t Bool) TypeName()      {}
func (t String) TypeName()    {}
func (t List) TypeName()      {}
func (t Map) TypeName()       {}
func (t Any) TypeName()       {}
func (t Duration) TypeName()  {}
func (t Timestamp) TypeName() {}
func (t Float) TypeName()     {}
func (t Proto) TypeName()     {}
