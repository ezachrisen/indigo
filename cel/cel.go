// package cel provides an implementation of the rules/Engine interface backed by Google's cel-go rules engine
package cel

import (
	"fmt"

	"github.com/ezachrisen/rules"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	exprbp "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// celType converts from a rules.Type to a CEL type
func celType(t rules.Type) *exprbp.Type {

	switch v := t.(type) {
	case rules.String:
		return decls.String
	case rules.Int:
		return decls.Int
	case rules.Float:
		return decls.Double
	case rules.Bool:
		return decls.Bool
	case rules.Duration:
		return decls.Duration
	case rules.Timestamp:
		return decls.Timestamp
	case rules.Map:
		return decls.NewMapType(celType(v.KeyType), celType(v.ValueType))
	case rules.List:
		return decls.NewListType(celType(v.ValueType))
	}
	return decls.Any
}

// schemaToDeclarations converts from a rules/Schema to a set of CEL declarations that
// are passed to the CEL engine
func schemaToDeclarations(s rules.Schema) (cel.EnvOption, error) {
	items := []*exprbp.Decl{}

	for _, d := range s.Elements {
		items = append(items, decls.NewIdent(d.Name, celType(d.Type), nil))
	}
	return cel.Declarations(items...), nil
}

type CELEngine struct {
	// Map of ruleSet.ID to rule set
	ruleSets map[string]rules.RuleSet
	// Map of ruleset.ID and rule.ID to program
	programs map[string]map[string]cel.Program
}

func NewEngine() *CELEngine {
	engine := CELEngine{}
	engine.ruleSets = make(map[string]rules.RuleSet)
	engine.programs = make(map[string]map[string]cel.Program)
	return &engine
}

func (e *CELEngine) RuleSet(id string) (rules.RuleSet, bool) {
	ruleSet, found := e.ruleSets[id]
	return ruleSet, found
}

func (e *CELEngine) EvaluateAll(data map[string]interface{}, ruleSetID string) ([]rules.Result, error) {
	return rules.EvaluateAll(e, data, ruleSetID)
}

func (e *CELEngine) EvaluateUntilTrue(data map[string]interface{}, ruleSetID string) (rules.Result, error) {
	return rules.EvaluateUntilTrue(e, data, ruleSetID)
}

func (e *CELEngine) EvaluateRule(data map[string]interface{}, ruleSetID string, ruleID string) (*rules.Result, error) {

	program, found := e.programs[ruleSetID][ruleID]

	if program == nil || !found {
		return nil, fmt.Errorf("Missing program for rule")
	}

	result := rules.Result{RuleSetID: ruleSetID, RuleID: ruleID}

	rawValue, _, error := program.Eval(data)
	if error == nil {
		switch v := rawValue.Value().(type) {
		case bool:
			result.Pass = v
		case float64:
			result.Float64Value = v
		case int64:
			result.Int64Value = v
		case string:
			result.StringValue = v
		}
	}
	return &result, nil
}

func (e *CELEngine) AddRuleSet(rs rules.RuleSet) error {

	// Prevent accidental updating of rule sets
	if _, found := e.ruleSets[rs.ID]; found {
		return fmt.Errorf("Rule set already exists: %s", rs.ID)
	}
	return e.AddOrReplaceRuleSet(rs)
}

func (e *CELEngine) AddOrReplaceRuleSet(rs rules.RuleSet) error {

	decls, err := schemaToDeclarations(rs.Schema)
	if err != nil {
		return err
	}

	env, err := cel.NewEnv(decls)
	if err != nil {
		return err
	}
	e.programs[rs.ID] = make(map[string]cel.Program)

	for k, r := range rs.Rules {

		// Parse the rule expression to an AST
		p, iss := env.Parse(r.Expression())
		if iss != nil && iss.Err() != nil {
			return iss.Err()
		}

		// Type-check the parsed AST against the declarations
		c, iss := env.Check(p)
		if iss != nil && iss.Err() != nil {
			return iss.Err()
		}

		// Generate an evaluable program
		prg, err := env.Program(c)
		if err != nil {
			return err
		}

		// Save the program ready to be evaluated
		e.programs[rs.ID][k] = prg
	}

	e.ruleSets[rs.ID] = rs
	return nil
}
