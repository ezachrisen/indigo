package indigo

import (
	"fmt"
	"sort"
	"strings"
)

type Engine struct {

	// The rules map holds the rules passed by the user of the engine
	rules map[string]*Rule

	// The Evaluator that will be used to evaluate rules in this engine
	evaluator Evaluator

	// Options used by the engine during compilation and evaluation
	opts EngineOptions
}

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

		e.rules[r.ID] = r
	}
	return nil
}

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

	err := e.evaluator.Compile(id, r.Expr, r.ResultType, s, e.opts.CollectDiagnostics)
	if err != nil {
		return err
	}

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
func (e *Engine) Rule(id string) (*Rule, bool) {
	r, ok := e.rules[id]
	return r, ok
}

// Rules provides a reference to the rules held in the engine.
// Callers should not attempt to modify the rules in the map.
// Doing so will lead to unexpected results, as rules must be compiled
// and added in a particular way for rule evaluation to work.
func (e *Engine) Rules() map[string]*Rule {
	return e.rules
}

// RuleCount is the number of rules in the engine.
func (e *Engine) RuleCount() int {
	return len(e.rules)
}

// Evaluate the rule against the input data.
// All rules will be evaluated, descending down through child rules up to the maximum depth
// Set EvalOptions to control which rules are evaluated, and what results are returned.
func (e *Engine) Evaluate(data map[string]interface{}, id string, opts ...EvalOption) (*Result, error) {

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

	rule, ok := e.rules[id]
	if !ok {
		return nil, fmt.Errorf("Rule not found")
	}

	return e.eval(data, rule, "", 0, o)
}

// See the functional definitions below for the meaning.
type EngineOptions struct {
	CollectDiagnostics       bool
	ForceDiagnosticsAllRules bool
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

// Recursively evaluate the rule and its children
func (e *Engine) eval(data map[string]interface{}, rule *Rule, parentID string, n int, opt EvalOptions) (*Result, error) {

	if n > opt.MaxDepth {
		return nil, nil
	}

	pr := Result{
		RuleID:      rule.ID,
		Meta:        rule.Meta,
		Action:      rule.Action,
		AsyncAction: rule.AsynchAction,
		Pass:        true,
		Results:     make(map[string]*Result, len(rule.Rules)),
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
				pr.Results[c.ID] = res
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
