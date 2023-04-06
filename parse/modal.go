package parse

import (
	"path"

	"github.com/gogf/gf/v2/os/gfile"
)

var globalModel = map[string]*Model{}

type ModelType struct {
	Raw     interface{}
	Content []byte
}

type Model struct {
	filename string
	imports  map[string]string
	types    map[string]*ModelType
}

func (m *Model) GetFileName() string {
	return m.filename
}

func (m *Model) ContainsType(name string) bool {
	_, ok := m.types[name]
	return ok
}

func (m *Model) AddType(name string, tpe *ModelType) {
	m.types[name] = tpe
}

func (m *Model) GetType(name string) *ModelType {
	return m.types[name]
}

func (m *Model) GetTypes() map[string]*ModelType {
	return m.types
}

func (m *Model) AddImport(name string, path string) {
	m.imports[name] = path
}

func (m *Model) GetImports() map[string]string {
	return m.imports
}

func (m *Model) RemoveType(name string) {
	delete(m.types, name)
}

func ParseFileModel(filename string) (*Model, error) {
	filename = formatPath(filename)
	if m, ok := globalModel[filename]; ok {
		return m, nil
	}
	model := &Model{
		filename: filename,
		types:    map[string]*ModelType{},
		imports:  map[string]string{},
	}
	if gfile.Exists(filename) {
		f, err := ParseFile(filename)
		if err != nil {
			return nil, err
		}
		decodeAstFile(f, model)
	}
	globalModel[filename] = model
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
		filename: dir,
		types:    map[string]*ModelType{},
		imports:  map[string]string{},
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
	for _, v := range f.Imports {
		model.imports[v.Export] = v.Path
	}
	for _, it := range f.InterfaceTypes {
		var bytes []byte
		bytes = append(bytes, []byte("type ")...)
		bytes = append(bytes, content[it.Pos-1:it.End-1]...)
		model.AddType(it.Name, &ModelType{
			Raw:     it,
			Content: bytes,
		})
	}
	for _, st := range f.StructTypes {
		var bytes []byte
		bytes = append(bytes, []byte("type ")...)
		bytes = append(bytes, content[st.Pos-1:st.End-1]...)
		model.AddType(st.Name, &ModelType{
			Raw:     st,
			Content: bytes,
		})
	}
}
