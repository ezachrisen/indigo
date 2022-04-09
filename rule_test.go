package indigo_test

import (
	"testing"

	"github.com/ezachrisen/indigo"
	"github.com/matryer/is"
)

func TestNew(t *testing.T) {

	is := is.New(t)
	r := indigo.NewRule("blah", "")

	is.True(r.Rules != nil) // child map initialized
	is.True(r.ID == "blah")
	is.True(len(r.Schema.Elements) == 0)
}

func TestAdd(t *testing.T) {

	is := is.New(t)
	r := indigo.NewRule("blah", "")
	r.Schema = indigo.Schema{
		Name: "blahschema",
		Elements: []indigo.DataElement{
			{Name: "x", Type: indigo.Int{}},
		},
	}

	err := r.Add(indigo.NewRule("bc", "abc"))
	is.NoErr(err)
	is.True(len(r.Rules) == 1)
	is.True(len(r.Rules["bc"].Schema.Elements) == 1)
	is.True(r.Rules["bc"].Expr == "abc")

	err = r.Add(indigo.NewRule("bc", "newbc expr"))
	is.NoErr(err)
	is.True(len(r.Rules) == 1)
	is.True(len(r.Rules["bc"].Schema.Elements) == 1)
	is.True(r.Rules["bc"].Expr == "newbc expr")

	err = r.Add(nil)
	is.True(err != nil)
	is.Equal(err.Error(), "child rule is nil")

	err = r.Add(indigo.NewRule("", ""))
	is.True(err != nil)
	is.Equal(err.Error(), "child rule is missing ID")

	var x *indigo.Rule
	err = x.Add(indigo.NewRule("dummy", ""))
	is.True(err != nil)
	is.Equal(err.Error(), "parent rule is nil")

}
