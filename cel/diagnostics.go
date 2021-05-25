// This file contains functions that collect and process diagnostic data from a CEL evaluation.
// The diagnostic data is returned to the Indigo user if requested.
package cel

import (
	"fmt"
	"strings"

	"github.com/ezachrisen/indigo"
	celgo "github.com/google/cel-go/cel"
	gexpr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// collectDiagnostics walks the CEL AST and annotates it with the result of the evaluation, returning
// a Diagnositc node.
func collectDiagnostics(ast *celgo.Ast, details *celgo.EvalDetails,
	data map[string]interface{}) (*indigo.Diagnostics, error) {

	if ast == nil || details == nil {
		return nil, fmt.Errorf("no ast or eval details")
	}

	d, err := printAST(ast.Expr(), 0, details, ast, data)
	if err != nil {
		return nil, err
	}

	return &d, nil
}

// printAST recursively walks the expression and its children, returning an indigo.Diagnostics
// node.
func printAST(ex *gexpr.Expr, n int, details *celgo.EvalDetails,
	ast *celgo.Ast, data map[string]interface{}) (indigo.Diagnostics, error) {

	d := indigo.Diagnostics{}
	evaluatedValue, ok := details.State().Value(ex.Id)
	if !ok {
		// The expression was not evaluated by CEL.
		// This happens when an expression's result can be
		// determined without evaluating all expressions. For
		// example, in the expression "true || blah > 2.0", "blah > 2.0"
		// will never be evaluated.
		return d, nil
	}

	d.Source = indigo.Evaluated

	value, err := convertRefValToIndigo2(evaluatedValue)
	if err != nil {
		return d, fmt.Errorf("converting from evaluated value to indigo value: %w", err)
	}

	d.Value = value
	d.Offset, d.Line, d.Column = getLocation(ex.Id, ast) // some types may override this below

	switch i := ex.GetExprKind().(type) {
	case *gexpr.Expr_CallExpr:
		d.Expr = strings.Trim(i.CallExpr.GetFunction(), "_")
		for x := range i.CallExpr.Args {
			dc, err := printAST(i.CallExpr.Args[x], n+1, details, ast, data)
			if err != nil {
				return d, fmt.Errorf("callExpr %d: %w", x, err)
			}
			if dc.Expr != "" {
				d.Children = append(d.Children, dc)
			}
		}
	case *gexpr.Expr_ComprehensionExpr:
		operandName := i.ComprehensionExpr.IterRange.GetSelectExpr().Operand.GetIdentExpr().GetName()
		fieldName := i.ComprehensionExpr.IterRange.GetSelectExpr().Field
		comprehensionName := i.ComprehensionExpr.LoopCondition.GetCallExpr().Function
		callExpression := getCallExpression(i.ComprehensionExpr.GetLoopStep().GetCallExpr())
		if comprehensionName == "@not_strictly_false" {
			comprehensionName = "all"
		}
		d.Expr = fmt.Sprintf("%s.%s.%s %s", operandName, fieldName, comprehensionName, callExpression)
	case *gexpr.Expr_ConstExpr:
		d.Expr = i.ConstExpr.String()
	case *gexpr.Expr_SelectExpr:
		operandName := getSelectIdent(i)
		//fieldName := i.SelectExpr.Field
		//fmt.Println("operand ", operandName, "fieldname", fieldName, "Operand ID: ", i.SelectExpr.Operand.Id)
		oper := i.SelectExpr.Operand
		if oper == nil {
			return d, fmt.Errorf("missing select operand")
		}
		d.Offset, d.Line, d.Column = getLocation(oper.Id, ast)
		// dottedName := operandName + "." + fieldName
		// inputValue, ok := data[dottedName]

		if ok {
			//			value = fmt.Sprintf("%60s", fmt.Sprintf("%v", inputValue))
			d.Source = indigo.Input
		} else {
			_, ok := data[operandName]
			if ok {
				//value = fmt.Sprintf("%60s", fmt.Sprintf("%v", obj)) //fmt.Sprintf("%v", x.FieldByName(fieldName)))
				d.Source = indigo.Input
			}
		}
		//		s.WriteString(fmt.Sprintf("%s %s %s %s\n", value, valueSource, indent, operandName+"."+fieldName))
	case *gexpr.Expr_IdentExpr:
		d.Expr = i.IdentExpr.Name
	default:
		d.Expr = "undefined"
	}
	return d, nil
}

// getLocation determines the position of an expression with the id in the original
// source expression for the rule
func getLocation(id int64, ast *celgo.Ast) (offset, line, column int) {

	si := ast.SourceInfo()
	if si == nil {
		return
	}

	offs, ok := si.Positions[id]
	if !ok {
		return
	}
	s := ast.Source()
	if s == nil {
		return
	}

	loc, ok := s.OffsetLocation(offs)
	if !ok {
		return
	}

	line = loc.Line()
	column = loc.Column()
	offset = int(offs)
	return
}

// getCallExpression unwraps a function call, returning a string representation
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

// getSelectIdent returns a string showing a struct field selection (e.g., myobj.myfield)
func getSelectIdent(i *gexpr.Expr_SelectExpr) string {
	switch v := i.SelectExpr.Operand.GetExprKind().(type) {
	case *gexpr.Expr_SelectExpr:
		return getSelectIdent(v) + fmt.Sprintf("[%v]", v) + "." + v.SelectExpr.Field
	case *gexpr.Expr_IdentExpr:
		return v.IdentExpr.Name
	}

	return ""
}
