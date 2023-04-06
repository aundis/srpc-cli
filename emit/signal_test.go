package emit

import "testing"

func TestEmitSignal(t *testing.T) {
	err := EmitSignal(`C:\Users\85124\Desktop\abc`)
	if err != nil {
		t.Error(err)
		return
	}
}
