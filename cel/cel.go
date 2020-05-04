// package cel provides an implementation of the rules/Engine interface backed by Google's cel-go rules engine
package cel

import (
	"fmt"

	"github.com/ezachrisen/rules"

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
	rules    map[string]rules.Rule
	programs map[string]cel.Program
}

func NewEngine() *CELEngine {
	engine := CELEngine{}
	engine.rules = make(map[string]rules.Rule)
	engine.programs = make(map[string]cel.Program)
	return &engine
}

func (e *CELEngine) Set(id string) (rules.Rule, bool) {
	r, found := e.rules[id]
	return r, found

}

func (e *CELEngine) Evaluate(data map[string]interface{}, id string) (*rules.Result, error) {
	return e.EvaluateN(data, id, rules.MaxLevels)
}

func (e *CELEngine) EvaluateN(data map[string]interface{}, id string, n int) (*rules.Result, error) {
	if n < 0 {
		return nil, nil
	}

	rule, found := e.rules[id]
	if rule == nil || !found {
		return nil, fmt.Errorf("rule not found %s", id)
	}

	parent_rule_result := rules.Result{RuleID: id, Rule: &rule}

	program, found := e.programs[id]
	if program != nil && found {
		rawValue, _, error := program.Eval(data)
		if error != nil {
			return nil, fmt.Errorf("Error evaluating rule %s:%w", id, error)
		}

		parent_rule_result.RawValue = rawValue
		parent_rule_result.Value = rawValue.Value()
	} else {
		parent_rule_result.RawValue = true
		parent_rule_result.Value = true
	}

	for _, c := range e.rules[id].Rules() {
		res, err := e.EvaluateN(data, c.ID(), n-1)
		if err != nil {
			return nil, err
		}
		if res != nil {
			parent_rule_result.Results = append(parent_rule_result.Results, *res)
		}
	}

	return &parent_rule_result, nil
}

func (e *CELEngine) CompileRule(env *cel.Env, r rules.Rule) (cel.Program, error) {

	// Parse the rule expression to an AST
	p, iss := env.Parse(r.Expression())
	if iss != nil && iss.Err() != nil {
		return nil, fmt.Errorf("parsing rule %s, %w", r.ID(), iss.Err())
	}

	// Type-check the parsed AST against the declarations
	c, iss := env.Check(p)
	if iss != nil && iss.Err() != nil {
		return nil, fmt.Errorf("checking rule %s, %w", r.ID(), iss.Err())
	}

	// Generate an evaluable program
	prg, err := env.Program(c)
	if err != nil {
		return nil, fmt.Errorf("generating program %s, %w", r.ID(), err)
	}
	return prg, nil
}

func (e *CELEngine) AddRule(r rules.Rule) error {

	e.rules[r.ID()] = r
	decls, err := schemaToDeclarations(r.Schema())
	if err != nil {
		return err
	}

	env, err := cel.NewEnv(decls)
	if err != nil {
		return err
	}

	if r.Expression() != "" {
		prg, err := e.CompileRule(env, r)
		if err != nil {
			return fmt.Errorf("compiling rule %s: %w", r.ID(), err)
		}
		e.programs[r.ID()] = prg
	}

	for _, c := range r.Rules() {
		err = e.AddRule(c)
		if err != nil {
			return fmt.Errorf("adding rule %s: %w", c.ID(), err)
		}
	}
	return nil
}

// func (e *CELEngine) EvaluateAll(data map[string]interface{}, ruleSetID string) ([]rules.Result, error) {
// 	return rules.EvaluateAll(e, data, ruleSetID)
// }

// func (e *CELEngine) EvaluateUntilTrue(data map[string]interface{}, ruleSetID string) (rules.Result, error) {
// 	return rules.EvaluateUntilTrue(e, data, ruleSetID)
// }

// switch v := rawValue.Value().(type) {
// case bool:
// 	parent_rule_result.Pass = v
// case float64:
// 	parent_rule_result.Float64Value = v
// case int64:
// 	parent_rule_result.Int64Value = v
// case string:
// 	parent_rule_result.StringValue = v
// case *duration.Duration:
// 	// Durations are returned as seconds; convert to nanosecond for
// 	parent_rule_result.Duration = time.Duration(v.Seconds * 1000000000)
