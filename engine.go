package indigo

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Engine struct {

	// The rules map holds the rules passed by the user of the engine
	rules map[string]*Rule

	// Mutex for the rules map
	mu sync.RWMutex

	// The Evaluator that will be used to evaluate rules in this engine
	evaluator Evaluator

	// Options used by the engine during compilation and evaluation
	opts EngineOptions
}

var ErrRuleNotFound = errors.New("rule not found")

// Initialize a new engine
func NewEngine(evaluator Evaluator, opts ...EngineOption) *Engine {
	engine := Engine{
		evaluator: evaluator,
		rules:     make(map[string]*Rule),
	}
	applyEngineOptions(&engine.opts, opts...)
	return &engine
}

// AddRule compiles the rule and adds it to the engine, ready to be evaluated.
// If a rule does not have a schema, it inherits its parent's schema.
func (e *Engine) AddRule(rules ...*Rule) error {

	for i := range rules {
		r := rules[i]
		if len(strings.Trim(r.ID, " ")) == 0 {
			return fmt.Errorf("Required rule ID for rule with expression %s", r.Expr)
		}

		if strings.ContainsAny(r.ID, bannedIDCharacters) {
			return fmt.Errorf("Rule ID is invalid (%s),  cannot contain any of '%s'", r.ID, bannedIDCharacters)
		}

		o := EvalOptions{
			MaxDepth:   defaultDepth,
			ReturnFail: true,
			ReturnPass: true,
		}

		applyEvalOptions(&o, r.EvalOpts...)

		// We pass in the rule's schema at the top of the rule tree.
		// Child rules will inherit the schema of their parent if
		// they do not have one specified.
		err := e.addRuleWithSchema(r, "", r.Schema, o)
		if err != nil {
			return err
		}

		e.mu.Lock()
		e.rules[r.ID] = r
		e.mu.Unlock()
	}
	return nil
}

func (e *Engine) ReplaceRule(path string, n *Rule) error {

	elems := strings.Split(path, "/")

	if len(elems) == 0 {
		return fmt.Errorf("missing path argument")
	}

	// From now on we must lock the rules so no other
	// modification can take place
	e.mu.Lock()
	defer e.mu.Unlock()

	root, ok := e.rules[elems[0]]
	if !ok {
		return fmt.Errorf("rule with path '%s' not found", path)
	}

	r, ok := root.FindChild(strings.Join(elems[1:len(elems)], "/"))
	if !ok {
		return fmt.Errorf("rule with path '%s' not found", path)
	}

	//	e.prepareRule(n)
	*r = *n
	return nil

}

// childKeys extracts the keys from a map of rules
// The resulting slice of keys is used to sort rules
// when rules are added to the engine
func childKeys(r map[string]*Rule) []string {
	keys := make([]string, 0, len(r))
	for k := range r {
		keys = append(keys, k)
	}

	return keys
}

func (e *Engine) addRuleWithSchema(r *Rule, parentRuleID string, s Schema, o EvalOptions) error {

	applyEvalOptions(&o, r.EvalOpts...)

	// If the rule has a schema, use it, otherwise use the parent rule's
	if len(r.Schema.Elements) > 0 {
		s = r.Schema
	}

	// In the Evaluator world, rules are not necessarily stored in a nested hierarchy
	// Therefore we must ensure that rule IDs are globally unique
	id := makeChildRuleID(parentRuleID, r.ID)
	//	start := time.Now()

	err := e.evaluator.Compile(id, r.Expr, r.ResultType, s, e.opts.CollectDiagnostics, e.opts.DryRun)
	if err != nil {
		return err
	}
	//	fmt.Printf("   Finished compile in %s\n", time.Since(start))

	r.sortedKeys = childKeys(r.Rules)
	if o.SortFunc != nil {
		sort.Slice(r.sortedKeys, o.SortFunc)
	} else {
		sort.Strings(r.sortedKeys)
	}

	for _, k := range r.sortedKeys {
		child := r.Rules[k]
		err := e.addRuleWithSchema(child, id, s, o)
		if err != nil {
			return err
		}
	}
	return nil
}

// Find a rule with the given ID
// Returns a copy of the rule
func (e *Engine) Rule(id string) (*Rule, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	r, ok := e.rules[id]
	if !ok {
		return nil, false
	}

	return copyRule(r), true
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
// Returns a pointer to the rule itself
// CAUTION
func (e *Engine) RuleWithPath(path string) (*Rule, bool) {
	if len(strings.Trim(path, " ")) == 0 {
		return nil, false
	}

	elems := strings.Split(path, "/")

	e.mu.RLock()
	r, ok := e.rules[elems[0]]
	e.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if len(elems) == 1 {
		return copyRule(r), true
	}

	c, ok := r.FindChild(strings.Join(elems[1:len(elems)], "/"))
	if !ok {
		return nil, false
	}
	return c, true
}

// Rules provides a copy of the engine's rules.
// If the engine contains a lot of rules, this is an expensive
// operation.
func (e *Engine) Rules() map[string]*Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return copyRules(e.rules)
}

func copySortedKeys(a []string) []string {
	b := make([]string, len(a))
	for _, s := range a {
		b = append(b, s)
	}
	return b
}

func copyEvalOpts(a []EvalOption) []EvalOption {
	b := make([]EvalOption, len(a))
	for _, o := range a {
		b = append(b, o)
	}
	return b
}

// RuleCount is the number of rules in the engine.
func (e *Engine) RuleCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.rules)
}

// Evaluate the rule against the input data.
// All rules will be evaluated, descending down through child rules up to the maximum depth
// Set EvalOptions to control which rules are evaluated, and what results are returned.
func (e *Engine) Evaluate(data map[string]interface{}, id string, opts ...EvalOption) (*Result, error) {

	// if data == nil {
	// 	return nil, fmt.Errorf("indigo.Evaluate called with nil data")
	// }

	o := EvalOptions{
		MaxDepth:   defaultDepth,
		ReturnFail: true,
		ReturnPass: true,
	}

	applyEvalOptions(&o, opts...)

	if e.opts.ForceDiagnosticsAllRules {
		o.ReturnDiagnostics = true
	}

	if o.ReturnDiagnostics && !e.opts.CollectDiagnostics {
		return nil, fmt.Errorf("option set to return diagnostic, but engine does not have CollectDiagnostics option set")
	}

	e.mu.RLock()
	rule, ok := e.rules[id]
	e.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrRuleNotFound, id)
	}

	return e.eval(data, rule, "", 0, o)
}

// See the functional definitions below for the meaning.
type EngineOptions struct {
	CollectDiagnostics       bool
	ForceDiagnosticsAllRules bool
	DryRun                   bool
}

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
// By default this is off.
func DryRun(b bool) EngineOption {
	return func(f *EngineOptions) {
		f.DryRun = b
	}
}

// Recursively evaluate the rule and its children
func (e *Engine) eval(data map[string]interface{}, rule *Rule, parentID string, n int, opt EvalOptions) (*Result, error) {

	if n > opt.MaxDepth {
		return nil, nil
	}

	pr := Result{
		RuleID:  rule.ID,
		Meta:    rule.Meta,
		Pass:    true,
		Results: make(map[string]*Result, len(rule.Rules)),
	}

	// Apply options for this rule evaluation
	applyEvalOptions(&opt, rule.EvalOpts...)

	// If this rule has a reference to a 'self' object, insert it into the data.
	// If it doesn't, we must remove any existing reference to self, so that
	// child rules do not accidentally "inherit" the self object.
	if rule.Self != nil {
		data[selfKey] = rule.Self
	} else {
		delete(data, selfKey)
	}

	id := makeChildRuleID(parentID, rule.ID)

	value, diagnostics, err := e.evaluator.Eval(data, id, rule.Expr, rule.ResultType, opt)
	pr.RulesEvaluated++

	if err != nil {
		return nil, err
	}

	pr.Value = value.Val // TODO: Use the indigo.Value
	pr.Diagnostics = diagnostics

	dummy := Bool{}

	if value.Typ == dummy {
		pass, ok := value.Val.(bool)
		if !ok {
			return nil, fmt.Errorf("Expected boolean value, got %T, evaluating rule %s", value.Val, rule.ID)
		}
		pr.Pass = pass
	}

	if opt.StopIfParentNegative && pr.Pass == false {
		return &pr, nil
	}

	// Evaluate child rules
	for _, k := range rule.sortedKeys {
		c := rule.Rules[k]
		res, err := e.eval(data, c, id, n+1, opt)
		if err != nil {
			return nil, err
		}
		if res != nil {
			if (!res.Pass && opt.ReturnFail) ||
				(res.Pass && opt.ReturnPass) {
				pr.Results[k] = res
			}
			pr.RulesEvaluated += res.RulesEvaluated
		}

		if opt.StopFirstPositiveChild && res.Pass == true {
			return &pr, nil
		}

		if opt.StopFirstNegativeChild && res.Pass == false {
			return &pr, nil
		}
	}
	return &pr, nil
}

func makeChildRuleID(parentID string, childID string) string {
	if parentID == "" {
		return childID
	}

	return parentID + idPathSeparator + childID
}

func originalRuleID(id string) string {
	if id == "" {
		return ""
	}

	parts := strings.Split(id, idPathSeparator)
	return parts[len(parts)-1]
}
