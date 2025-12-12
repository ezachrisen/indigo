package indigo

import "testing"

func TestVault_NilRules(t *testing.T) {
	u := Update(nil)
	if u.op != noOp {
		t.Errorf("got %v", u.op)
	}
}
