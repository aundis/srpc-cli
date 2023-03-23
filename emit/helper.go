package emit

import (
	"fmt"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sr/parse"
	"strings"

	"github.com/aundis/mate"
	"github.com/gogf/gf/v2/os/gfile"
)

func emitSlotHelper(root, module string, writer TextWriter, st *parse.StructType) error {
	emiter := helperEmiter{
		root:   root,
		module: module,
		writer: writer,
	}
	err := emiter.emitHelperRest(st.Parent, st.Name[1:], "slot", st.Functions)
	if err != nil {
		return err
	}
	return nil
}

func emitSignalHelper(root, module string, writer TextWriter, it *parse.InterfaceType) error {
	emiter := helperEmiter{
		root:   root,
		module: module,
		writer: writer,
	}
	err := emiter.emitHelperRest(it.Parent, it.Name[1:], "signal", it.Functions)
	if err != nil {
		return err
	}
	return nil
}

type helperEmiter struct {
	root   string
	module string
	writer TextWriter
}

func (e *helperEmiter) emitHelperRest(file *parse.File, name, kind string, funcs []*parse.Function) error {
	writer := e.writer
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
				fmate, err := e.resolveFieldMate(e.module, file, p.Name, p.Type, p.Pos)
				if err != nil {
					return err
				}
				e.emitFiledHelper(writer, fmate)
			}
			writer.DecreaseIndent().WriteString("},").WriteLine()
		}
		if len(f.Results) > 0 {
			writer.WriteString(`Results: []*mate.FieldMate{`).WriteLine().IncreaseIndent()
			for _, r := range f.Results {
				fmate, err := e.resolveFieldMate(e.module, file, r.Name, r.Type, r.Pos)
				if err != nil {
					return err
				}
				e.emitFiledHelper(writer, fmate)
			}
			writer.DecreaseIndent().WriteString("},").WriteLine()
		}
		writer.DecreaseIndent().WriteString("},").WriteLine()
	}
	writer.DecreaseIndent().WriteString("},").WriteLine()
	writer.DecreaseIndent().WriteString("}").WriteLine()
	return nil
}

func (e *helperEmiter) emitFiledHelper(writer TextWriter, fmate *mate.FieldMate) error {
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

func (e *helperEmiter) resolveFieldMate(module string, file *parse.File, name, tpe string, pos token.Pos) (*mate.FieldMate, error) {
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
				return nil, formatError(file.FileSet, pos, fmt.Sprintf("cannot parse scope:%s, please place the local type under the model package", scope))
			}
			if isProjectPackage(module, imp.Path) {
				// raw
				if imp.Export != "model" {
					fmt.Printf("warning: type %s.%s is not placed under the project's modal package, and code generation may cause duplicate names", scope, typeName)
				}
				// 提取本地类型的代码
				code, err := e.resolveLocalTypeCode(imp.Path, typeName)
				if err != nil {
					return nil, err
				}
				// 替换类型的scope为{localScope}
				scopeReg := regexp.MustCompile(fmt.Sprintf(`\b%s\.`, scope))
				fmate.Type = scopeReg.ReplaceAllString(fmate.Type, "{scope}.")
				// 存储这个本地类型
				fmate.Raws = append(fmate.Raws, &mate.CodeMate{
					Name: typeName,
					From: imp.Path,
					Code: code,
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

func (e *helperEmiter) resolveLocalTypeCode(pkgPath string, typeName string) (string, error) {
	localPath := e.convertPackagePathToLocalPath(pkgPath)
	modal, err := parse.ParsePackageModel(localPath)
	if err != nil {
		return "", err
	}
	if modal.ContainsType(typeName) {
		return string(modal.Types[typeName]), nil
	}
	return "", fmt.Errorf("package: %s, cannot found type: %s", pkgPath, typeName)
}

func (e *helperEmiter) convertPackagePathToLocalPath(pkgPath string) string {
	part := strings.Split(pkgPath, "/")
	return path.Join(e.root, strings.Join(part[1:], "/"))
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

func EmitCallInterfaceFromHelper(root string, target string, omate *mate.ObjectMate) error {
	module, err := getProjectModuleName(root)
	if err != nil {
		return err
	}
	emiter := &callInterfaceEmiter{
		root:   root,
		target: target,
		omate:  omate,
		module: module,
		writer: newTextWriter(),
	}
	err = emiter.emit()
	if err != nil {
		return err
	}
	return nil
}

type callInterfaceEmiter struct {
	root   string
	target string
	omate  *mate.ObjectMate
	module string
	writer TextWriter
}

func (e *callInterfaceEmiter) emit() error {
	err := e.emitHeader()
	if err != nil {
		return err
	}
	err = e.emitRaw()
	if err != nil {
		return err
	}
	err = e.emitBody()
	if err != nil {
		return err
	}

	outDir := path.Join(e.root, "internal", "srpc", "service", e.target)
	err = os.MkdirAll(outDir, os.ModePerm)
	if err != nil {
		return err
	}
	outPath := path.Join(e.root, "internal", "srpc", "service", e.target, toSnakeCase(e.omate.Name)+".go")
	err = ioutil.WriteFile(outPath, e.writer.Bytes(), os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func (e *callInterfaceEmiter) emitHeader() error {
	e.writer.WriteString("package ", e.target).WriteLine()
	err := e.emitImports()
	if err != nil {
		return err
	}
	return nil
}

func (e *callInterfaceEmiter) emitImports() error {
	collect := newImportCollect()
	var fmates []*mate.FieldMate
	for _, f := range e.omate.Functions {
		fmates = append(fmates, f.Parameters...)
		fmates = append(fmates, f.Results...)
	}
	for _, fmate := range fmates {
		for _, imp := range fmate.Imports {
			collect.Set(getImportMateExport(imp), imp.Path)
		}
		if len(fmate.Raws) > 0 {
			fmate.Type = strings.ReplaceAll(fmate.Type, "{scope}.", "")
			// collect.Set(e.target, e.module+"/internal/srpc/model/"+e.target)
		}
	}
	collect.Emit(e.writer)
	return nil
}

func (e *callInterfaceEmiter) emitRaw() error {
	var rmates []*mate.CodeMate
	var fmates []*mate.FieldMate
	for _, f := range e.omate.Functions {
		fmates = append(fmates, f.Parameters...)
		fmates = append(fmates, f.Results...)
	}
	for _, fmate := range fmates {
		rmates = append(rmates, fmate.Raws...)
	}
	serviceDir := path.Join(e.root, "internal", "srpc", "service", e.target)
	if !gfile.Exists(serviceDir) {
		err := os.MkdirAll(serviceDir, os.ModePerm)
		if err != nil {
			return err
		}
	}
	modelPath := path.Join(serviceDir, "model.go")
	model, err := parse.ParseFileModel(modelPath)
	if err != nil {
		return err
	}
	for _, rmate := range rmates {
		model.AddType(rmate.Name, []byte(rmate.Code))
	}
	writer := newTextWriter()
	writer.WriteString("package ", e.target).WriteLine()
	for _, v := range model.Types {
		writer.Write(v)
		writer.WriteLine()
	}
	err = ioutil.WriteFile(modelPath, writer.Bytes(), os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func (e *callInterfaceEmiter) emitBody() error {
	omate := e.omate
	writer := e.writer
	writer.WriteString("type I", omate.Name, " interface {").WriteLine().IncreaseIndent()
	for _, fmate := range omate.Functions {
		err := emitCallFunctionFromHelper(writer, fmate)
		if err != nil {
			return err
		}
	}
	writer.DecreaseIndent().WriteString("}").WriteLine()

	return nil
}

func getImportMateExport(imate *mate.ImportMate) string {
	if len(imate.Alias) > 0 {
		return imate.Alias
	}
	index := strings.LastIndex(imate.Path, "/") + 1
	return imate.Path[index:]
}

func emitCallFunctionFromHelper(writer TextWriter, fmate *mate.FunctionMate) error {
	writer.WriteString(fmate.Name, "(")
	for i, p := range fmate.Parameters {
		if i != 0 {
			writer.WriteString(", ")
		}
		writer.WriteString(p.Name, " ", p.Type)
	}
	writer.WriteString(") ")
	if len(fmate.Results) > 0 {
		writer.WriteString("(")
	}
	for i, r := range fmate.Results {
		if i != 0 {
			writer.WriteString(", ")
		}
		if len(r.Name) > 0 {
			writer.WriteString(r.Name, " ")
		}
		writer.WriteString(r.Type)
	}
	if len(fmate.Results) > 0 {
		writer.WriteString(")")
	}
	writer.WriteLine()
	return nil
}

func EmitListenFromHelper() {

}
