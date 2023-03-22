package emit

import (
	"fmt"
	"go/token"
	"regexp"
	"sr/parse"
	"strings"

	"github.com/aundis/mate"
)

func emitSlotHelper(writer TextWriter, st *parse.StructType) error {
	return emitHelperRest(writer, st.Parent, st.Name[1:], "slot", st.Functions)
}

func emitSignalHelper(writer TextWriter, it *parse.InterfaceType) error {
	return emitHelperRest(writer, it.Parent, it.Name[1:], "signal", it.Functions)
}

func emitHelperRest(writer TextWriter, file *parse.File, name, kind string, funcs []*parse.Function) error {
	writer.WriteString("var ", firstLower(name), "Helper = mate.ObjectMate{").WriteLine().IncreaseIndent()
	writer.WriteString(`Name: "`, name, `",`).WriteLine()
	writer.WriteString(`Kind: "`, kind, `",`).WriteLine()
	writer.WriteString(`Functions: []*mate.FunctionMate{`).WriteLine().IncreaseIndent()
	for _, f := range funcs {
		writer.WriteString("{").WriteLine().IncreaseIndent()
		writer.WriteString(`Name: `, `"`, f.Name, `",`).WriteLine()
		if len(f.Params) > 0 {
			writer.WriteString(`Parameters: []*mate.FieldMate{`).WriteLine().IncreaseIndent()
			for _, p := range f.Params {
				fmate, err := resolveFieldMate(globalModule, file, p.Name, p.Type, p.Pos)
				if err != nil {
					return err
				}
				emitFiledHelper(writer, fmate)
			}
			writer.DecreaseIndent().WriteString("},").WriteLine()
		}
		if len(f.Results) > 0 {
			writer.WriteString(`Results: []*mate.FieldMate{`).WriteLine().IncreaseIndent()
			for _, r := range f.Results {
				fmate, err := resolveFieldMate(globalModule, file, r.Name, r.Type, r.Pos)
				if err != nil {
					return err
				}
				emitFiledHelper(writer, fmate)
			}
			writer.DecreaseIndent().WriteString("},").WriteLine()
		}
		writer.DecreaseIndent().WriteString("},").WriteLine()
	}
	writer.DecreaseIndent().WriteString("},").WriteLine()
	writer.DecreaseIndent().WriteString("}").WriteLine()
	return nil
}

func emitFiledHelper(writer TextWriter, fmate *mate.FieldMate) error {
	writer.WriteString("{").WriteLine().IncreaseIndent()
	writer.WriteString(`Name: "`, fmate.Name, `",`).WriteLine()
	writer.WriteString(`Kind: "`, fmate.Kind, `",`).WriteLine()
	writer.WriteString(`Type: "`, fmate.Type, `",`).WriteLine()
	if len(fmate.Imports) > 0 {
		writer.WriteString(`Imports: []*mate.ImportMate{`).WriteLine().IncreaseIndent()
		for _, imp := range fmate.Imports {
			writer.WriteString("{").WriteLine().IncreaseIndent()
			writer.WriteString(`Alias: "`, imp.Alias, `",`).WriteLine()
			writer.WriteString(`Path: "`, imp.Path, `",`).WriteLine()
			writer.DecreaseIndent().WriteString("},").WriteLine()
		}
		writer.DecreaseIndent().WriteString("},").WriteLine()
	}
	if len(fmate.Raws) > 0 {
		writer.WriteString(`Raws: []*mate.CodeMate{`).WriteLine().IncreaseIndent()
		for _, r := range fmate.Raws {
			writer.WriteString("{").WriteLine().IncreaseIndent()
			writer.WriteString(`Name: "`, r.Name, `",`).WriteLine()
			writer.WriteString(`From: "`, r.From, `",`).WriteLine()
			writer.WriteString(`Code: "`, formatToCodeString(r.Code), `",`).WriteLine()
			writer.DecreaseIndent().WriteString("},").WriteLine()
		}
		writer.DecreaseIndent().WriteString("},").WriteLine()
	}
	writer.DecreaseIndent().WriteString("},").WriteLine()
	return nil
}

func resolveFieldMate(module string, file *parse.File, name, tpe string, pos token.Pos) (*mate.FieldMate, error) {
	reg := regexp.MustCompile(`\b(\w+)\s*\.\s*(\w+)\b`)
	results := reg.FindAllStringSubmatch(tpe, -1)
	if len(results) > 0 {
		fmate := &mate.FieldMate{}
		fmate.Name = name
		fmate.Kind = "compound"
		fmate.Type = tpe
		for i := 0; i < len(results); i++ {
			scope := results[i][1]
			typeName := results[i][2]
			imp := resolveImport(file, scope)
			if imp == nil {
				return nil, formatError(file.FileSet, pos, fmt.Sprintf("无法解析scope:%s, 本地类型请放置在model包下", scope))
			}
			if isProjectPackage(module, imp.Path) {
				// raw
				if imp.Export != "model" {
					fmt.Printf("警告: 类型%s.%s未放置在项目的modal包下, 代码生成可能会造成重名", scope, typeName)
				}
				// 提取本地类型的代码
				code, err := resolveLocalTypeCode(imp.Path, typeName)
				if err != nil {
					return nil, err
				}
				// 替换类型的scope为{localScope}
				scopeReg := regexp.MustCompile(fmt.Sprintf(`\b%s\.`, scope))
				fmate.Type = scopeReg.ReplaceAllString(fmate.Type, "{scope}.")
				// 存储这个本地类型
				fmate.Raws = append(fmate.Raws, &mate.CodeMate{
					Name: name,
					From: imp.Path,
					Code: "type " + code,
				})
			} else {
				fmate.Imports = append(fmate.Imports, &mate.ImportMate{
					Path:  imp.Path,
					Alias: imp.Name,
				})
			}
		}
		return fmate, nil
	} else {
		// simple
		return &mate.FieldMate{
			Name: name,
			Kind: "simple",
			Type: tpe,
		}, nil
	}
}

func formatToCodeString(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\n", "\\n")
	content = strings.ReplaceAll(content, `"`, `\"`)
	return content
}

func isProjectPackage(module string, path string) bool {
	return strings.Contains(path, module+"/")
}
