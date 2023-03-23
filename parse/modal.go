package parse

import (
	"path"

	"github.com/gogf/gf/v2/os/gfile"
)

var globalModel = map[string]*Model{}

type Model struct {
	Types map[string][]byte
}

func (m *Model) ContainsType(name string) bool {
	_, ok := m.Types[name]
	return ok
}

func (m *Model) AddType(name string, content []byte) {
	m.Types[name] = content
}

func (m *Model) RemoveType(name string) {
	delete(m.Types, name)
}

func ParseFileModel(filename string) (*Model, error) {
	model := &Model{
		Types: map[string][]byte{},
	}
	if gfile.Exists(filename) {
		f, err := ParseFile(filename)
		if err != nil {
			return nil, err
		}
		decodeAstFile(f, model)
	}
	return model, nil
}

func ParsePackageModel(dir string) (*Model, error) {
	dir = formatPath(dir)
	// 进行全局缓存
	if m, ok := globalModel[dir]; ok {
		return m, nil
	}

	files, err := listFile(dir)
	if err != nil {
		return nil, err
	}
	model := &Model{
		Types: map[string][]byte{},
	}
	for _, file := range files {
		if path.Ext(file) != ".go" {
			continue
		}
		f, err := ParseFile(file)
		if err != nil {
			return nil, err
		}
		decodeAstFile(f, model)
	}
	globalModel[dir] = model
	return model, nil
}

func decodeAstFile(f *File, model *Model) {
	content := f.Content
	for _, it := range f.InterfaceTypes {
		var bytes []byte
		bytes = append(bytes, []byte("type ")...)
		bytes = append(bytes, content[it.Pos-1:it.End-1]...)
		model.AddType(it.Name, bytes)
	}
	for _, st := range f.StructTypes {
		var bytes []byte
		bytes = append(bytes, []byte("type ")...)
		bytes = append(bytes, content[st.Pos-1:st.End-1]...)
		model.AddType(st.Name, bytes)
	}
}
