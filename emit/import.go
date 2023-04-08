package emit

import (
	"sr/parse"
	"sr/util"

	"github.com/aundis/meta"
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

func (c *importCollect) Emit(writer util.TextWriter) {
	for name, path := range c.imports {
		if util.StringEndOf(path, name) {
			writer.WriteString(`import "` + path + `"`)
		} else {
			writer.WriteString("import " + name + ` "` + path + `"`)
		}
		writer.WriteLine()
	}
}

func resolveStructImports(st *parse.StructType, collect *importCollect, toPackage, module, root string) error {
	return resolveFieldImports(st.Parent, getStructFields(st), collect, toPackage, module, root)
}

func resolveInterfaceImports(it *parse.InterfaceType, collect *importCollect, toPackage, module, root string) error {
	return resolveFieldImports(it.Parent, getInterfaceFields(it), collect, toPackage, module, root)
}

func resolveFieldImports(file *parse.File, fields []*parse.Field, collect *importCollect, toPackage string, module string, root string) error {
	for _, field := range fields {
		if !hasCustomerType(field.Type) {
			continue
		}
		resolver := typeResolver{
			module:   module,
			root:     root,
			resolved: map[string]*meta.TypeMeta{},
		}
		template, err := resolver.resolve(file, field.Type, field.Pos)
		if err != nil {
			return err
		}
		tmetas := resolver.getTypeMetas()
		ids := findAllTypeMetaIds(template)
		for _, id := range ids {
			tmeta := findTypeMetaForId(tmetas, id)
			if tmeta == nil {
				return formatError(file.FileSet, field.Pos, "not found type meta "+id, root)
			}
			if tmeta.Import != nil {
				collect.Set(getImportMetaExport(tmeta.Import), tmeta.Import.Path)
			}
			if len(tmeta.Code) > 0 && tmeta.From != toPackage {
				collect.Set(getImportPathExport(tmeta.From), tmeta.From)
			}
		}
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
