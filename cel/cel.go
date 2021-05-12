package cel

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ezachrisen/indigo"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// CELEvaluator implements the indigo.Evaluator interface.
// It uses the CEL-Go package to compile and evaluate rules.
type CELEvaluator struct {
	programs map[*indigo.Rule]CELProgram
}

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
	e := CELEvaluator{
		programs: map[*indigo.Rule]CELProgram{},
	}
	return &e
}

// Compile checks a rule, prepares a compiled CELProgram, and stores the program
// in rule.Program. CELProgram contains the compiled program used to evaluate the rules,
// and if we're collecting diagnostics, CELProgram also contains the CEL AST to provide
// type and symbol information in diagnostics.
//
// Any errors in compilation are returned, and the rule.Program is set to nil.
// If dryRun is true, this does nothing.
func (e *CELEvaluator) Compile(expr string, s indigo.Schema, resultType indigo.Type, collectDiagnostics bool, dryRun bool) (interface{}, error) {
	prog := CELProgram{}

	if expr == "" {
		return nil, nil
	}

	var opts []cel.EnvOption
	var err error

	// Convert from an Indigo schema to a set of CEL options
	opts, err = schemaToDeclarations(s)
	if err != nil {
		return nil, err
	}

	if opts == nil || len(opts) == 0 {
		return nil, fmt.Errorf("no valid schema")
	}

	env, err := cel.NewEnv(opts...)
	if err != nil {
		return nil, err
	}

	// Parse the rule expression to an AST
	ast, iss := env.Parse(expr)
	if iss != nil && iss.Err() != nil {
		// Remove some wonky formatting from CEL's error message.
		return nil, fmt.Errorf("parsing rule:\n%s", strings.ReplaceAll(fmt.Sprintf("%s", iss.Err()), "<input>:", ""))
	}

	// Type-check the parsed AST against the declarations
	c, iss := env.Check(ast)
	if iss != nil && iss.Err() != nil {
		return nil, fmt.Errorf("checking rule:\n%w", iss.Err())
	}

	if resultType != nil {
		err := doTypesMatch(c.ResultType(), resultType)
		if err != nil {
			return nil, fmt.Errorf("compiling rule: %w", err)
		}
	}

	options := cel.EvalOptions()
	if collectDiagnostics {
		options = cel.EvalOptions(cel.OptTrackState)
	}

	prog.program, err = env.Program(c, options)
	if err != nil {
		return nil, fmt.Errorf("generating program: %w", err)
	}

	if !dryRun {
		if collectDiagnostics {
			prog.ast = ast
		}
	}

	return prog, nil
}

// Evaluate a rule against the input data.
// Called by indigo.Engine.Evaluate for the rule and its children.
func (e *CELEvaluator) Evaluate(data map[string]interface{}, expr string, s indigo.Schema, self interface{}, evalData interface{}, returnDiagnostics bool) (indigo.Value, string, error) {

	program, ok := evalData.(CELProgram)

	// If the rule doesn't have a program, return a default result
	if !ok {
		return indigo.Value{
			Val:  true,
			Type: indigo.Bool{},
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
		return indigo.Value{}, diagnostics, fmt.Errorf("evaluating rule: %w", err)
	}

	switch v := rawValue.Value().(type) {
	case bool:
		return indigo.Value{
			Val:  v,
			Type: indigo.Bool{},
		}, diagnostics, nil
	default:
		return indigo.Value{
			Val:  v,
			Type: indigo.Any{},
		}, diagnostics, nil
	}

}

// --------------------------------------------------------------------------- COMPILATION

// Parse, check and store the rule

func doTypesMatch(result *expr.Type, expected indigo.Type) error {

	// Convert the CEL result type to an Indigo type
	indigoResultType, err := indigoType(result)
	if err != nil {
		return err
	}

	switch e := expected.(type) {
	case indigo.Proto:
		resultAsProto, ok := indigoResultType.(indigo.Proto)
		if !ok {
			return fmt.Errorf("Rule.ResultValue is a proto message. Result from rule compilation is %T", result)
		}
		if resultAsProto.Protoname != e.Protoname {
			return fmt.Errorf("Rule.ResultValue is a proto message with type %s, the result from compilation is a proto message with type %s", e.Protoname, resultAsProto.Protoname)
		}
	case indigo.Bool:
		_, ok := indigoResultType.(indigo.Bool)
		if !ok {
			return fmt.Errorf("Rule.ResultValue is a boolean. Result from rule compilation is %T", result)
		}
	case indigo.Float:
		_, ok := indigoResultType.(indigo.Float)
		if !ok {
			return fmt.Errorf("Rule.ResultValue is a float. Result from rule compilation is %T", result)
		}
	}

	return nil
}

// indigoType convertes from a CEL type to an indigo.Type
// TODO: cover more types
func indigoType(t *expr.Type) (indigo.Type, error) {

	switch v := t.TypeKind.(type) {
	case *expr.Type_MessageType:
		return indigo.Proto{Protoname: t.GetMessageType()}, nil
	case *expr.Type_Primitive:
		switch t.GetPrimitive() {
		case expr.Type_BOOL:
			return indigo.Bool{}, nil
		case expr.Type_DOUBLE:
			return indigo.Float{}, nil
		default:
			return nil, fmt.Errorf("Unexpected primitive type %v", v)
		}
	default:
		return nil, fmt.Errorf("Unexpected type %v", v)
	}
}

// celType converts from a indigo.Type to a CEL type
func celType(t indigo.Type) (*expr.Type, error) {

	switch v := t.(type) {
	case indigo.String:
		return decls.String, nil
	case indigo.Int:
		return decls.Int, nil
	case indigo.Float:
		return decls.Double, nil
	case indigo.Bool:
		return decls.Bool, nil
	case indigo.Duration:
		return decls.Duration, nil
	case indigo.Timestamp:
		return decls.Timestamp, nil
	case indigo.Map:
		key, err := celType(v.KeyType)
		if err != nil {
			return nil, fmt.Errorf("Setting key of %v map: %w", v.KeyType, err)
		}
		val, err := celType(v.ValueType)
		if err != nil {
			return nil, fmt.Errorf("Setting value of %v map: %w", v.ValueType, err)
		}
		return decls.NewMapType(key, val), nil
	case indigo.List:
		val, err := celType(v.ValueType)
		if err != nil {
			return nil, fmt.Errorf("Setting value of %v list: %w", v.ValueType, err)
		}
		return decls.NewListType(val), nil
	case indigo.Proto:
		return decls.NewObjectType(v.Protoname), nil
	}
	return decls.Any, nil

}

// schemaToDeclarations converts from a rules/Schema to a set of CEL declarations that
// are passed to the CEL engine
func schemaToDeclarations(s indigo.Schema) ([]cel.EnvOption, error) {
	declarations := []*expr.Decl{}
	types := []interface{}{}

	for _, d := range s.Elements {
		typ, err := celType(d.Type)
		if err != nil {
			return nil, err
		}
		declarations = append(declarations, decls.NewVar(d.Name, typ))

		switch v := d.Type.(type) {
		case indigo.Proto:
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
