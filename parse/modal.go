package parse

import "path"

var globalModal = map[string]*Model{}

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

func ParsePackageModal(dir string) (*Model, error) {
	dir = formatPath(dir)
	// 进行全局缓存
	if m, ok := globalModal[dir]; ok {
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
		content := f.Content
		for _, it := range f.InterfaceTypes {
			model.AddType(it.Name, content[it.Pos-1:it.End-1])
		}
		for _, st := range f.StructTypes {
			model.AddType(st.Name, content[st.Pos-1:st.End-1])
		}
	}
	globalModal[dir] = model
	return model, nil
}
