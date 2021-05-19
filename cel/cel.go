package cel

import (
	"fmt"
	"math"
	"reflect" // required by CEL to construct a proto from an expression
	"strings"
	"time"

	"github.com/ezachrisen/indigo"

	celgo "github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	ctypes "github.com/google/cel-go/common/types"
	gexpr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Evaluator implements the indigo.Evaluator interface.
// It uses the CEL-Go package to compile and evaluate rules.
type Evaluator struct{}

// celProgram holds a compiled CEL Program and
// (potentially) an AST. The AST is used if we're collecting diagnostics
// for the engine.
type celProgram struct {
	program celgo.Program
	ast     *celgo.Ast
}

// NewEvaluator creates a new CEL Evaluator.
// The evaluator contains internal data used to facilitate CEL expression evaluation.
func NewEvaluator() *Evaluator {
	e := Evaluator{}
	return &e
}

// Compile checks a rule, prepares a compiled CELProgram, and stores the program
// in rule.Program. CELProgram contains the compiled program used to evaluate the rules,
// and if we're collecting diagnostics, CELProgram also contains the CEL AST to provide
// type and symbol information in diagnostics.
//
// Any errors in compilation are returned, and the rule.Program is set to nil.
// If dryRun is true, this does nothing.
func (*Evaluator) Compile(expr string, s indigo.Schema, resultType indigo.Type,
	collectDiagnostics bool, dryRun bool) (interface{}, error) {
	prog := celProgram{}

	if expr == "" {
		return nil, nil
	}

	var opts []celgo.EnvOption
	var err error

	// Convert from an Indigo schema to a set of CEL options
	opts, err = schemaToDeclarations(s)
	if err != nil {
		return nil, err
	}

	if len(opts) == 0 {
		return nil, fmt.Errorf("no valid schema")
	}

	env, err := celgo.NewEnv(opts...)
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

	options := celgo.EvalOptions()
	if collectDiagnostics {
		options = celgo.EvalOptions(celgo.OptTrackState)
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
func (*Evaluator) Evaluate(data map[string]interface{}, _ string,
	_ indigo.Schema, _ interface{}, evalData interface{}, resultValue indigo.Type, returnDiagnostics bool) (indigo.Value, string, error) {

	program, ok := evalData.(celProgram)

	// If the rule doesn't have a program, return a default result
	if !ok {
		return indigo.Value{
			Val:  true,
			Type: indigo.Bool{},
		}, "", nil
	}

	rawValue, details, err := program.program.Eval(data)
	// Do not check the error yet. Grab the diagnostics first

	// TODO: Check return type
	// Determine if the value produced matched the rule's expectations
	// if r.ResulType != nil {
	// 	err := doTypesMatch(result *gexpr.Type, r.ResultType)
	// }

	var diagnostics string
	if returnDiagnostics {
		diagnostics = collectDiagnostics(program.ast, details, data)
	}

	if err != nil {
		return indigo.Value{}, diagnostics, fmt.Errorf("evaluating rule: %w", err)
	}

	var iv indigo.Value

	switch v := rawValue.Value().(type) {
	case bool:
		iv.Val = v
		iv.Type = indigo.Bool{}
	case *dynamicpb.Message:
		resultProto, ok := resultValue.(indigo.Proto)
		if !ok {
			return indigo.Value{}, diagnostics, fmt.Errorf("expected type %T, got proto of type %v", resultValue, v.ProtoReflect().Descriptor().FullName())
		}
		pb, err := rawValue.ConvertToNative(reflect.TypeOf(resultProto.Message))
		if err != nil {
			return indigo.Value{}, diagnostics, fmt.Errorf("conversion to %T failed: %v", resultProto, err)
		}
		iv.Val = pb
		iv.Type = indigo.Proto{
			Protoname: string(v.ProtoReflect().Descriptor().FullName()),
			Message:   pb,
		}
	default:
		iv.Val = v
		iv.Type = indigo.Any{}
	}

	return iv, diagnostics, nil

}

// --------------------------------------------------------------------------- TYPE CONVERSIONS
//

// doTypesMatch determines if the indigo and cel types match, meaning
// they can be converted from one to the other
func doTypesMatch(cel *gexpr.Type, indigo indigo.Type) error {

	celConverted, err := indigoType(cel)
	if err != nil {
		return err
	}

	if celConverted.String() != indigo.String() {
		return fmt.Errorf("types do no match: %T (%v), %T (%v)", celConverted, celConverted, indigo, indigo)
	}

	return nil
}

// indigoType convertes from a CEL type to an indigo.Type
func indigoType(t *gexpr.Type) (indigo.Type, error) {

	switch v := t.TypeKind.(type) {
	case *gexpr.Type_MessageType:
		return indigo.Proto{Protoname: t.GetMessageType()}, nil
	case *gexpr.Type_WellKnown:
		switch v.WellKnown {
		case gexpr.Type_DURATION:
			return indigo.Duration{}, nil
		case gexpr.Type_TIMESTAMP:
			return indigo.Timestamp{}, nil
		default:
			return nil, fmt.Errorf("unknown 'wellknow' type: %T", v)
		}
	case *gexpr.Type_MapType_:
		kType, err := indigoType(v.MapType.KeyType)
		if err != nil {
			return nil, err
		}

		vType, err := indigoType(v.MapType.ValueType)
		if err != nil {
			return nil, err
		}

		return indigo.Map{
			KeyType:   kType,
			ValueType: vType,
		}, nil
	case *gexpr.Type_ListType_:
		vType, err := indigoType(v.ListType.ElemType)
		if err != nil {
			return nil, err
		}
		return indigo.List{
			ValueType: vType,
		}, nil
	case *gexpr.Type_Dyn:
		return indigo.Any{}, nil
	case *gexpr.Type_Primitive:
		switch t.GetPrimitive() {
		case gexpr.Type_BOOL:
			return indigo.Bool{}, nil
		case gexpr.Type_DOUBLE:
			return indigo.Float{}, nil
		case gexpr.Type_STRING:
			return indigo.String{}, nil
		case gexpr.Type_INT64:
			return indigo.Int{}, nil
		default:
			return nil, fmt.Errorf("unexpected primitive type %v", v)
		}
	default:
		return nil, fmt.Errorf("unexpected type %v", v)
	}
}

// celType converts from an indigo type to a CEL type
func celType(t indigo.Type) (*gexpr.Type, error) {

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
			return nil, fmt.Errorf("setting key of %v map: %w", v.KeyType, err)
		}
		val, err := celType(v.ValueType)
		if err != nil {
			return nil, fmt.Errorf("setting value of %v map: %w", v.ValueType, err)
		}
		return decls.NewMapType(key, val), nil
	case indigo.List:
		val, err := celType(v.ValueType)
		if err != nil {
			return nil, fmt.Errorf("setting value of %v list: %w", v.ValueType, err)
		}
		return decls.NewListType(val), nil
	case indigo.Proto:
		return decls.NewObjectType(v.Protoname), nil
	default:
		return nil, fmt.Errorf("unknown indigo type %s", t)
	}

}

// schemaToDeclarations converts from a rules/Schema to a set of CEL declarations that
// are passed to the CEL engine
func schemaToDeclarations(s indigo.Schema) ([]celgo.EnvOption, error) {
	declarations := []*gexpr.Decl{}
	types := []interface{}{}

	for _, d := range s.Elements {
		typ, err := celType(d.Type)
		if err != nil {
			return nil, err
		}
		declarations = append(declarations, decls.NewVar(d.Name, typ))

		if v, ok := d.Type.(indigo.Proto); ok {
			types = append(types, v.Message)
		}
	}
	opts := []celgo.EnvOption{}
	opts = append(opts, celgo.Declarations(declarations...))
	//	opts = append(opts, celgo.TypeDescs(types...))
	opts = append(opts, celgo.Types(types...))
	return opts, nil
}

// --------------------------------------------------------------------------- DIAGNOSTICS

func printAST(ex *gexpr.Expr, n int, details *celgo.EvalDetails, data map[string]interface{}) string {
	s := strings.Builder{}

	indent := strings.Repeat(" ", n*2)

	var value string
	var valueSource string
	evaluatedValue, ok := details.State().Value(ex.Id)

	if ok {
		switch v := evaluatedValue.(type) {
		case ctypes.Duration:
			dur := time.Duration(v.Seconds() * float64(math.Pow10(9)))
			value = fmt.Sprintf("%60s", dur)
		case ctypes.Timestamp:
			value = fmt.Sprintf("%60s", time.Unix(int64(v.Second()), 0))
		default:
			value = fmt.Sprintf("%60s", fmt.Sprintf("%v", evaluatedValue))
		}
		valueSource = "E"
	} else {
		value = fmt.Sprintf("%60s (%v)", "?", ex.Id)
	}

	switch i := ex.GetExprKind().(type) {
	case *gexpr.Expr_CallExpr:
		s.WriteString(fmt.Sprintf("%s %s %s %s\n", value, valueSource, indent,
			strings.Trim(i.CallExpr.GetFunction(), "_")))
		for x := range i.CallExpr.Args {
			s.WriteString(printAST(i.CallExpr.Args[x], n+1, details, data))
		}
	case *gexpr.Expr_ComprehensionExpr:
		operandName := i.ComprehensionExpr.IterRange.GetSelectExpr().Operand.GetIdentExpr().GetName()
		fieldName := i.ComprehensionExpr.IterRange.GetSelectExpr().Field
		comprehensionName := i.ComprehensionExpr.LoopCondition.GetCallExpr().Function
		callExpression := getCallExpression(i.ComprehensionExpr.GetLoopStep().GetCallExpr())
		if comprehensionName == "@not_strictly_false" {
			comprehensionName = "all"
		}
		s.WriteString(fmt.Sprintf("%s %s %s %s.%s.%s %s\n", value, valueSource, indent,
			operandName, fieldName, comprehensionName, callExpression))
	case *gexpr.Expr_ConstExpr:
		s.WriteString(fmt.Sprintf("%s %s %s %s\n", value, valueSource, indent,
			strings.Trim(i.ConstExpr.String(), " ")))
	case *gexpr.Expr_SelectExpr:
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
	case *gexpr.Expr_IdentExpr:
		s.WriteString(fmt.Sprintf("%s %s %s %s\n", value, valueSource, indent, i.IdentExpr.Name))
	default:
		s.WriteString(fmt.Sprintf("%s %s %s %s\n", value, valueSource, indent, "** Unknown"))
	}
	return s.String()
}

func getCallExpression(e *gexpr.Expr_Call) string {

	x := ""

	if e.Function != "_&&_" {
		x = e.Function
	}

	for _, a := range e.Args {
		switch aa := a.GetExprKind().(type) {
		case *gexpr.Expr_IdentExpr:
			if aa.IdentExpr.Name != "__result__" {
				x = x + " " + aa.IdentExpr.Name
			}
		case *gexpr.Expr_CallExpr:
			x = x + "(" + getCallExpression(a.GetCallExpr()) + ")"
		case *gexpr.Expr_ConstExpr:
			x = x + " " + aa.ConstExpr.String()
		}
	}
	return x
}

func getSelectIdent(i *gexpr.Expr_SelectExpr) string {

	switch v := i.SelectExpr.Operand.GetExprKind().(type) {
	case *gexpr.Expr_SelectExpr:
		return getSelectIdent(v) + "." + v.SelectExpr.Field
	case *gexpr.Expr_IdentExpr:
		return v.IdentExpr.Name
	}

	return ""
}

func collectDiagnostics(ast *celgo.Ast, details *celgo.EvalDetails, data map[string]interface{}) string {

	if ast == nil || details == nil {
		fmt.Println(ast, details)
		return ""

	}

	s := strings.Builder{}
	s.WriteString("-------------------------------------------------------------------------------------------------\n")
	//	s.WriteString(fmt.Sprintf("Rule ID: %s\n", r.RuleID))
	s.WriteString("Expression:\n")
	s.WriteString(fmt.Sprintf("%s\n\n", wordWrap(ast.Source().Content(), 100)))
	// s.WriteString(fmt.Sprintf("Evaluation Result   :  %t\n", r.Pass))
	// s.WriteString(fmt.Sprintf("Evaluation Raw Value:  %v\n", r.Value))
	// s.WriteString(fmt.Sprintf("Rule Expression:\n"))
	// s.WriteString(fmt.Sprintf("%s\n", word_wrap(ast.Source().Content(), 100)))
	s.WriteString("-------------------------------------------------------------------------------------------------\n")
	s.WriteString("                                          EVALUATION TREE\n")
	s.WriteString("-------------------------------------------------------------------------------------------------\n")
	s.WriteString(fmt.Sprintf("%60s    %-30s\n", "VALUE", "EXPRESSION"))
	s.WriteString("-------------------------------------------------------------------------------------------------\n")
	s.WriteString(printAST(ast.Expr(), 0, details, data))
	return s.String()
}

func wordWrap(text string, lineWidth int) string {
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
