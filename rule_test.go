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
