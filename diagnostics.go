package indigo

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Delta456/box-cli-maker/v2"
	"github.com/alexeyco/simpletable"
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
}

func (d *Diagnostics) AsString(r *Rule, data map[string]interface{}) string {
	Box := box.New(box.Config{Px: 2, Py: 1, Type: "Double", Color: "Cyan", TitlePos: "Top", ContentAlign: "Left"})

	s := strings.Builder{}
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

	e := d.diagnosticTable()
	s.WriteString("Evaluation State:\n")
	s.WriteString("-----------------\n")
	s.WriteString(e.String())

	if data != nil {
		dt := dataTable(data)
		s.WriteString("\n\n")
		s.WriteString("Input Data:\n")
		s.WriteString("-----------\n")
		s.WriteString(dt.String())
	}
	return Box.String("INDIGO EVALUATION DIAGNOSTIC REPORT", s.String())

}

func dataTable(data map[string]interface{}) *simpletable.Table {
	table := simpletable.New()
	table.Header = &simpletable.Header{
		Cells: []*simpletable.Cell{
			{Align: simpletable.AlignCenter, Text: "Name"},
			{Align: simpletable.AlignCenter, Text: "Value"},
		},
	}

	for k, v := range data {
		r := []*simpletable.Cell{
			{Text: k},
			{Text: fmt.Sprintf("%v", v)},
		}
		table.Body.Cells = append(table.Body.Cells, r)
	}

	table.SetStyle(simpletable.StyleUnicode)

	return table

}

func (d *Diagnostics) diagnosticTable() *simpletable.Table {

	table := simpletable.New()
	table.Header = &simpletable.Header{
		Cells: []*simpletable.Cell{
			{Align: simpletable.AlignCenter, Text: "Loc"},
			{Align: simpletable.AlignCenter, Text: "Expression"},
			{Align: simpletable.AlignCenter, Text: "Type"},
			{Align: simpletable.AlignCenter, Text: "Value"},
			{Align: simpletable.AlignCenter, Text: "Source"},
		},
	}
	fd := flattenDiagnostics(*d)
	sortListByPosition(fd)

	for _, cd := range fd {
		r := []*simpletable.Cell{
			{Align: simpletable.AlignRight, Text: fmt.Sprintf("%d:%d", cd.Line, cd.Column)},
			{Text: cd.Expr},
			{Text: fmt.Sprintf("%s", cd.Value.Type)},
			{Text: fmt.Sprintf("%v", cd.Value.Val)},
			{Text: fmt.Sprintf("%s", cd.Source)},
		}
		table.Body.Cells = append(table.Body.Cells, r)
	}

	table.SetStyle(simpletable.StyleUnicode)

	return table
}

// func (d Diagnostics) String() string {

// 	//	ind := strings.Repeat("", indent)
// 	s := strings.Builder{}

// 	fd := flattenDiagnostics(d)
// 	sortListByPosition(fd)
// 	//s.WriteString("----------------------------------------------------------------------------------------------------------------------------------------------\n")
// 	s.WriteString("==================================================================================================================================================\n")
// 	s.WriteString("| INDIGO EVALUATION DIAGNOSTICS                                                                                                       Value      |\n")
// 	s.WriteString("| Loc     Expression                                                    Type             Value                                        Source     |\n")
// 	s.WriteString("==================================================================================================================================================\n")
// 	for _, c := range fd {
// 		s.WriteString(one("| ", c, "|") + "\n")
// 	}
// 	s.WriteString("==================================================================================================================================================\n")
// 	//	s.WriteString("----------------------------------------------------------------------------------------------------------------------------------------------\n")

// 	// s.WriteString(fmt.Sprintf("%s%-7s %-30s %-10s %-30s\n", ind, fmt.Sprintf("%d:%d:", d.Line, d.Column), fmt.Sprintf("%-15s %v", d.Value.Type, d.Value.Val), d.Source, d.Expr))
// 	// for _, c := range d.Children {
// 	// 	s.WriteString(c.asString(indent))
// 	// }
// 	return s.String()
// }

// func one(prefix string, d Diagnostics, suffix string) string {
// 	return fmt.Sprintf("%s%-7s %-60s | %-59s  %-10s %s", prefix, fmt.Sprintf("%d:%d", d.Line, d.Column), d.Expr, fmt.Sprintf("%-15s %v", d.Value.Type, d.Value.Val), d.Source, suffix)
// }

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
