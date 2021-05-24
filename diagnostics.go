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

type Diagnostics struct {
	Expr     string
	Value    Value
	Source   ValueSource
	Children []Diagnostics
	Line     int
	Column   int
	Offset   int
}

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

func DiagnosticsReport(r *Rule, data map[string]interface{}, d *Diagnostics) string {

	b := box.New(box.Config{Px: 2, Py: 1, Type: "Double", Color: "Cyan", TitlePos: "Top", ContentAlign: "Left"})

	s := strings.Builder{}

	if r == nil && data == nil && d == nil {
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

	if d != nil {
		s.WriteString("Evaluation State:\n")
		s.WriteString("-----------------\n")
		s.WriteString(d.String())
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

func flattenDiagnostics(d Diagnostics) []Diagnostics {
	l := []Diagnostics{}

	l = append(l, d)

	for _, c := range d.Children {
		l = append(l, flattenDiagnostics(c)...)
	}
	return l
}

func sortListByPosition(l []Diagnostics) {
	sort.Slice(l, func(i, j int) bool {
		return l[i].Offset < l[j].Offset
	})
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
