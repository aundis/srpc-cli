package emit

import (
	"sr/parse"
	"testing"

	"github.com/aundis/meta"
)

func TestResolver(t *testing.T) {
	resolver := &typeResolver{
		module:   "abc",
		root:     `testdata/resolver`,
		resolved: map[string]*meta.TypeMeta{},
	}
	file, err := parse.ParseFile(`testdata/resolver/model1/model.go`)
	if err != nil {
		t.Error(err)
		return
	}
	template, err := resolver.resolve(file, "model2.M2", 0)
	if err != nil {
		t.Error(err)
		return
	}
	list := resolver.getTypeMetas()
	if len(list) != 3 {
		t.Errorf("except type metas count = 3, bug got %d", len(list))
		return
	}
	if template == "model2.M2" {
		t.Errorf("except tempalte change, bug got %s", template)
		return
	}
}
