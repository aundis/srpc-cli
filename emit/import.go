package emit

import (
	"sr/parse"
	"strings"
)

func resolveImport(file *parse.File, packageName string) *parse.Import {
	for _, imp := range file.Imports {
		if imp.Export == packageName {
			return imp
		}
	}
	return nil
}

type importCollect struct {
	imports map[string]string
}

func newImportCollect() *importCollect {
	return &importCollect{
		imports: map[string]string{},
	}
}

func (c *importCollect) Set(name string, path string) {
	c.imports[name] = path
}

func (c *importCollect) Get(name string) string {
	return c.imports[name]
}

func (c *importCollect) Emit(writer TextWriter) {
	for name, path := range c.imports {
		if stringEndOf(path, name) {
			writer.WriteString(`import "` + path + `"`)
		} else {
			writer.WriteString("import " + name + ` "` + path + `"`)
		}
		writer.WriteLine()
	}
}

func stringEndOf(content string, part string) bool {
	return strings.LastIndex(content, part) == len(content)-len(part)
}

func resolveStructImports(st *parse.StructType, collect *importCollect) error {
	return resolveFieldImports(st.Parent, getStructFields(st), collect)
}

func resolveInterfaceImports(it *parse.InterfaceType, collect *importCollect) error {
	return resolveFieldImports(it.Parent, getInterfaceFields(it), collect)
}

func resolveFieldImports(file *parse.File, fields []*parse.Field, collect *importCollect) error {
	for _, field := range fields {
		expr := field.Type
		if len(expr) == 0 {
			continue
		}
		if !isUsePackage(expr) {
			continue
		}
		name := getPackageName(expr)
		imp := resolveImport(file, name)
		if imp == nil {
			return formatError(file.FileSet, field.Pos, "not found import "+name)
		}
		collect.Set(imp.Export, imp.Path)
	}
	return nil
}

func getStructFields(structType *parse.StructType) []*parse.Field {
	var result []*parse.Field
	for _, fun := range structType.Functions {
		result = append(result, fun.Params...)
		result = append(result, fun.Results...)
	}
	return result
}

func getInterfaceFields(interfaceType *parse.InterfaceType) []*parse.Field {
	var result []*parse.Field
	for _, fun := range interfaceType.Functions {
		result = append(result, fun.Params...)
		result = append(result, fun.Results...)
	}
	return result
}
