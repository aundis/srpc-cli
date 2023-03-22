package emit

import "testing"

func TestSlot(t *testing.T) {
	err := EmitSlot(`C:\Users\85124\Desktop\abc`)
	if err != nil {
		t.Error(err)
		return
	}
}
