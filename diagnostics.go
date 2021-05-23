package indigo

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Delta456/box-cli-maker/v2"
	//	"github.com/alexeyco/simpletable"
	"github.com/jedib0t/go-pretty/v6/table"
)

//go:generate stringer -type=ValueSource
type ValueSource int

const (
	Input ValueSource = iota
	Evaluated
)

type Diagnostics struct {
	Expr      string
	Value     Value
	Source    ValueSource
	Children  []Diagnostics
	InputData map[string]interface{}
	Line      int
	Column    int
	Offset    int
	Rule      *Rule
	Data      map[string]interface{}
}

func (d *Diagnostics) String() string {

	fd := flattenDiagnostics(*d)
	sortListByPosition(fd)

	Box := box.New(box.Config{Px: 2, Py: 1, Type: "Double", Color: "Cyan", TitlePos: "Top", ContentAlign: "Left"})

	s := strings.Builder{}
	if d.Rule != nil {
		s.WriteString("Rule:\n")
		s.WriteString("-----\n")
		s.WriteString(d.Rule.ID)
		s.WriteString("\n\n")
		s.WriteString("Expression:\n")
		s.WriteString("-----------\n")
		s.WriteString(wordWrap(d.Rule.Expr, 100))
		s.WriteString("\n\n")
	}

	s.WriteString("Evaluation State:\n")
	s.WriteString("-----------------\n")
	tw := table.NewWriter()
	tw.AppendHeader(table.Row{"Loc", "Expression", "Type", "Value", "Source"})
	for _, cd := range fd {
		tw.AppendRow(table.Row{
			fmt.Sprintf("%d:%d", cd.Line, cd.Column),
			cd.Expr,
			fmt.Sprintf("%s", cd.Value.Type),
			fmt.Sprintf("%v", cd.Value.Val),
			fmt.Sprintf("%s", cd.Source),
		})
	}

	tw.SetStyle(table.StyleLight)
	s.WriteString(tw.Render())

	if d.Data != nil {
		dt := dataTable(d.Data)
		s.WriteString("\n\n")
		s.WriteString("Input Data:\n")
		s.WriteString("-----------\n")
		s.WriteString(dt.Render())
	}
	return Box.String("INDIGO EVALUATION DIAGNOSTIC REPORT", s.String())
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

	tw.SetStyle(table.StyleLight)

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
