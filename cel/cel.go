// package cel provides an implementation of the rules/Engine interface backed by Google's cel-go rules engine
package cel

import (
	"fmt"
	"time"

	"github.com/ezachrisen/rules"
	"github.com/golang/protobuf/ptypes/duration"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types/pb"
	exprbp "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/protobuf/runtime/protoiface"
)

// celType converts from a rules.Type to a CEL type
func celType(t rules.Type) (*exprbp.Type, error) {

	switch v := t.(type) {
	case rules.String:
		return decls.String, nil
	case rules.Int:
		return decls.Int, nil
	case rules.Float:
		return decls.Double, nil
	case rules.Bool:
		return decls.Bool, nil
	case rules.Duration:
		return decls.Duration, nil
	case rules.Timestamp:
		return decls.Timestamp, nil
	case rules.Map:
		key, err := celType(v.KeyType)
		if err != nil {
			return nil, fmt.Errorf("Setting key of %v map: %w", v.KeyType, err)
		}
		val, err := celType(v.ValueType)
		if err != nil {
			return nil, fmt.Errorf("Setting value of %v map: %w", v.ValueType, err)
		}
		return decls.NewMapType(key, val), nil
	case rules.List:
		val, err := celType(v.ValueType)
		if err != nil {
			return nil, fmt.Errorf("Setting value of %v list: %w", v.ValueType, err)
		}
		return decls.NewListType(val), nil
	case rules.Proto:
		protoMessage, ok := v.Message.(protoiface.MessageV1)
		if !ok {
			return nil, fmt.Errorf("Casting to proto message %v", v.Protoname)
		}
		_, err := pb.DefaultDb.RegisterMessage(protoMessage)
		if err != nil {
			return nil, fmt.Errorf("registering proto message %v: %w", v.Protoname, err)
		}
		return decls.NewObjectType(v.Protoname), nil
	}
	return decls.Any, nil

}

// schemaToDeclarations converts from a rules/Schema to a set of CEL declarations that
// are passed to the CEL engine
func schemaToDeclarations(s rules.Schema) (cel.EnvOption, error) {
	items := []*exprbp.Decl{}

	for _, d := range s.Elements {
		typ, err := celType(d.Type)
		if err != nil {
			return nil, err
		}
		items = append(items, decls.NewIdent(d.Name, typ, nil))
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
	if error != nil {
		return nil, fmt.Errorf("Error evaluating rule %s:%s:%w", ruleSetID, ruleID, error)
	}

	result.RawValue = rawValue

	switch v := rawValue.Value().(type) {
	case bool:
		result.Pass = v
	case float64:
		result.Float64Value = v
	case int64:
		result.Int64Value = v
	case string:
		result.StringValue = v
	case *duration.Duration:
		// Durations are returned as seconds; convert to nanosecond for
		// TODO: check how CEL supports sub-second durations
		result.Duration = time.Duration(v.Seconds * 1000000000)
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
			return fmt.Errorf("parsing rule %s:%s, %w", rs.ID, k, iss.Err())
		}

		// Type-check the parsed AST against the declarations
		c, iss := env.Check(p)
		if iss != nil && iss.Err() != nil {
			return fmt.Errorf("checking rule %s:%s, %w", rs.ID, k, iss.Err())
		}

		// Generate an evaluable program
		prg, err := env.Program(c)
		if err != nil {
			return fmt.Errorf("generating program %s:%s, %w", rs.ID, k, err)
		}

		// Save the program ready to be evaluated
		e.programs[rs.ID][k] = prg
	}

	e.ruleSets[rs.ID] = rs
	return nil
}
