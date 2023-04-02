package emit

import (
	"sr/parse"
	"sr/util"
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
		if util.StringEndOf(path, name) {
			writer.WriteString(`import "` + path + `"`)
		} else {
			writer.WriteString("import " + name + ` "` + path + `"`)
		}
		writer.WriteLine()
	}
}

func resolveStructImports(st *parse.StructType, collect *importCollect, root string) error {
	return resolveFieldImports(st.Parent, getStructFields(st), collect, root)
}

func resolveInterfaceImports(it *parse.InterfaceType, collect *importCollect, root string) error {
	return resolveFieldImports(it.Parent, getInterfaceFields(it), collect, root)
}

func resolveFieldImports(file *parse.File, fields []*parse.Field, collect *importCollect, root string) error {
	for _, field := range fields {
		expr := field.Type
		if len(expr) == 0 {
			continue
		}
		refs := getRefExprs(expr)
		if len(refs) == 0 {
			continue
		}
		for _, v := range refs {
			imp := resolveImport(file, v.scope)
			if imp == nil {
				return formatError(file.FileSet, field.Pos, "not found import "+v.scope, root)
			}
			collect.Set(imp.Export, imp.Path)
		}
		// name := getPackageName(expr)
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
