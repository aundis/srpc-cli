package emit

import (
	"fmt"
	"go/token"
	"os"
	"path"
	"regexp"
	"sr/parse"
	"sr/util"
	"strings"

	"github.com/aundis/meta"
	"github.com/gogf/gf/v2/os/gfile"
)

func emitSlotHelper(root, module string, writer util.TextWriter, st *parse.StructType) error {
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

func emitSignalHelper(root, module string, writer util.TextWriter, it *parse.InterfaceType) error {
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
	writer util.TextWriter
}

func (e *helperEmiter) emitHelperRest(file *parse.File, name, kind string, funcs []*parse.Function) error {
	writer := e.writer
	writer.WriteString("manager.AddObjectMetaHelper(meta.ObjectMeta{").WriteLine().IncreaseIndent()
	writer.WriteString(`Name: "`, name, `",`).WriteLine()
	writer.WriteString(`Kind: "`, kind, `",`).WriteLine()
	writer.WriteString(`Functions: []*meta.FunctionMeta{`).WriteLine().IncreaseIndent()
	for _, f := range funcs {
		writer.WriteString("{").WriteLine().IncreaseIndent()
		writer.WriteString(`Name: `, `"`, f.Name, `",`).WriteLine()
		if len(f.Params) > 0 {
			writer.WriteString(`Parameters: []*meta.FieldMeta{`).WriteLine().IncreaseIndent()
			for _, p := range f.Params {
				fmeta, err := e.resolveFieldMeta(e.module, file, p.Name, p.Type, p.Pos)
				if err != nil {
					return err
				}
				e.emitFiledHelper(writer, fmeta)
			}
			writer.DecreaseIndent().WriteString("},").WriteLine()
		}
		if len(f.Results) > 0 {
			writer.WriteString(`Results: []*meta.FieldMeta{`).WriteLine().IncreaseIndent()
			for _, r := range f.Results {
				fmeta, err := e.resolveFieldMeta(e.module, file, r.Name, r.Type, r.Pos)
				if err != nil {
					return err
				}
				e.emitFiledHelper(writer, fmeta)
			}
			writer.DecreaseIndent().WriteString("},").WriteLine()
		}
		writer.DecreaseIndent().WriteString("},").WriteLine()
	}
	writer.DecreaseIndent().WriteString("},").WriteLine()
	writer.DecreaseIndent().WriteString("})").WriteLine()
	return nil
}

func (e *helperEmiter) emitFiledHelper(writer util.TextWriter, fmeta *meta.FieldMeta) error {
	writer.WriteString("{").WriteLine().IncreaseIndent()
	writer.WriteString(`Name: "`, fmeta.Name, `",`).WriteLine()
	writer.WriteString(`Kind: "`, fmeta.Kind, `",`).WriteLine()
	writer.WriteString(`Type: "`, fmeta.Type, `",`).WriteLine()
	if len(fmeta.Imports) > 0 {
		writer.WriteString(`Imports: []*meta.ImportMeta{`).WriteLine().IncreaseIndent()
		for _, imp := range fmeta.Imports {
			writer.WriteString("{").WriteLine().IncreaseIndent()
			writer.WriteString(`Alias: "`, imp.Alias, `",`).WriteLine()
			writer.WriteString(`Path: "`, imp.Path, `",`).WriteLine()
			writer.DecreaseIndent().WriteString("},").WriteLine()
		}
		writer.DecreaseIndent().WriteString("},").WriteLine()
	}
	if len(fmeta.Raws) > 0 {
		writer.WriteString(`Raws: []*meta.CodeMeta{`).WriteLine().IncreaseIndent()
		for _, r := range fmeta.Raws {
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

func (e *helperEmiter) resolveFieldMeta(module string, file *parse.File, name, tpe string, pos token.Pos) (*meta.FieldMeta, error) {
	reg := regexp.MustCompile(`\b(\w+)\s*\.\s*(\w+)\b`)
	results := reg.FindAllStringSubmatch(tpe, -1)
	if len(results) > 0 {
		fmeta := &meta.FieldMeta{}
		fmeta.Name = name
		fmeta.Kind = "compound"
		fmeta.Type = tpe
		for i := 0; i < len(results); i++ {
			scope := results[i][1]
			typeName := results[i][2]
			imp := resolveImport(file, scope)
			if imp == nil {
				return nil, formatError(file.FileSet, pos, fmt.Sprintf("cannot parse scope:%s, please place the local type under the model package", scope), e.root)
			}
			if isProjectPackage(module, imp.Path) {
				// raw
				if imp.Export != "model" {
					fmt.Printf("warning: type %s.%s is not placed under the project's modal package, and code generation may cause duplicate names\n", scope, typeName)
				}
				// 提取本地类型的代码
				code, err := e.resolveLocalTypeCode(imp.Path, typeName)
				if err != nil {
					return nil, err
				}
				// 替换类型的scope为{localScope}
				scopeReg := regexp.MustCompile(fmt.Sprintf(`\b%s\.`, scope))
				fmeta.Type = scopeReg.ReplaceAllString(fmeta.Type, "{scope}.")
				// 存储这个本地类型
				fmeta.Raws = append(fmeta.Raws, &meta.CodeMeta{
					Name: typeName,
					From: imp.Path,
					Code: code,
				})
			} else {
				fmeta.Imports = append(fmeta.Imports, &meta.ImportMeta{
					Path:  imp.Path,
					Alias: imp.Name,
				})
			}
		}
		return fmeta, nil
	} else {
		// simple
		return &meta.FieldMeta{
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

func EmitCallInterfaceFromHelper(root string, target string, ometa *meta.ObjectMeta) error {
	module, err := getProjectModuleName(root)
	if err != nil {
		return err
	}
	emiter := &callInterfaceEmiter{
		root:   root,
		target: target,
		ometa:  ometa,
		module: module,
		writer: util.NewTextWriter(),
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
	ometa  *meta.ObjectMeta
	module string
	writer util.TextWriter
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
	err = ensureDirExist(outDir)
	if err != nil {
		return err
	}
	outPath := path.Join(e.root, "internal", "srpc", "service", e.target, toSnakeCase(e.ometa.Name)+".call.go")
	err = util.WriteGenerateFile(outPath, e.writer.Bytes(), e.root)
	if err != nil {
		return err
	}
	return nil
}

func (e *callInterfaceEmiter) emitHeader() error {
	e.writer.WriteString(generatedHeader).WriteLine()
	e.writer.WriteString("package ", e.target).WriteLine()
	err := e.emitImports()
	if err != nil {
		return err
	}
	return nil
}

func (e *callInterfaceEmiter) emitImports() error {
	collect := newImportCollect()
	var fmetas []*meta.FieldMeta
	for _, f := range e.ometa.Functions {
		fmetas = append(fmetas, f.Parameters...)
		fmetas = append(fmetas, f.Results...)
	}
	for _, fmeta := range fmetas {
		for _, imp := range fmeta.Imports {
			if len(imp.Alias) != 0 {
				collect.Set(imp.Alias, imp.Path)
			} else {
				collect.Set(getImportMetaExport(imp), imp.Path)
			}
		}
		if len(fmeta.Raws) > 0 {
			fmeta.Type = strings.ReplaceAll(fmeta.Type, "{scope}.", "")
			// collect.Set(e.target, e.module+"/internal/srpc/model/"+e.target)
		}
	}
	e.writer.WriteEmptyLine()
	collect.Emit(e.writer)
	return nil
}

func (e *callInterfaceEmiter) emitRaw() error {
	var rmetas []*meta.CodeMeta
	var fmetas []*meta.FieldMeta
	for _, f := range e.ometa.Functions {
		fmetas = append(fmetas, f.Parameters...)
		fmetas = append(fmetas, f.Results...)
	}
	for _, fmeta := range fmetas {
		rmetas = append(rmetas, fmeta.Raws...)
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
	for _, rmeta := range rmetas {
		model.AddType(rmeta.Name, []byte(rmeta.Code))
	}
	writer := util.NewTextWriter()
	writer.WriteString(generatedHeader).WriteLine()
	writer.WriteString("package ", e.target).WriteLine()
	for _, v := range model.Types {
		writer.WriteEmptyLine()
		writer.Write(v)
		writer.WriteLine()
	}
	err = util.WriteGenerateFile(modelPath, writer.Bytes(), e.root)
	if err != nil {
		return err
	}
	return nil
}

func (e *callInterfaceEmiter) emitBody() error {
	ometa := e.ometa
	writer := e.writer
	writer.WriteEmptyLine()
	writer.WriteString("type I", ometa.Name, " interface {").WriteLine().IncreaseIndent()
	for _, fmeta := range ometa.Functions {
		err := e.emitFunction(fmeta)
		if err != nil {
			return err
		}
	}
	writer.DecreaseIndent().WriteString("}").WriteLine()

	// var localMath IMath
	// func Math() IMath {
	// 	if localSort == nil {
	// 		panic("implement not found for interface IMath, forgot register?")
	// 	}
	// 	return localMath
	// }
	// func RegisterMath(i IMath) {
	// 	localMath = i
	// }
	writer.WriteEmptyLine()
	writer.WriteString("var local", e.ometa.Name, " I", e.ometa.Name).WriteLine()
	writer.WriteEmptyLine()
	writer.WriteString("func ", e.ometa.Name, "() I", e.ometa.Name, "{").WriteLine().IncreaseIndent()
	writer.WriteString("if local", e.ometa.Name, " == nil {").WriteLine().IncreaseIndent()
	writer.WriteString(`panic("implement not found for interface I`, e.ometa.Name, `, forgot register?")`).WriteLine()
	writer.DecreaseIndent().WriteString("}").WriteLine()
	writer.WriteString("return local", e.ometa.Name).WriteLine()
	writer.DecreaseIndent().WriteString("}").WriteLine()
	writer.WriteEmptyLine()
	writer.WriteString("func Register", e.ometa.Name, "(i I", e.ometa.Name, ") {").WriteLine().IncreaseIndent()
	writer.WriteString("local", e.ometa.Name, " = i").WriteLine()
	writer.DecreaseIndent().WriteString("}").WriteLine()
	return nil
}

func (e *callInterfaceEmiter) emitFunction(fmeta *meta.FunctionMeta) error {
	writer := e.writer
	writer.WriteString(fmeta.Name, "(")
	for i, p := range fmeta.Parameters {
		if i != 0 {
			writer.WriteString(", ")
		}
		writer.WriteString(p.Name, " ", p.Type)
	}
	writer.WriteString(") ")
	if len(fmeta.Results) > 0 {
		writer.WriteString("(")
	}
	for i, r := range fmeta.Results {
		if i != 0 {
			writer.WriteString(", ")
		}
		if len(r.Name) > 0 {
			writer.WriteString(r.Name, " ")
		}
		writer.WriteString(r.Type)
	}
	if len(fmeta.Results) > 0 {
		writer.WriteString(")")
	}
	writer.WriteLine()
	return nil
}

func EmitListenInterfaceFromHelper(root string, target string, ometa *meta.ObjectMeta) error {
	module, err := getProjectModuleName(root)
	if err != nil {
		return err
	}
	emiter := &listenInterfaceEmiter{
		root:   root,
		target: target,
		ometa:  ometa,
		module: module,
		writer: util.NewTextWriter(),
	}
	err = emiter.emit()
	if err != nil {
		return err
	}
	return nil
}

type listenInterfaceEmiter struct {
	root   string
	target string
	ometa  *meta.ObjectMeta
	module string
	writer util.TextWriter
}

func (e *listenInterfaceEmiter) emit() error {
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
	err = ensureDirExist(outDir)
	if err != nil {
		return err
	}
	outPath := path.Join(e.root, "internal", "srpc", "service", e.target, toSnakeCase(e.ometa.Name)+".listen.go")
	err = util.WriteGenerateFile(outPath, e.writer.Bytes(), e.root)
	if err != nil {
		return err
	}
	return nil
}

func (e *listenInterfaceEmiter) emitHeader() error {
	e.writer.WriteString(generatedHeader).WriteLine()
	e.writer.WriteString("package ", e.target).WriteLine()
	err := e.emitImports()
	if err != nil {
		return err
	}
	return nil
}

func (e *listenInterfaceEmiter) emitImports() error {
	collect := newImportCollect()
	var fmetas []*meta.FieldMeta
	for _, f := range e.ometa.Functions {
		fmetas = append(fmetas, f.Parameters...)
		fmetas = append(fmetas, f.Results...)
	}
	for _, fmeta := range fmetas {
		for _, imp := range fmeta.Imports {
			if len(imp.Alias) != 0 {
				collect.Set(imp.Alias, imp.Path)
			} else {
				collect.Set(getImportMetaExport(imp), imp.Path)
			}
		}
		if len(fmeta.Raws) > 0 {
			fmeta.Type = strings.ReplaceAll(fmeta.Type, "{scope}.", "")
			// collect.Set(e.target, e.module+"/internal/srpc/model/"+e.target)
		}
	}
	e.writer.WriteEmptyLine()
	collect.Emit(e.writer)
	return nil
}

func (e *listenInterfaceEmiter) emitRaw() error {
	var rmetas []*meta.CodeMeta
	var fmetas []*meta.FieldMeta
	for _, f := range e.ometa.Functions {
		fmetas = append(fmetas, f.Parameters...)
		fmetas = append(fmetas, f.Results...)
	}
	for _, fmeta := range fmetas {
		rmetas = append(rmetas, fmeta.Raws...)
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
	for _, rmeta := range rmetas {
		model.AddType(rmeta.Name, []byte(rmeta.Code))
	}
	writer := util.NewTextWriter()
	writer.WriteString(generatedHeader).WriteLine()
	writer.WriteString("package ", e.target).WriteLine()
	for _, v := range model.Types {
		writer.WriteEmptyLine()
		writer.Write(v)
		writer.WriteLine()
	}
	err = util.WriteGenerateFile(modelPath, writer.Bytes(), e.root)
	if err != nil {
		return err
	}
	return nil
}

func (e *listenInterfaceEmiter) emitBody() error {
	ometa := e.ometa
	writer := e.writer
	writer.WriteEmptyLine()
	writer.WriteString("type I", ometa.Name, " interface {").WriteLine().IncreaseIndent()
	for _, fmeta := range ometa.Functions {
		err := e.emitFunction(fmeta)
		if err != nil {
			return err
		}
	}
	writer.DecreaseIndent().WriteString("}").WriteLine()

	// var localMath IMath
	// func Math() IMath {
	// 	if localSort == nil {
	// 		panic("implement not found for interface IMath, forgot register?")
	// 	}
	// 	return localMath
	// }
	// func RegisterMath(i IMath) {
	// 	localMath = i
	// }
	writer.WriteEmptyLine()
	writer.WriteString("var local", e.ometa.Name, " I", e.ometa.Name).WriteLine()
	writer.WriteEmptyLine()
	writer.WriteString("func ", e.ometa.Name, "() I", e.ometa.Name, "{").WriteLine().IncreaseIndent()
	writer.WriteString("if local", e.ometa.Name, " == nil {").WriteLine().IncreaseIndent()
	writer.WriteString(`panic("implement not found for interface I`, e.ometa.Name, `, forgot register?")`).WriteLine()
	writer.DecreaseIndent().WriteString("}").WriteLine()
	writer.WriteString("return local", e.ometa.Name).WriteLine()
	writer.DecreaseIndent().WriteString("}").WriteLine()
	writer.WriteEmptyLine()
	writer.WriteString("func Register", e.ometa.Name, "(i I", e.ometa.Name, ") {").WriteLine().IncreaseIndent()
	writer.WriteString("local", e.ometa.Name, " = i").WriteLine()
	writer.DecreaseIndent().WriteString("}").WriteLine()
	return nil
}

func (e *listenInterfaceEmiter) emitFunction(fmeta *meta.FunctionMeta) error {
	writer := e.writer
	writer.WriteString("On", fmeta.Name, "(fun func(")
	for i, p := range fmeta.Parameters {
		if i != 0 {
			writer.WriteString(", ")
		}
		writer.WriteString(p.Name, " ", p.Type)
	}
	writer.WriteString(") ")
	if len(fmeta.Results) > 0 {
		writer.WriteString("(")
	}
	for i, r := range fmeta.Results {
		if i != 0 {
			writer.WriteString(", ")
		}
		if len(r.Name) > 0 {
			writer.WriteString(r.Name, " ")
		}
		writer.WriteString(r.Type)
	}
	if len(fmeta.Results) > 0 {
		writer.WriteString(")")
	}
	writer.WriteString(")")
	writer.WriteLine()
	return nil
}
