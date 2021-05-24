package cel

import (
	"fmt" // required by CEL to construct a proto from an expression
	"strings"

	"github.com/ezachrisen/indigo"

	celgo "github.com/google/cel-go/cel"
	gexpr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
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

	// A blank expression is ok, but it won't pass through the compilation
	if expr == "" {
		return nil, nil
	}

	var opts []celgo.EnvOption
	var err error

	// Convert from an Indigo schema to a set of CEL declarations (schema)
	opts, err = convertIndigoSchemaToDeclarations(s)
	if err != nil {
		return nil, err
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
			return nil, fmt.Errorf("compiling: %w", err)
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
// Expression  expression
func (*Evaluator) Evaluate(data map[string]interface{}, expr string,
	_ indigo.Schema, _ interface{}, evalData interface{}, expectedResultType indigo.Type,
	returnDiagnostics bool) (indigo.Value, *indigo.Diagnostics, error) {

	program, ok := evalData.(celProgram)

	// If the rule doesn't have a program, return a default result
	if !ok {
		// No program is not OK if we have an expression to evaluate
		if expr != "" {
			return indigo.Value{}, nil, fmt.Errorf("missing program")
		}
		return indigo.Value{true, indigo.Bool{}}, nil, nil
	}

	rawValue, details, err := program.program.Eval(data)
	// Do not check the error yet. Grab the diagnostics first
	var diagnostics *indigo.Diagnostics
	if returnDiagnostics {
		diagnostics, err = collectDiagnostics(program.ast, details, data)
		if err != nil {
			return indigo.Value{}, nil, fmt.Errorf("collecting diagnostics: %w", err)
		}
	}

	if err != nil {
		return indigo.Value{}, diagnostics, fmt.Errorf("evaluating rule: %w", err)
	}

	iv, err := convertRefValToIndigo(rawValue, expectedResultType)

	return iv, diagnostics, err
}

// --------------------------------------------------------------------------- TYPE CONVERSIONS
//

// // doTypesMatch determines if the indigo and cel types match, meaning
// // they can be converted from one to the other
func doTypesMatch(cel *gexpr.Type, indigo indigo.Type) error {

	celConverted, err := indigoType(cel)
	if err != nil {
		return err
	}

	if celConverted.String() != indigo.String() {
		return fmt.Errorf("types do no match: CEL: %T (%v), Indigo: %T (%v)", celConverted, celConverted, indigo, indigo)
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
		keyType, err := indigoType(v.MapType.KeyType)
		if err != nil {
			return nil, err
		}

		valType, err := indigoType(v.MapType.ValueType)
		if err != nil {
			return nil, err
		}

		return indigo.Map{
			KeyType:   keyType,
			ValueType: valType,
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
