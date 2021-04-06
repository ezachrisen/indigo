// Package cel provides an implementation of the indigo.Evaluator interface backed by Google's cel-go rules engine.
//
// See https://github.com/google/cel-go and https://opensource.google/projects/cel for more information
// about CEL.
//
// The rule expressions you write must conform to the CEL spec: https://github.com/google/cel-spec.
//
package cel

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/schema"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// CELEvaluator implements the indigo.Evaluator interface.
// It uses the CEL-Go package to compile and evaluate rules.
type CELEvaluator struct{}

// CELProgram holds a compiled CEL Program and
// (potentially) an AST. The AST is used if we're collecting diagnostics
// for the engine.
type CELProgram struct {
	program cel.Program
	ast     *cel.Ast
}

// Initialize a new CEL Evaluator
// The evaluator contains internal data used to facilitate CEL expression evaluation.
func NewEvaluator() *CELEvaluator {
	e := CELEvaluator{}
	return &e
}

// Compile checks a rule, prepares a compiled CELProgram, and stores the program
// in rule.Program. CELProgram contains the compiled program used to evaluate the rules,
// and if we're collecting diagnostics, CELProgram also contains the CEL AST to provide
// type and symbol information in diagnostics.
//
// Any errors in compilation are returned, and the rule.Program is set to nil.
// If dryRun is true, this does nothing.
func (e *CELEvaluator) Compile(r *indigo.Rule, collectDiagnostics bool, dryRun bool) error {
	prog := CELProgram{}
	r.Program = nil // ensure the previous program is cleared

	if r.Expr == "" {
		return nil
	}

	var opts []cel.EnvOption
	var err error

	// Convert from an Indigo schema to a set of CEL options
	opts, err = schemaToDeclarations(r.Schema)
	if err != nil {
		return err
	}

	if opts == nil || len(opts) == 0 {
		return fmt.Errorf("No valid schema for rule %s", r.ID)
	}

	env, err := cel.NewEnv(opts...)
	if err != nil {
		return err
	}

	// Parse the rule expression to an AST
	ast, iss := env.Parse(r.Expr)
	if iss != nil && iss.Err() != nil {
		// Remove some wonky formatting from CEL's error message.
		return fmt.Errorf("parsing rule %s:\n%s", r.ID, strings.ReplaceAll(fmt.Sprintf("%s", iss.Err()), "<input>:", ""))
	}

	// Type-check the parsed AST against the declarations
	c, iss := env.Check(ast)
	if iss != nil && iss.Err() != nil {
		return fmt.Errorf("checking rule %s:\n%w", r.ID, iss.Err())
	}

	if r.ResultType != nil {
		err := doTypesMatch(c.ResultType(), r.ResultType)
		if err != nil {
			return fmt.Errorf("Error compiling rule: %w", err)
		}
	}

	options := cel.EvalOptions()
	if collectDiagnostics {
		options = cel.EvalOptions(cel.OptTrackState)
	}

	prog.program, err = env.Program(c, options)
	if err != nil {
		return fmt.Errorf("generating program %s, %w", r.ID, err)
	}

	if !dryRun {
		if collectDiagnostics {
			prog.ast = ast
		}
		r.Program = prog
	}

	return nil
}

// Evaluate a rule against the input data.
// Called by indigo.Engine.Evaluate for the rule and its children.
func (e *CELEvaluator) Eval(data map[string]interface{}, r *indigo.Rule, returnDiagnostics bool) (indigo.Value, string, error) {

	program, ok := r.Program.(CELProgram)

	// If the rule doesn't have a program, return a default result
	if !ok {
		return indigo.Value{
			Val: true,
			Typ: schema.Bool{},
		}, "", nil
	}

	rawValue, details, err := program.program.Eval(data)
	// Do not check the error yet. Grab the diagnostics first
	// TODO: Return diagnostics with errors

	// TODO: Check return type
	// Determine if the value produced matched the rule's expectations
	// if r.ResulType != nil {
	// 	err := doTypesMatch(result *expr.Type, r.ResultType)
	// }

	var diagnostics string
	if returnDiagnostics {
		diagnostics = collectDiagnostics(program.ast, details, data)
	}

	if err != nil {
		return indigo.Value{}, diagnostics, fmt.Errorf("Error evaluating rule %s:%w", r.ID, err)
	}

	switch v := rawValue.Value().(type) {
	case bool:
		return indigo.Value{
			Val: v,
			Typ: schema.Bool{},
		}, diagnostics, nil
	default:
		return indigo.Value{
			Val: v,
			Typ: schema.Any{},
		}, diagnostics, nil
	}

}

// --------------------------------------------------------------------------- COMPILATION

// Parse, check and store the rule

func doTypesMatch(result *expr.Type, expected schema.Type) error {

	// Convert the CEL result type to an Indigo type
	indigoResultType, err := indigoType(result)
	if err != nil {
		return err
	}

	switch e := expected.(type) {
	case schema.Proto:
		resultAsProto, ok := indigoResultType.(schema.Proto)
		if !ok {
			return fmt.Errorf("Rule.ResultValue is a proto message. Result from rule compilation is %T", result)
		}
		if resultAsProto.Protoname != e.Protoname {
			return fmt.Errorf("Rule.ResultValue is a proto message with type %s, the result from compilation is a proto message with type %s", e.Protoname, resultAsProto.Protoname)
		}
	case schema.Bool:
		_, ok := indigoResultType.(schema.Bool)
		if !ok {
			return fmt.Errorf("Rule.ResultValue is a boolean. Result from rule compilation is %T", result)
		}
	case schema.Float:
		_, ok := indigoResultType.(schema.Float)
		if !ok {
			return fmt.Errorf("Rule.ResultValue is a float. Result from rule compilation is %T", result)
		}
	}

	return nil
}

// indigoType convertes from a CEL type to an schema.Type
// TODO: cover more types
func indigoType(t *expr.Type) (schema.Type, error) {

	switch v := t.TypeKind.(type) {
	case *expr.Type_MessageType:
		return schema.Proto{Protoname: t.GetMessageType()}, nil
	case *expr.Type_Primitive:
		switch t.GetPrimitive() {
		case expr.Type_BOOL:
			return schema.Bool{}, nil
		case expr.Type_DOUBLE:
			return schema.Float{}, nil
		default:
			return nil, fmt.Errorf("Unexpected primitive type %v", v)
		}
	default:
		return nil, fmt.Errorf("Unexpected type %v", v)
	}
}

// celType converts from a schema.Type to a CEL type
func celType(t schema.Type) (*expr.Type, error) {

	switch v := t.(type) {
	case schema.String:
		return decls.String, nil
	case schema.Int:
		return decls.Int, nil
	case schema.Float:
		return decls.Double, nil
	case schema.Bool:
		return decls.Bool, nil
	case schema.Duration:
		return decls.Duration, nil
	case schema.Timestamp:
		return decls.Timestamp, nil
	case schema.Map:
		key, err := celType(v.KeyType)
		if err != nil {
			return nil, fmt.Errorf("Setting key of %v map: %w", v.KeyType, err)
		}
		val, err := celType(v.ValueType)
		if err != nil {
			return nil, fmt.Errorf("Setting value of %v map: %w", v.ValueType, err)
		}
		return decls.NewMapType(key, val), nil
	case schema.List:
		val, err := celType(v.ValueType)
		if err != nil {
			return nil, fmt.Errorf("Setting value of %v list: %w", v.ValueType, err)
		}
		return decls.NewListType(val), nil
	case schema.Proto:
		return decls.NewObjectType(v.Protoname), nil
	}
	return decls.Any, nil

}

// schemaToDeclarations converts from a rules/Schema to a set of CEL declarations that
// are passed to the CEL engine
func schemaToDeclarations(s schema.Schema) ([]cel.EnvOption, error) {
	declarations := []*expr.Decl{}
	types := []interface{}{}

	for _, d := range s.Elements {
		typ, err := celType(d.Type)
		if err != nil {
			return nil, err
		}
		declarations = append(declarations, decls.NewVar(d.Name, typ))

		switch v := d.Type.(type) {
		case schema.Proto:
			types = append(types, v.Message)
			//fmt.Printf("Added a new type: %T with name %s\n", v.Message, v.Protoname)
		}

	}
	opts := []cel.EnvOption{}
	opts = append(opts, cel.Declarations(declarations...))
	opts = append(opts, cel.Types(types...))
	return opts, nil
}

// --------------------------------------------------------------------------- DIAGNOSTICS

func printAST(ex *expr.Expr, n int, details *cel.EvalDetails, data map[string]interface{}) string {
	s := strings.Builder{}

	indent := strings.Repeat(" ", n*2)

	value := "?"
	valueSource := ""
	evaluatedValue, ok := details.State().Value(ex.Id)

	if ok {
		switch v := evaluatedValue.(type) {
		case types.Duration:
			dur := time.Duration(v.Seconds * int64(math.Pow10(9)))
			value = fmt.Sprintf("%60s", dur)
		case types.Timestamp:
			value = fmt.Sprintf("%60s", time.Unix(v.Seconds, 0))
		default:
			value = fmt.Sprintf("%60s", fmt.Sprintf("%v", evaluatedValue))
		}
		valueSource = "E"
	} else {
		value = fmt.Sprintf("%60s (%v)", "?", ex.Id)
	}

	switch i := ex.GetExprKind().(type) {
	case *expr.Expr_CallExpr:
		s.WriteString(fmt.Sprintf("%s %s %s %s\n", value, valueSource, indent, strings.Trim(i.CallExpr.GetFunction(), "_")))
		for x, _ := range i.CallExpr.Args {
			s.WriteString(printAST(i.CallExpr.Args[x], n+1, details, data))
		}
	case *expr.Expr_ComprehensionExpr:
		operandName := i.ComprehensionExpr.IterRange.GetSelectExpr().Operand.GetIdentExpr().GetName()
		fieldName := i.ComprehensionExpr.IterRange.GetSelectExpr().Field
		comprehensionName := i.ComprehensionExpr.LoopCondition.GetCallExpr().Function
		callExpression := getCallExpression(i.ComprehensionExpr.GetLoopStep().GetCallExpr())
		if comprehensionName == "@not_strictly_false" {
			comprehensionName = "all"
		}
		s.WriteString(fmt.Sprintf("%s %s %s %s.%s.%s %s\n", value, valueSource, indent, operandName, fieldName, comprehensionName, callExpression))
	case *expr.Expr_ConstExpr:
		s.WriteString(fmt.Sprintf("%s %s %s %s\n", value, valueSource, indent, strings.Trim(i.ConstExpr.String(), " ")))
	case *expr.Expr_SelectExpr:
		operandName := getSelectIdent(i)
		fieldName := i.SelectExpr.Field

		dottedName := operandName + "." + fieldName
		inputValue, ok := data[dottedName]
		if ok {
			value = fmt.Sprintf("%60s", fmt.Sprintf("%v", inputValue))
			valueSource = "I"
		} else {
			obj, ok := data[operandName]
			if ok {
				//x := reflect.ValueOf(obj).Elem()
				value = fmt.Sprintf("%60s", fmt.Sprintf("%v", obj)) //fmt.Sprintf("%v", x.FieldByName(fieldName)))
				valueSource = "I"
			}
		}

		s.WriteString(fmt.Sprintf("%s %s %s %s\n", value, valueSource, indent, operandName+"."+fieldName))
	case *expr.Expr_IdentExpr:
		s.WriteString(fmt.Sprintf("%s %s %s %s\n", value, valueSource, indent, i.IdentExpr.Name))
	default:
		s.WriteString(fmt.Sprintf("%s %s %s %s\n", value, valueSource, indent, "** Unknown"))
	}
	return s.String()
}

func getCallExpression(e *expr.Expr_Call) string {

	x := ""

	if e.Function != "_&&_" {
		x = e.Function
	}

	for _, a := range e.Args {
		switch aa := a.GetExprKind().(type) {
		case *expr.Expr_IdentExpr:
			if aa.IdentExpr.Name != "__result__" {
				x = x + " " + aa.IdentExpr.Name
			}
		case *expr.Expr_CallExpr:
			x = x + "(" + getCallExpression(a.GetCallExpr()) + ")"
		case *expr.Expr_ConstExpr:
			x = x + " " + aa.ConstExpr.String()
		}
	}
	return x
}

func getSelectIdent(i *expr.Expr_SelectExpr) string {

	switch v := i.SelectExpr.Operand.GetExprKind().(type) {
	case *expr.Expr_SelectExpr:
		return getSelectIdent(v) + "." + v.SelectExpr.Field
	case *expr.Expr_IdentExpr:
		return v.IdentExpr.Name
	}

	return ""
}

func collectDiagnostics(ast *cel.Ast, details *cel.EvalDetails, data map[string]interface{}) string {

	if ast == nil || details == nil {
		fmt.Println(ast, details)
		return ""

	}

	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("----------------------------------------------------------------------------------------------------\n"))
	//	s.WriteString(fmt.Sprintf("Rule ID: %s\n", r.RuleID))
	s.WriteString(fmt.Sprintf("Expression:\n"))
	s.WriteString(fmt.Sprintf("%s\n\n", word_wrap(ast.Source().Content(), 100)))
	// s.WriteString(fmt.Sprintf("Evaluation Result   :  %t\n", r.Pass))
	// s.WriteString(fmt.Sprintf("Evaluation Raw Value:  %v\n", r.Value))
	// s.WriteString(fmt.Sprintf("Rule Expression:\n"))
	// s.WriteString(fmt.Sprintf("%s\n", word_wrap(ast.Source().Content(), 100)))
	s.WriteString(fmt.Sprintf("----------------------------------------------------------------------------------------------------\n"))
	s.WriteString(fmt.Sprintf("                                          EVALUATION TREE\n"))
	s.WriteString(fmt.Sprintf("----------------------------------------------------------------------------------------------------\n"))
	s.WriteString(fmt.Sprintf("%60s    %-30s\n", "VALUE", "EXPRESSION"))
	s.WriteString(fmt.Sprintf("----------------------------------------------------------------------------------------------------\n"))
	s.WriteString(printAST(ast.Expr(), 0, details, data))
	return s.String()
}

func word_wrap(text string, lineWidth int) string {
	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return text
	}
	wrapped := words[0]
	spaceLeft := lineWidth - len(wrapped)
	for _, word := range words[1:] {
		if len(word)+1 > spaceLeft {
			wrapped += "\n" + word
			spaceLeft = lineWidth - len(word)
		} else {
			wrapped += " " + word
			spaceLeft -= 1 + len(word)
		}
	}

	return wrapped

}
