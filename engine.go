package indigo

import (
	"fmt"
	"strings"
	"sync"
)

// Engine is the Indigo rules engine.
// It maintains a list of rules and evaluates them against
// data, producing results.
type Engine struct {
	// The root rule holds all rules passed by the user of the engine
	// It is initiated with the default evaluation options
	root *Rule

	// Mutex for the root rule
	mu sync.RWMutex

	// The Evaluator that will be used to evaluate rules in this engine
	evaluator Evaluator

	// Options used by the engine during compilation and evaluation
	opts EngineOptions
}

// Initialize a new engine
func NewEngine(evaluator Evaluator, opts ...EngineOption) *Engine {
	root := NewRule("/")
	root.EvalOpts = []EvalOption{MaxDepth(defaultDepth),
		ReturnFail(true),
		ReturnPass(true)}

	engine := Engine{
		evaluator: evaluator,
		root:      root,
	}
	applyEngineOptions(&engine.opts, opts...)
	return &engine
}

// AddRule compiles the rule and adds it to the engine, ready to be evaluated.
// If a rule does not have a schema, it inherits its parent's schema.
// If a rules exists, an error is returned to make certain the user's intent.
// Use ReplaceRule to replace an existing rule.
func (e *Engine) AddRule(path string, r *Rule) error {

	if r == nil {
		return fmt.Errorf("rule required")
	}

	err := e.compileRule(r)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	_, c, ok := e.root.FindRule(path)
	if !ok || c == nil {
		return fmt.Errorf("rule not found: '%s'", path)
	}
	err = c.AddChild(r)
	if err != nil {
		return err
	}

	return nil
	//	return e.addRule(c, r, "", add)
}

// ReplaceRule replaces the rule identified by path
// If the rule is not found, an error is returned to make certain the user's intent.
func (e *Engine) ReplaceRule(path string, n *Rule) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	p, c, ok := e.root.FindRule(path)
	if !ok {
		return fmt.Errorf("rule not found: '%s'", path)
	}

	err := p.ReplaceChild(c.ID, n)
	if err != nil {
		return err
	}
	return nil
}

func (e *Engine) DeleteRule(path string) error {
	return e.deleteRule(e.root, path)
}

// Find a rule with the given path.
// Returns a copy of the rule, or an empty rule if none is found.
func (e *Engine) Rule(path string) (*Rule, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, c, ok := e.root.FindRule(path)
	if !ok {
		fmt.Println("returning balnk 1")
		return NewRule(""), false
	}

	if c == nil {
		fmt.Println("returing balnk 2")
		return NewRule(""), false
	}

	return c.Copy(), true
}

func (e *Engine) AllParents(id string) []*Rule {
	return e.root.FindRuleParents(id)
}

// RuleCount is the number of rules in the engine.
func (e *Engine) RuleCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.root.CountRules()
}

// --------------------------------------------------
// Unexported functions

func (e *Engine) deleteRule(root *Rule, path string) error {

	e.mu.Lock()
	defer e.mu.Unlock()

	p, c, _ := root.FindRule(path)
	if p == nil {
		return fmt.Errorf("Rule with path '%s' not found", path)
	}

	err := p.DeleteChild(c)
	if err != nil {
		return err
	}

	return nil
}

func (e *Engine) compileRule(r *Rule) error {

	if len(strings.Trim(r.ID, " ")) == 0 {
		return fmt.Errorf("missing rule ID (expr '%s')", r.Expr)
	}

	if strings.ContainsAny(r.ID, bannedIDCharacters) {
		return fmt.Errorf("invalid rule ID (%s), cannot contain any of '%s'", r.ID, bannedIDCharacters)
	}

	var o EvalOptions

	// TODO: Should we inherit the parent rule's options?
	// Previous implementation did
	// applyEvalOptions(&o, p.EvalOpts...)
	applyEvalOptions(&o, r.EvalOpts...)

	// TODO: consider inherting schemas again
	// if len(r.Schema.Elements) == 0 {
	// 	r.Schema = p.Schema
	// }

	// We make sure that the child keys are sorted per the rule's options
	// The evaluator's compiler may rely on the rule's sort order.
	r.SortChildKeys()

	err := e.evaluator.Compile(r, e.opts.CollectDiagnostics, e.opts.DryRun)
	if err != nil {
		return err
	}

	for _, k := range r.sortedKeys {
		child := r.Rules[k]
		err := e.compileRule(child)
		if err != nil {
			return err
		}
	}

	return nil
}

// Evaluate the rule against the input data.
// All rules will be evaluated, descending down through child rules up to the maximum depth
// Set EvalOptions to control which rules are evaluated, and what results are returned.
func (e *Engine) Evaluate(data map[string]interface{}, id string, opts ...EvalOption) (*Result, error) {

	e.mu.RLock()
	defer e.mu.RUnlock()

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

	rule, ok := e.root.Rules[id]
	if !ok || rule == nil {
		return nil, fmt.Errorf("rule not found: '%s'", id)
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
	if rule == nil {
		return nil, fmt.Errorf("eval: rule is null, not found in parent '%s'", parentID)
	}

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

	//	id := makeChildRuleID(parentID, rule.ID)

	value, diagnostics, err := e.evaluator.Eval(data, rule, opt)
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
		c, ok := rule.Rules[k]
		if !ok {
			return nil, fmt.Errorf("Evaluate: rule with id '%s' not found in parent '%s'", k, rule.ID)
		}
		res, err := e.eval(data, c, rule.ID, n+1, opt)
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

// func makeChildRuleID(parentID string, childID string) string {
// 	if parentID == "" {
// 		return childID
// 	}

// 	return parentID + idPathSeparator + childID
// }

// func originalRuleID(id string) string {
// 	if id == "" {
// 		return ""
// 	}

// 	parts := strings.Split(id, idPathSeparator)
// 	return parts[len(parts)-1]
// }
