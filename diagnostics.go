package indigo

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Delta456/box-cli-maker/v2"
	//	"github.com/alexeyco/simpletable"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

//go:generate stringer -type=ValueSource
type ValueSource int

const (
	Input ValueSource = iota
	Evaluated
)

// Diagnostics holds the internal rule-engine intermediate results.
// Request diagnostics for an evaluation to help understand how the engine
// reached the final output value.
// Diagnostics is a nested set of nodes, with 1 root node per rule evaluated.
// The children represent elements of the expression evaluated.
type Diagnostics struct {
	Expr     string        // the part of the rule expression evaluated
	Value    Value         // the value of the expression part
	Source   ValueSource   // where the value came from: input data, or evaluted by the engine
	Line     int           // the 1-based line number in the original source expression
	Column   int           // the 0-based column number in the original source expression
	Offset   int           // the 0-based character offset from the start of the original source expression
	Children []Diagnostics // one child per sub-expression. Each Evaluator may produce different results.
}

// String produces an ASCII table with human-readable diagnostics.
func (d *Diagnostics) String() string {
	fd := flattenDiagnostics(*d)
	sortListByPosition(fd)

	tw := table.NewWriter()
	tw.SetTitle("\nINDIGO EVAL DIAGNOSTIC\n")
	tw.AppendSeparator()
	tw.AppendHeader(table.Row{"Expression", "Value", "Type", "Source", "Loc"})
	for _, cd := range fd {
		tw.AppendRow(table.Row{
			cd.Expr,
			fmt.Sprintf("%v", cd.Value.Val),
			cd.Value.Type.String(),
			cd.Source.String(),
			fmt.Sprintf("%d:%d", cd.Line, cd.Column),
		})
	}
	style := table.StyleLight
	style.Format.Header = text.FormatDefault
	tw.SetStyle(style)

	return tw.Render()
}

// DiagnosticsReport produces an ASCII report of the input rules, input data, the evaluation diagnostics and the results.
func DiagnosticsReport(r *Rule, data map[string]interface{}, u *Result) string {

	b := box.New(box.Config{Px: 2, Py: 1, Type: "Double", Color: "Cyan", TitlePos: "Top", ContentAlign: "Left"})

	s := strings.Builder{}
	if r == nil && data == nil && u == nil {
		s.WriteString("No rule, data or diagnostics provided")
	}

	if r != nil {
		s.WriteString("Rule:\n")
		s.WriteString("-----\n")
		s.WriteString(r.ID)
		s.WriteString("\n\n")
		s.WriteString("Expression:\n")
		s.WriteString("-----------\n")
		s.WriteString(wordWrap(r.Expr, 100))
		s.WriteString("\n\n")
	}

	if u != nil {
		s.WriteString("Results:\n")
		s.WriteString("--------\n")
		s.WriteString(u.String())
		s.WriteString("\n\n")
	}

	if u != nil && u.Diagnostics != nil {
		s.WriteString("Evaluation State:\n")
		s.WriteString("-----------------\n")
		s.WriteString(u.Diagnostics.String())
	}

	if data != nil {
		dt := dataTable(data)
		s.WriteString("\n\n")
		s.WriteString("Input Data:\n")
		s.WriteString("-----------\n")
		s.WriteString(dt.Render())
	}

	return b.String("INDIGO EVALUATION DIAGNOSTIC REPORT", s.String())
}

// dataTable renders a table of the input data to a rule
func dataTable(data map[string]interface{}) table.Writer {
	tw := table.NewWriter()
	tw.AppendHeader(table.Row{"Name", "Value"})
	for k, v := range data {
		tw.AppendRow(table.Row{
			k,
			fmt.Sprintf("%v", v),
		})
	}
	style := table.StyleLight
	style.Format.Header = text.FormatDefault
	tw.SetStyle(style)

	return tw
}

// flattenDiagnostics takes nested list of diagnostic nodes
// and creates a flat slice with all the nodes
func flattenDiagnostics(d Diagnostics) []Diagnostics {
	l := []Diagnostics{d}
	for _, c := range d.Children {
		l = append(l, flattenDiagnostics(c)...)
	}
	return l
}

// sort a flattened list of diagnostic nodes by their
// "position" in the expression source, the position being given
// by the character offset
func sortListByPosition(l []Diagnostics) {
	sort.Slice(l, func(i, j int) bool {
		return l[i].Offset < l[j].Offset
	})
}

// wordWrap wraps a string to a specific line width,
// using the strings.Fields function to determine what a word is.
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
