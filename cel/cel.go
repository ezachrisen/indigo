// package cel provides an implementation of the rules/Engine interface backed by Google's cel-go rules engine
// See https://github.com/google/cel-go and https://opensource.google/projects/cel for more information
// about CEL. The rules you write must conform to the CEL spec: https://github.com/google/cel-spec.

package cel

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ezachrisen/indigo"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	exprbp "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

type CELEngine struct {
	// Rules holds the raw rules passed by the user of the engine.
	rules map[string]indigo.Rule

	// Rules are parsed, checked and stored as runnable CEL prorgrams
	programs map[string]cel.Program

	// ASTs are the result of compiling a rule into a program. It is not necessary to keep
	// the ASTs for rule evaluation, but it is required to provide proper diagnostic information
	// if enabled through options.
	asts map[string]*cel.Ast

	opts indigo.EngineOptions
}

// Initialize a new CEL Engine
func NewEngine(opts ...indigo.EngineOption) *CELEngine {
	engine := CELEngine{}

	indigo.ApplyEngineOptions(&engine.opts, opts...)

	engine.rules = make(map[string]indigo.Rule)
	engine.programs = make(map[string]cel.Program)
	engine.asts = make(map[string]*cel.Ast)
	return &engine
}

// AddRule compiles the rule and adds it to the engine, ready to
// be evaluated.
// Any errors from the compilation will be returned.
func (e *CELEngine) AddRule(rules ...indigo.Rule) error {
	for _, r := range rules {

		if len(strings.Trim(r.ID, " ")) == 0 {
			return fmt.Errorf("Required rule ID for rule with expression %s", r.Expr)
		}

		err := e.addRuleWithSchema(r, r.Schema)
		if err != nil {
			return err
		}
		e.rules[r.ID] = r
	}
	return nil
}

// Find a rule with the given ID
func (e *CELEngine) Rule(id string) (indigo.Rule, bool) {
	r, ok := e.rules[id]
	return r, ok
}

func (e *CELEngine) Rules() map[string]indigo.Rule {
	return e.rules
}

func (e *CELEngine) PrintStructure() {
	fmt.Println("-------------------------------------------------- RULES")
	//	fmt.Printf("%v\n", e.rules)
	spew.Config.Indent = "\t"
	spew.Config.DisableCapacities = true
	spew.Config.DisablePointerAddresses = true
	spew.Config.DisablePointerMethods = true

	// spew.Dump(e.rules)
	fmt.Println("-------------------------------------------------- PROGRAMS")
	spew.Config.MaxDepth = 1
	//	spew.Dump(e.programs)
}

// Remove rule with the ID
func (e *CELEngine) RemoveRule(id string) {
	delete(e.rules, id)
	delete(e.programs, id)
}

func (e *CELEngine) RuleCount() int {
	return len(e.rules)
}

// Evaluate the rule against the input data.
// All rules will be evaluated, descending down through child rules up to the maximum depth
func (e *CELEngine) Evaluate(data map[string]interface{}, id string, opts ...indigo.EvalOption) (*indigo.Result, error) {
	o := indigo.EvalOptions{
		MaxDepth:   indigo.DefaultDepth,
		ReturnFail: true,
		ReturnPass: true,
	}

	indigo.ApplyEvalOptions(&o, opts...)

	if o.ReturnDiagnostics && !e.opts.CollectDiagnostics {
		return nil, fmt.Errorf("option set to return diagnostic, but engine does not have CollectDiagnostics option set")
	}

	rule, ok := e.rules[id]
	if !ok {
		return nil, fmt.Errorf("Rule not found")
	}

	return e.evaluate(data, &rule, 0, o)
}

func printAST(ex *exprbp.Expr, n int, details *cel.EvalDetails, data map[string]interface{}, r *indigo.Result) string {
	s := strings.Builder{}

	indent := strings.Repeat(" ", n*2)

	value := "?"
	valueSource := ""
	evaluatedValue, ok := details.State().Value(ex.Id)

	if ok {
		switch v := evaluatedValue.(type) {
		case types.Duration:
			dur := time.Duration(v.Seconds * int64(math.Pow10(9)))
			value = fmt.Sprintf("%30s", dur)
		case types.Timestamp:
			value = fmt.Sprintf("%30s", time.Unix(v.Seconds, 0))
		default:
			value = fmt.Sprintf("%30s", fmt.Sprintf("%v", evaluatedValue))
		}
		valueSource = "E"
	} else {
		value = fmt.Sprintf("%30s", "?")
	}

	switch i := ex.GetExprKind().(type) {
	case *exprbp.Expr_CallExpr:
		s.WriteString(fmt.Sprintf("%s %s %s %s\n", value, valueSource, indent, strings.Trim(i.CallExpr.GetFunction(), "_")))
		for x, _ := range i.CallExpr.Args {
			s.WriteString(printAST(i.CallExpr.Args[x], n+1, details, data, r))
		}
	case *exprbp.Expr_ComprehensionExpr:
		operandName := i.ComprehensionExpr.IterRange.GetSelectExpr().Operand.GetIdentExpr().GetName()
		fieldName := i.ComprehensionExpr.IterRange.GetSelectExpr().Field
		comprehensionName := i.ComprehensionExpr.LoopCondition.GetCallExpr().Function
		callExpression := getCallExpression(i.ComprehensionExpr.GetLoopStep().GetCallExpr())
		if comprehensionName == "@not_strictly_false" {
			comprehensionName = "all"
		}
		s.WriteString(fmt.Sprintf("%s %s %s %s.%s.%s %s\n", value, valueSource, indent, operandName, fieldName, comprehensionName, callExpression))
	case *exprbp.Expr_ConstExpr:
		s.WriteString(fmt.Sprintf("%s %s %s %s\n", value, valueSource, indent, strings.Trim(i.ConstExpr.String(), " ")))
	case *exprbp.Expr_SelectExpr:
		operandName := getSelectIdent(i)
		fieldName := i.SelectExpr.Field

		inputValue, ok := data[operandName+"."+fieldName]
		if ok {
			value = fmt.Sprintf("%30s", fmt.Sprintf("%v", inputValue))
			valueSource = "I"
		} else {
			obj, ok := data[operandName]
			if ok {
				x := reflect.ValueOf(obj).Elem()
				value = fmt.Sprintf("%30s", fmt.Sprintf("%v", x.FieldByName(fieldName)))
				valueSource = "I"
			}
		}

		s.WriteString(fmt.Sprintf("%s %s %s %s\n", value, valueSource, indent, operandName+"."+fieldName))
	case *exprbp.Expr_IdentExpr:
		s.WriteString(fmt.Sprintf("%s %s %s %s\n", value, valueSource, indent, i.IdentExpr.Name))
	default:
		s.WriteString(fmt.Sprintf("%s %s %s %s\n", value, valueSource, indent, "** Unknown"))
	}
	return s.String()
}

func getCallExpression(e *exprbp.Expr_Call) string {

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
	case *exprbp.Expr_SelectExpr:
		return getSelectIdent(v) + "." + v.SelectExpr.Field
	case *exprbp.Expr_IdentExpr:
		return v.IdentExpr.Name
	}

	return ""
}

func collectDiagnostics(ast *cel.Ast, details *cel.EvalDetails, data map[string]interface{}, r *indigo.Result) {

	if ast == nil || details == nil {
		return
	}

	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("----------------------------------------------------------------------------------------------------\n"))
	s.WriteString(fmt.Sprintf("Rule ID: %s\n", r.RuleID))
	s.WriteString(fmt.Sprintf("Expression:\n"))
	s.WriteString(fmt.Sprintf("%s\n\n", word_wrap(ast.Source().Content(), 100)))
	s.WriteString(fmt.Sprintf("Evaluation Result   :  %t\n", r.Pass))
	s.WriteString(fmt.Sprintf("Evaluation Raw Value:  %v\n", r.Value))
	s.WriteString(fmt.Sprintf("Rule Expression:\n"))
	s.WriteString(fmt.Sprintf("%s\n", word_wrap(ast.Source().Content(), 100)))
	s.WriteString(fmt.Sprintf("----------------------------------------------------------------------------------------------------\n"))
	s.WriteString(fmt.Sprintf("                                          EVALUATION TREE\n"))
	s.WriteString(fmt.Sprintf("----------------------------------------------------------------------------------------------------\n"))
	s.WriteString(fmt.Sprintf("%30s    %-30s\n", "VALUE", "EXPRESSION"))
	s.WriteString(fmt.Sprintf("----------------------------------------------------------------------------------------------------\n"))
	s.WriteString(printAST(ast.Expr(), 0, details, data, r))
	r.Diagnostics = s.String()
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

// Recursively evaluate the rule and its child indigo.
func (e *CELEngine) evaluate(data map[string]interface{}, rule *indigo.Rule, n int, opt indigo.EvalOptions) (*indigo.Result, error) {

	if n > opt.MaxDepth {
		return nil, nil
	}

	pr := indigo.Result{
		RuleID:      rule.ID,
		Meta:        rule.Meta,
		Action:      rule.Action,
		AsyncAction: rule.AsynchAction,
		Pass:        true,
		Results:     make(map[string]*indigo.Result, len(rule.Rules)),
	}

	// Apply options for this rule evaluation
	indigo.ApplyEvalOptions(&opt, rule.EvalOpts...)

	program, found := e.programs[rule.ID]

	// If the rule has an expression, evaluate it
	if program != nil && found {
		addSelf(data, rule.Self)
		rawValue, details, err := program.Eval(data)
		if err != nil {
			return nil, fmt.Errorf("Error evaluating rule %s:%w", rule.ID, err)
		}
		pr.Value = rawValue.Value()
		if v, ok := rawValue.Value().(bool); ok {
			pr.Pass = v
		}
		pr.RulesEvaluated++
		if e.opts.CollectDiagnostics && (opt.ReturnDiagnostics || e.opts.ForceDiagnosticsAllRules) {
			collectDiagnostics(e.asts[rule.ID], details, data, &pr)
		}
	} else {
		// If the rule has no expression default the result to true
		// Likely this means that this rule is a "set" of child rules,
		// and the user is only interested in the result of the children.
		pr.Value = true
		pr.Pass = true
	}

	if opt.StopIfParentNegative && pr.Pass == false {
		return &pr, nil
	}

	// Evaluate child rules
	for _, c := range rule.Rules {
		res, err := e.evaluate(data, &c, n+1, opt)
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

// Calculate a numeric expression and return the result.
// Although a numeric expression can be calculated in any rule and the result returned via Result.Value,
// this function provides type conversion convenience when it is known that the result will be numeric.
// If the expression is not already present, it will be compiled and added, before evaluating.
// Subsequent invocations of the expression will use the compiled progrom to evaluate.
// Returns an error if the expression evaluation result is not float64.
func (e *CELEngine) Calculate(data map[string]interface{}, expr string, schema indigo.Schema) (float64, error) {
	prg, found := e.programs[expr]
	if !found {
		r := indigo.Rule{
			ID:     expr,
			Expr:   expr,
			Schema: schema,
		}
		err := e.AddRule(r)
		if err != nil {
			return 0.0, err
		}
		prg = e.programs[expr]
	}

	result, err := e.evaluateProgram(data, prg)
	if err != nil {
		return 0.0, err
	}

	val, ok := result.(float64)
	if !ok {
		return 0.0, fmt.Errorf("Result (%v), type %T could not be converted to float64", result, result)
	}
	return val, nil
}

// Add the self object (if provided) to the data
func addSelf(data map[string]interface{}, self interface{}) {
	if self != nil {
		data[indigo.SelfKey] = self
	} else {
		delete(data, indigo.SelfKey)
	}
}

// Parse, check and store the rule
func (e *CELEngine) compileRule(env *cel.Env, r indigo.Rule) (cel.Program, error) {

	// Parse the rule expression to an AST
	p, iss := env.Parse(r.Expr)
	if iss != nil && iss.Err() != nil {
		return nil, fmt.Errorf("parsing rule %s:\n%s", r.ID, strings.ReplaceAll(fmt.Sprintf("%s", iss.Err()), "<input>:", ""))
	}

	// Type-check the parsed AST against the declarations
	c, iss := env.Check(p)
	if iss != nil && iss.Err() != nil {
		return nil, fmt.Errorf("checking rule %s:\n%w", r.ID, iss.Err())
	}

	if e.opts.CollectDiagnostics {
		e.asts[r.ID] = p
	}

	// Generate an evaluable program
	// 	cel.EvalOptions(cel.OptTrackState)

	options := cel.EvalOptions()
	if e.opts.CollectDiagnostics {
		options = cel.EvalOptions(cel.OptTrackState)
	}
	prg, err := env.Program(c, options) // cel.OptExhaustiveEval)) //OptTrackState))
	if err != nil {
		return nil, fmt.Errorf("generating program %s, %w", r.ID, err)
	}
	return prg, nil
}

// Excecute the stored program and provide the results
func (e *CELEngine) evaluateProgram(data map[string]interface{}, prg cel.Program) (interface{}, error) {
	rawValue, _, err := prg.Eval(data)
	if err != nil {
		return nil, err
	}

	return rawValue.Value(), nil
}

func (e *CELEngine) addRuleWithSchema(r indigo.Rule, s indigo.Schema) error {

	var opts []cel.EnvOption
	var err error
	var schemaToPassOn indigo.Schema
	// If the rule has a schema, use it, otherwise use the parent rule's
	if len(r.Schema.Elements) > 0 {
		opts, err = schemaToDeclarations(r.Schema)
		if err != nil {
			return err
		}
		schemaToPassOn = r.Schema
	} else if len(s.Elements) > 0 {
		opts, err = schemaToDeclarations(s)
		if err != nil {
			return err
		}
		schemaToPassOn = s
	}

	if opts == nil || len(opts) == 0 {
		return fmt.Errorf("No valid schema for rule %s", r.ID)
	}

	env, err := cel.NewEnv(opts...)
	if err != nil {
		return err
	}

	if r.Expr != "" {
		prg, err := e.compileRule(env, r)
		if err != nil {
			return fmt.Errorf("compiling rule %s\n%w", r.ID, err)
		}
		e.programs[r.ID] = prg
	}

	for _, c := range r.Rules {
		err = e.addRuleWithSchema(c, schemaToPassOn)
		if err != nil {
			return fmt.Errorf("adding rule %s: %w", c.ID, err)
		}
	}
	return nil
}

// celType converts from a indigo.Type to a CEL type
func celType(t indigo.Type) (*exprbp.Type, error) {

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
		//		protoMessage, ok := v.Message.(protoiface.MessageV1)
		//		if !ok {
		//			return nil, fmt.Errorf("Casting to proto message %v", v.Protoname)
		//		}
		//		_, err := pb.DefaultDb.RegisterMessage(protoMessage)

		// if err != nil {
		// 	return nil, fmt.Errorf("registering proto message %v: %w", v.Protoname, err)
		// }
		return decls.NewObjectType(v.Protoname), nil
	}
	return decls.Any, nil

}

// schemaToDeclarations converts from a rules/Schema to a set of CEL declarations that
// are passed to the CEL engine
func schemaToDeclarations(s indigo.Schema) ([]cel.EnvOption, error) {
	declarations := []*exprbp.Decl{}
	protoTypes := []interface{}{}

	for _, d := range s.Elements {
		typ, err := celType(d.Type)
		if err != nil {
			return nil, err
		}
		declarations = append(declarations, decls.NewVar(d.Name, typ))

		switch v := d.Type.(type) {
		case indigo.Proto:
			protoTypes = append(protoTypes, v.Message)
		}

	}
	opts := []cel.EnvOption{}
	opts = append(opts, cel.Declarations(declarations...))
	opts = append(opts, cel.Types(protoTypes...))

	return opts, nil
}
