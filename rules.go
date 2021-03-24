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
	"sort"
	"strings"
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

	// Program is a compiled representation of the expression.
	// Evaluators may attach a compiled representation of the
	// rule.
	Program interface{}

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
	// Useful for external processes to attach an object to the rule.
	Meta interface{}

	// Set options for how the engine should evaluate this rule and the child
	// rules. Options will be inherited by all children, but the children can
	// override with options of their own.
	EvalOpts []EvalOption
}

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
	bannedIDCharacters = "/ "
)

func NewRule(id string) *Rule {
	return &Rule{
		ID:    id,
		Rules: map[string]*Rule{},
	}
}

func (r *Rule) AddChild(c *Rule) error {
	if c == nil {
		return fmt.Errorf("attempt to add nil rule")
	}

	if _, ok := r.Rules[c.ID]; ok {
		return fmt.Errorf("attempt to overwrite rule '%s' in parent rule '%s'", c.ID, r.ID)
	}
	r.Rules[c.ID] = c
	r.SortChildKeys()
	return nil
}

func (r *Rule) DeleteChild(c *Rule) error {

	if c == nil {
		return fmt.Errorf("attempt to delete nil rule")
	}

	if _, ok := r.Rules[c.ID]; !ok {
		return fmt.Errorf("attempt to delete rule '%s': does not exist", c.ID)
	}
	delete(r.Rules, c.ID)
	r.SortChildKeys()
	return nil
}

func (r *Rule) ReplaceChild(id string, c *Rule) error {
	if c == nil {
		return fmt.Errorf("attempt to replace rule with nil rule")
	}

	if _, ok := r.Rules[id]; !ok {
		return fmt.Errorf("attempt to replace rule '%s': does not exist", id)
	}

	delete(r.Rules, id)
	r.Rules[c.ID] = c
	r.SortChildKeys()
	return nil
}

func (r *Rule) Child(id string) (*Rule, bool) {

	c, ok := r.Rules[id]
	return c, ok
}

func (r *Rule) FindRuleParents(id string) []*Rule {
	rs := []*Rule{}

	_, ok := r.Rules[id]
	if ok {
		rs = append(rs, r)
	}

	for k := range r.Rules {
		crs := r.Rules[k].FindRuleParents(id)
		rs = append(rs, crs...)
	}
	return rs
}

func (parent *Rule) FindRule(path string) (*Rule, *Rule, bool) {

	if path == "" {
		return nil, nil, false
	}

	// Special case: / denotes the root rule in the engine
	if path == "/" {
		return parent, parent, true
	}

	// Strip the first /
	if path[0] == '/' {
		path = path[1:]
	}

	//	fmt.Println("PATH=", path)
	elems := strings.Split(path, "/")
	//fmt.Println("ELEMS=", elems)
	switch len(elems) {
	case 1:
		//fmt.Println("1")
		// We're at the leaf element, "b-1" in the
		// sample path rule1/b/b-1
		if c, ok := parent.Rules[elems[0]]; ok {
			return parent, c, true
		}
		return nil, nil, false
	default:
		//fmt.Println("def:", parent.ID, ">", elems[0])
		// There are more than 1 elements left
		// we are either at rule1 or b in the sample path
		// rule1/b/b1
		// Assuming we're at rule1, c will be = b
		// We recurse further, with b the new parent
		if c, ok := parent.Rules[elems[0]]; ok {
			//fmt.Println("found c in ", parent.ID)
			return c.FindRule(strings.Join(elems[1:], "/"))
		} else {
			return nil, nil, false
		}
	}
}

func (r *Rule) CountRules() int {
	i := 1

	for k := range r.Rules {
		i += r.Rules[k].CountRules()
	}
	return i
}

func (r *Rule) SortChildKeys() {

	var o EvalOptions
	applyEvalOptions(&o, r.EvalOpts...)

	r.sortedKeys = r.childKeys()
	if o.SortFunc != nil {
		sort.Slice(r.sortedKeys, o.SortFunc)
	} else {
		sort.Strings(r.sortedKeys)
	}
}

func (r *Rule) Copy() *Rule {

	nr := Rule{
		ID:         r.ID,
		Expr:       r.ID,
		ResultType: r.ResultType,
		Schema:     r.Schema,
		Self:       r.Self,
		Rules:      r.copyRules(),
		sortedKeys: r.copySortedKeys(),
		Meta:       r.Meta,
		EvalOpts:   r.copyEvalOpts(),
	}

	return &nr
}

// childKeys extracts the keys from a map of rules
// The resulting slice of keys is used to sort rules
// when rules are added to the engine
func (r *Rule) childKeys() []string {
	keys := make([]string, 0, len(r.Rules))
	for k := range r.Rules {
		keys = append(keys, k)
	}
	return keys
}

// Find a rule with the given path
// The rule path is the concatenation of the
// rule IDs in a hierarchy, separated by /
// For example, given this hierarchy of rule IDs:
//  rule1
//    b
//    c
//      c1
//      c2
//
// c1 can be identified with the path
// rule1/c/c1
// Returns a copy of the rule

// func (r *Rule) FindChild(path string) (*Rule, *Rule, bool) {

// 	elems := strings.Split(path, "/")
// 	c, ok := r.Rules[elems[0]]
// 	if !ok {
// 		return nil, nil, false
// 	}

// 	// If we're down to the last path element
// 	if len(elems) == 1 {
// 		return r, c, true
// 	}

// 	return c.FindChild(strings.Join(elems[1:len(elems)], "/"))
// }

func (r *Rule) copyRules() map[string]*Rule {
	mn := make(map[string]*Rule, len(r.Rules))
	for k := range r.Rules {
		cc := r.Rules[k].Copy()
		mn[cc.ID] = cc
	}
	return mn
}

func (r *Rule) copySortedKeys() []string {
	b := make([]string, 0, len(r.sortedKeys))
	for _, s := range r.sortedKeys {
		b = append(b, s)
	}
	return b
}

func (r *Rule) copyEvalOpts() []EvalOption {
	b := make([]EvalOption, len(r.EvalOpts))
	for _, o := range r.EvalOpts {
		b = append(b, o)
	}
	return b
}

func (r *Rule) DescribeStructure(n ...int) string {

	s := strings.Builder{}
	if len(n) == 0 {
		n = append(n, 0)
	}

	s.WriteString(strings.Repeat(" ", n[0])) // indent
	s.WriteString(r.ID)
	s.WriteString("\n")
	if len(r.sortedKeys) != len(r.Rules) {
		s.WriteString(fmt.Sprintf("ERROR: in rule '%s', length of sortedKeys (%d) != length of rules (%d)\n", r.ID, len(r.sortedKeys), len(r.Rules)))
		s.WriteString(fmt.Sprintf("sortedKeys = %s\n", r.sortedKeys))
	}
	for _, k := range r.sortedKeys {
		if c, ok := r.Rules[k]; ok {
			s.WriteString(c.DescribeStructure(n[0] + 1))
		} else {
			s.WriteString("ERROR: missing rule '" + k + "'\n")
		}
	}
	return s.String()
}
