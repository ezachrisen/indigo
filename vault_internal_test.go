package indigo

import "testing"

func TestVault_NilRules(t *testing.T) {
	a := Add(nil, "")
	if a.op != noOp {
		t.Errorf("got %v", a.op)
	}
	u := Update(nil)
	if u.op != noOp {
		t.Errorf("got %v", u.op)
	}
}
