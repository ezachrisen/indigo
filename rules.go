package rules

import (
	"fmt"
)

// --------------------------------------------------
// Rules Engine

// The Engine interface represents a rules engine capable of evaluating rules.
// against a specific rule set.
type Engine interface {
	// Add the rule set to the engine, likely performing error checks or compilation
	AddRuleSet(r *RuleSet) error

	// Return the rule set if found
	RuleSet(ruleSetID string) (*RuleSet, bool)

	// Evaluate a single rule agains the the data
	EvaluateRule(data map[string]interface{}, r *Rule) (*Result, error)

	// Evaluate all rules against the data
	EvaluateAll(data map[string]interface{}, ruleSetID string) ([]Result, error)

	// Evaluate rules, but stop at the first true rule
	EvaluateUntilTrue(data map[string]interface{}, ruleSetID string) (*Result, error)
}

// Result of evaluating a rule. A slice of these will be returned after evaluating a rule set.
// See the documentation for the Evaluate* methods for information on the
// result set.
type Result struct {
	Rule *Rule // The rule that was evaluated
	Pass bool  // Whether the expression was satisfied by the input data
}

// These functions are intended to be called by implementors of the Engine interface.
// Engines are free to create their own implementations.

// Evaluate all rules in a rule set and return the true/false results of each rule
func EvaluateAll(e Engine, data map[string]interface{}, ruleSetID string) ([]Result, error) {

	ruleSet, found := e.RuleSet(ruleSetID)
	if !found {
		return nil, fmt.Errorf("Ruleset %v not found", ruleSet)
	}

	results := make([]Result, 0, len(ruleSet.Rules))

	for i := range ruleSet.Rules {
		result, err := e.EvaluateRule(data, &ruleSet.Rules[i])
		if err != nil {
			return results, fmt.Errorf("Error evaluating rule: %v", err)
		}
		results = append(results, *result)
	}
	return results, nil
}

// Evaluate rules in the rule set, but stop as soon as a true rule is found. The true rule is returned.
// If no true rules are found, the result is nil.
func EvaluateUntilTrue(e Engine, data map[string]interface{}, ruleSetID string) (*Result, error) {
	ruleSet, found := e.RuleSet(ruleSetID)
	if !found {
		return nil, fmt.Errorf("Ruleset %v not found", ruleSet)
	}

	for i := range ruleSet.Rules {
		result, err := e.EvaluateRule(data, &ruleSet.Rules[i])
		if err != nil {
			return nil, fmt.Errorf("Error evaluating rule: %v", err)
		}
		if result.Pass {
			return result, nil
		}
	}
	return nil, nil
}

// --------------------------------------------------
// Rules

// Rule is an evaluable unit of logic represented as a text expression
// (see https://pkg.go.dev/github.com/google/cel-go/cel for documentation)
type Rule struct {
	ID         string
	Expression string
}

// RuleSet contains a group of rules that will be evaluated together to produce results.
type RuleSet struct {
	ID     string
	Rules  []Rule // The rules to evaluate
	Schema Schema // The data schema that all rules and data must adhere to
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
	ID          string
	Name        string // Short, user-friendly name of the variable
	Key         string // Name of the key used in the data and the rule expressions
	Type        Type
	Description string
}

// --------------------------------------------------
// Data Types in a Schema

// Type of data element represented in a schema
type Type interface {
	TypeName()
}

type String struct{}
type Int struct{}
type Any struct{}
type Bool struct{}
type Duration struct{}
type Timestamp struct{}

// List is an array of items
type List struct {
	ValueType Type // The type of value stored in the list
}

// Map is a map of items. Maps can be nested.
type Map struct {
	KeyType   Type
	ValueType Type
}

func (t Int) TypeName()       {}
func (t Bool) TypeName()      {}
func (t String) TypeName()    {}
func (t List) TypeName()      {}
func (t Map) TypeName()       {}
func (t Any) TypeName()       {}
func (t Duration) TypeName()  {}
func (t Timestamp) TypeName() {}
