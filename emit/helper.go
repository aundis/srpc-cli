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
				writer.WriteString("{").WriteLine().IncreaseIndent()
				writer.WriteString(`Name: `, `"`, p.Name, `",`).WriteLine()
				if hasCustomerType(p.Type) {
					template, typeMetas, err := e.resolveTypeMetas(f.Parent, p.Type, p.Pos)
					if err != nil {
						return err
					}
					writer.WriteString(`Type : `, `"`, template, `",`).WriteLine()
					e.emitTypeMetas(typeMetas)
				} else {
					writer.WriteString(`Type : `, `"`, p.Type, `",`).WriteLine()
				}
				writer.DecreaseIndent().WriteString("},").WriteLine()
			}
			writer.DecreaseIndent().WriteString("},").WriteLine()
		}
		if len(f.Results) > 0 {
			writer.WriteString(`Results: []*meta.FieldMeta{`).WriteLine().IncreaseIndent()
			for _, r := range f.Results {
				writer.WriteString("{").WriteLine().IncreaseIndent()
				writer.WriteString(`Name: `, `"`, r.Name, `",`).WriteLine()
				if hasCustomerType(r.Type) {
					template, typeMetas, err := e.resolveTypeMetas(f.Parent, r.Type, r.Pos)
					if err != nil {
						return err
					}
					writer.WriteString(`Type : `, `"`, template, `",`).WriteLine()
					e.emitTypeMetas(typeMetas)
				} else {
					writer.WriteString(`Type : `, `"`, r.Type, `",`).WriteLine()
				}
				writer.DecreaseIndent().WriteString("},").WriteLine()
			}
			writer.DecreaseIndent().WriteString("},").WriteLine()
		}
		writer.DecreaseIndent().WriteString("},").WriteLine()
	}
	writer.DecreaseIndent().WriteString("},").WriteLine()
	writer.DecreaseIndent().WriteString("})").WriteLine()
	return nil
}

func (e *helperEmiter) resolveTypeMetas(file *parse.File, compound string, pos token.Pos) (string, []*meta.TypeMeta, error) {
	resolver := &typeResolver{
		module:   e.module,
		root:     e.root,
		resolved: map[string]*meta.TypeMeta{},
	}
	template, err := resolver.resolve(file, compound, pos)
	if err != nil {
		return "", nil, err
	}
	return template, resolver.getTypeMetas(), nil
}

func (e *helperEmiter) emitTypeMetas(typeMetas []*meta.TypeMeta) error {
	if len(typeMetas) == 0 {
		return nil
	}
	writer := e.writer
	writer.WriteString(`TypeMetas: []*meta.TypeMeta{`).WriteLine().IncreaseIndent()
	for _, t := range typeMetas {
		writer.WriteString("{").WriteLine().IncreaseIndent()
		writer.WriteString(`Id: "`, t.Id, `",`).WriteLine()
		writer.WriteString(`Name: "`, t.Name, `",`).WriteLine()
		writer.WriteString(`From: "`, t.From, `",`).WriteLine()
		writer.WriteString(`Code: "`, formatToCodeString(t.Code), `",`).WriteLine()
		if t.Import != nil {
			writer.WriteString(`Import: &meta.ImportMeta{`).WriteLine().IncreaseIndent()
			writer.WriteString(`Path: "`, t.Import.Path, `",`).WriteLine()
			writer.WriteString(`Alias: "`, t.Import.Alias, `",`).WriteLine()
			writer.DecreaseIndent().WriteString("},").WriteLine()
		}
		writer.DecreaseIndent().WriteString("},").WriteLine()
	}
	writer.DecreaseIndent().WriteString("},").WriteLine()
	return nil
}

func (e *helperEmiter) convertPackagePathToLocalPath(pkgPath string) string {
	part := strings.Split(pkgPath, "/")
	return path.Join(e.root, strings.Join(part[1:], "/"))
}

func EmitInterfaceFromHelper(root string, target string, ometa *meta.ObjectMeta, kind string) error {
	module, err := getProjectModuleName(root)
	if err != nil {
		return err
	}
	emiter := &helperInterfaceEmiter{
		kind:      kind,
		root:      root,
		target:    target,
		ometa:     ometa,
		module:    module,
		writer:    util.NewTextWriter(),
		toPackage: fmt.Sprintf("%s/internal/srpc/service/%s", module, target),
		exportTo:  map[string]string{},
	}
	err = emiter.emit()
	if err != nil {
		return err
	}
	return nil
}

type helperInterfaceEmiter struct {
	kind      string
	root      string
	target    string
	ometa     *meta.ObjectMeta
	module    string
	writer    util.TextWriter
	toPackage string
	exportTo  map[string]string
	fmetas    []*meta.FieldMeta
	tmetas    []*meta.TypeMeta
}

func (e *helperInterfaceEmiter) emit() error {
	e.initMetas()
	e.redirectTypePackage()
	err := e.emitHeader()
	if err != nil {
		return err
	}
	err = e.emitTypeMetas()
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
	outPath := path.Join(e.root, "internal", "srpc", "service", e.target, toSnakeCase(e.ometa.Name)+"."+e.kind+".go")
	err = util.WriteGenerateFile(outPath, e.writer.Bytes(), e.root)
	if err != nil {
		return err
	}
	return nil
}

func (e *helperInterfaceEmiter) initMetas() {
	for _, f := range e.ometa.Functions {
		e.fmetas = append(e.fmetas, f.Parameters...)
		e.fmetas = append(e.fmetas, f.Results...)
	}
	for _, fmeta := range e.fmetas {
		e.tmetas = append(e.tmetas, fmeta.TypeMetas...)
	}
}

func (e *helperInterfaceEmiter) redirectTypePackage() {
	modelPackage := fmt.Sprintf("%s/internal/srpc/service/%s", e.module, e.target)
	for _, fmeta := range e.fmetas {
		for _, tmeta := range fmeta.TypeMetas {
			if isFromOtherService(tmeta.From) {
				e.exportTo[tmeta.Id] = fmt.Sprintf("%s/internal/srpc/service/%s", e.module, getImportPathExport(tmeta.From))
			} else {
				e.exportTo[tmeta.Id] = modelPackage
			}
		}
	}
}

var isFromOtherServiceReg = regexp.MustCompile(`internal\/srpc\/service\/(\w+)$`)

func isFromOtherService(path string) bool {
	return isFromOtherServiceReg.MatchString(path)
}

func (e *helperInterfaceEmiter) emitHeader() error {
	e.writer.WriteString(generatedHeader).WriteLine()
	e.writer.WriteString("package ", e.target).WriteLine()
	err := e.emitImports()
	if err != nil {
		return err
	}
	return nil
}

func (e *helperInterfaceEmiter) emitImports() error {
	collect := newImportCollect()
	var fmetas []*meta.FieldMeta
	for _, f := range e.ometa.Functions {
		fmetas = append(fmetas, f.Parameters...)
		fmetas = append(fmetas, f.Results...)
	}
	currentPackage := fmt.Sprintf("%s/internal/srpc/service/%s", e.module, e.target)
	for _, fmeta := range fmetas {
		for _, tmeta := range fmeta.TypeMetas {
			impo := tmeta.Import
			if impo != nil {
				collect.Set(getImportMetaExport(impo), impo.Path)
			}
			if len(tmeta.Code) > 0 {
				// 不是同一个包的代码片段需要import
				to := e.exportTo[tmeta.Id]
				if to != currentPackage {
					collect.Set(getImportPathExport(to), to)
				}
			}
		}
	}
	e.writer.WriteEmptyLine()
	collect.Emit(e.writer)
	return nil
}

func (e *helperInterfaceEmiter) emitTypeMetas() error {
	// 更改model
	models := map[*parse.Model]bool{}
	serviceDir := path.Join(e.root, "internal", "srpc", "service", e.target)
	if !gfile.Exists(serviceDir) {
		err := os.MkdirAll(serviceDir, os.ModePerm)
		if err != nil {
			return err
		}
	}
	for _, tmeta := range e.tmetas {
		if len(tmeta.Code) == 0 {
			continue
		}
		modelPackage := e.exportTo[tmeta.Id]
		filename := packagePathToFileName(e.root, modelPackage)
		modelFileName := path.Join(filename, "model.go")
		model, err := parse.ParseFileModel(modelFileName)
		if err != nil {
			return err
		}
		models[model] = true
		// 添加类型的import
		for _, id := range findAllTypeMetaIds(tmeta.Code) {
			cur := findTypeMetaForId(e.tmetas, id)
			if cur.Import != nil {
				model.AddImport(getImportMetaExport(cur.Import), cur.Import.Path)
			}
			if e.exportTo[id] != modelPackage {
				model.AddImport(getImportPathExport(e.exportTo[id]), e.exportTo[id])
			}
		}
		// 写入类型的代码
		model.AddType(tmeta.Name, &parse.ModelType{
			Raw:     nil,
			Content: []byte(e.replacePseudocodePart(modelPackage, tmeta.Code)),
		})

	}
	// 写出 models
	for model := range models {
		filename := model.GetFileName()
		err := emitModel(model, filename, e.root)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *helperInterfaceEmiter) emitBody() error {
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

func (e *helperInterfaceEmiter) emitFunction(fmeta *meta.FunctionMeta) error {
	writer := e.writer
	if e.kind == "listen" {
		writer.WriteString("On", fmeta.Name, "(fun func(")
	} else {
		writer.WriteString(fmeta.Name, "(")
	}
	for i, p := range fmeta.Parameters {
		if i != 0 {
			writer.WriteString(", ")
		}
		writer.WriteString(p.Name, " ", e.replacePseudocodePart(e.toPackage, p.Type))
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
		writer.WriteString(e.replacePseudocodePart(e.toPackage, r.Type))
	}
	if len(fmeta.Results) > 0 {
		writer.WriteString(")")
	}
	if e.kind == "listen" {
		writer.WriteString(")")
	}
	writer.WriteLine()
	return nil
}

func (e *helperInterfaceEmiter) replacePseudocodePart(pkg string, content string) string {
	return replacePseudocodePart(replacePseudocodePartInput{
		content: content,
		tmetas:  e.tmetas,
		getExportTo: func(tmetaId string) string {
			return e.exportTo[tmetaId]
		},
		currentPackage: pkg,
	})
}
