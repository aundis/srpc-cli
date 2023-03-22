package parse

import "testing"

func TestTag(t *testing.T) {
	tag := Tag("`name: \"123\"`,listen:\"main\"")
	if tag.Get("name") != "123" {
		t.Errorf("except name = %s, but got %s", "123", tag.Get("name"))
		return
	}
	if tag.Get("listen") != "main" {
		t.Errorf("except name = %s, but got %s", "main", tag.Get("main"))
		return
	}
}
