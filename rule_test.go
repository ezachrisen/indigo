package indigo_test

import (
	"testing"

	"github.com/ezachrisen/indigo"
)

func TestNew(t *testing.T) {
	r := indigo.NewRule("blah", "")

	if r.Rules == nil {
		t.Error("expected Rules to be initialized")
	}
	if r.ID != "blah" {
		t.Errorf("expected ID to be 'blah', got %q", r.ID)
	}
	if len(r.Schema.Elements) != 0 {
		t.Errorf("expected Schema.Elements length to be 0, got %d", len(r.Schema.Elements))
	}
}
