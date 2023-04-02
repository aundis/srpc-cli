package emit

import (
	"go/ast"
	"path"
	"regexp"
	"sr/parse"
	"sr/util"
	"strconv"
)

func EmitListen(root string) error {
	// 取项目模块名
	module, err := getProjectModuleName(root)
	if err != nil {
		return err
	}
	// 获取待处理的Go目录
	dir := path.Join(root, "internal", "srpc", "service")
	err = ensureDirExist(dir)
	if err != nil {
		return err
	}
	dirs, err := listDir(dir)
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		err = emitListenDir(root, module, dir)
		if err != nil {
			return err
		}
	}
	// 生成初始化文件
	writer := util.NewTextWriter()
	writer.WriteString(generatedHeader).WriteLine()
	writer.WriteString("package srpc").WriteString().WriteLine()
	writer.WriteEmptyLine()
	for _, dir := range dirs {
		base := path.Base(dir)
		has, err := hasGoFile(path.Join(dir, "listen"))
		if err != nil {
			return err
		}
		if has {
			writer.WriteString("import _ \"", module, "/internal/srpc/service/", base, `/listen"`).WriteLine()
		}
	}
	err = util.WriteGenerateFile(path.Join(root, "internal", "srpc", "listen.go"), writer.Bytes(), root)
	if err != nil {
		return err
	}
	return nil
}

func emitListenDir(root, module string, dir string) error {
	base := path.Base(dir)
	// 删除历史生成的文件
	err := util.RemoveGenerateFiles(path.Join(dir, "listen"))
	if err != nil {
		return err
	}
	// 获取待处理的Go文件
	files, err := listFile(dir)
	if err != nil {
		return err
	}
	var goFiles []string
	for _, v := range files {
		if util.StringEndOf(v, ".listen.go") {
			goFiles = append(goFiles, v)
		}
	}
	if len(goFiles) == 0 {
		return nil
	}
	// 拿到所有的接口类型
	var interfaceTypes []*parse.InterfaceType
	for _, filename := range goFiles {
		astFile, err := parse.ParseFile(filename)
		if err != nil {
			return err
		}
		for _, it := range astFile.InterfaceTypes {
			// 只处理I开头的interface
			if len(it.Name) == 0 {
				continue
			}
			if it.Name[0] != 'I' {
				continue
			}
			interfaceTypes = append(interfaceTypes, it)
		}
	}
	// 内容不生成文件
	if len(interfaceTypes) == 0 {
		return nil
	}
	// 生成
	for _, it := range interfaceTypes {
		target := base
		err = emitListenStruct(root, module, target, it)
		if err != nil {
			return err
		}
	}
	return nil
}

func emitListenStruct(root, module, target string, it *parse.InterfaceType) error {
	e := &listenStructEmiter{
		writer: util.NewTextWriter(),
		root:   root,
		module: module,
		target: target,
		it:     it,
	}
	return e.emit()
}

type listenStructEmiter struct {
	root   string
	module string
	target string
	it     *parse.InterfaceType
	writer util.TextWriter
}

func (e *listenStructEmiter) emit() error {
	err := e.emitHeader()
	if err != nil {
		return err
	}
	err = e.emitBody()
	if err != nil {
		return err
	}

	outDir := path.Join(e.root, "internal", "srpc", "service", e.target, "listen")
	err = ensureDirExist(outDir)
	if err != nil {
		return err
	}
	outPath := path.Join(e.root, "internal", "srpc", "service", e.target, "listen", toSnakeCase(e.it.Name[1:])+".go")
	err = util.WriteGenerateFile(outPath, e.writer.Bytes(), e.root)
	if err != nil {
		return err
	}
	return nil
}

func (e *listenStructEmiter) emitHeader() error {
	e.writer.WriteString(generatedHeader).WriteLine()
	e.writer.WriteString("package listen").WriteLine()
	e.writer.WriteEmptyLine()
	err := e.emitImports()
	if err != nil {
		return err
	}
	return nil
}

func (e *listenStructEmiter) emitImports() error {
	collect := newImportCollect()
	collect.Set("context", "context")
	collect.Set("json", "encoding/json")
	// collect.Set("srpc", "github.com/aundis/srpc")
	// collect.Set("service", e.module+"/internal/service")
	collect.Set("manager", e.module+"/internal/srpc/manager")
	collect.Set("garray", "github.com/gogf/gf/v2/container/garray")
	collect.Set(e.target, e.module+"/internal/srpc/service/"+e.target)
	err := resolveInterfaceImports(e.it, collect, e.root)
	if err != nil {
		return err
	}
	collect.Emit(e.writer)
	return nil
}

func (e *listenStructEmiter) emitBody() error {
	// 检查函数签名是否合法
	var paramAndResultArr []paramAndResult
	for _, fun := range e.it.Functions {
		if len(fun.Name) < 2 || string(fun.Name[:2]) != "On" {
			return formatError(e.it.Parent.FileSet, fun.Pos, "listen interface function name must start with On", e.root)
		}
		if len(fun.Params) != 1 {
			return formatError(e.it.Parent.FileSet, fun.Pos, "listen interface function params count must be 1", e.root)
		}
		if !parse.IsFuncType(fun.Params[0].TypeRaw) {
			return formatError(e.it.Parent.FileSet, fun.Params[0].Pos, "listen interface function first params type must be function type", e.root)
		}
		funcType := fun.Params[0].TypeRaw.(*ast.FuncType)
		params, results := parse.ParseFuncType(e.it.Parent.Content, funcType)
		if len(params) == 0 {
			return formatError(e.it.Parent.FileSet, funcType.Pos(), "the function type has at least one parameter", e.root)
		}
		if params[0].Type != "context.Context" {
			return formatError(e.it.Parent.FileSet, funcType.Pos(), "the function type first param type must be context.Context", e.root)
		}
		if len(results) == 0 {
			return formatError(e.it.Parent.FileSet, funcType.Pos(), "the function type must has a return type", e.root)
		}
		if results[0].Type != "error" {
			return formatError(e.it.Parent.FileSet, results[0].Pos, "the function type first return type must be error", e.root)
		}
		paramAndResultArr = append(paramAndResultArr, paramAndResult{
			params:  params,
			results: results,
		})
	}

	writer := e.writer
	objectName := e.it.Name[1:]
	writer.WriteEmptyLine()
	writer.WriteString("func init() {").WriteLine().IncreaseIndent()
	for _, fun := range e.it.Functions {
		orgFunctionName := string(fun.Name[2:])
		action := e.target + "@" + e.it.Name[1:] + "." + orgFunctionName
		writer.WriteString(`manager.AddListenName("`, action, `")`).WriteLine()
	}
	writer.WriteEmptyLine()
	writer.WriteString("listen := &l", e.it.Name[1:], "{}").WriteLine()
	// abc.RegisterBox(listen)
	writer.WriteString(e.target, ".Register", objectName, "(listen)").WriteLine()
	for i, fun := range e.it.Functions {
		orgFunctionName := string(fun.Name[2:])
		params := paramAndResultArr[i].params
		// results := paramAndResultArr[i].results

		action := e.target + "@" + e.it.Name[1:] + "." + orgFunctionName
		writer.WriteString(`manager.AddController("`, action, `", func(ctx context.Context, req []byte) (res interface{}, err error) {`).WriteLine().IncreaseIndent()
		if len(params) > 1 {
			reqStructName := firstLower(e.it.Name[1:]) + orgFunctionName + `Request`
			writer.WriteString(`var params *`, reqStructName).WriteLine()
			writer.WriteString(`err = json.Unmarshal(req, &params)`).WriteLine()
			writer.WriteString(`if err != nil {`).WriteLine().IncreaseIndent()
			writer.WriteString(`return`).WriteLine()
			writer.DecreaseIndent().WriteString("}").WriteLine()
		}
		writer.WriteString("err = ", "listen.", firstLower(orgFunctionName), "(ctx")
		for i := range params {
			if i == 0 {
				continue
			}
			writer.WriteString(", ")
			writer.WriteString("params.P", strconv.Itoa(i))
		}
		writer.WriteString(")").WriteLine()
		writer.WriteString(`if err != nil {`).WriteLine().IncreaseIndent()
		writer.WriteString(`return`).WriteLine()
		writer.DecreaseIndent().WriteString("}").WriteLine()
		writer.WriteString("res = map[string]interface{}{}").WriteLine()
		writer.WriteString("return").WriteLine()
		writer.DecreaseIndent().WriteString("})").WriteLine()
	}
	// writer.WriteString("service.Register", ometa.Name, "(&s", ometa.Name, "{})").WriteLine()
	writer.DecreaseIndent().WriteString("}").WriteLine()

	for i, fun := range e.it.Functions {
		orgFunctionName := string(fun.Name[2:])
		params := paramAndResultArr[i].params
		// results := paramAndResultArr[i].results
		if len(params) <= 1 {
			continue
		}
		reqStructName := firstLower(e.it.Name[1:]) + orgFunctionName + `Request`
		writer.WriteEmptyLine()
		writer.WriteString("type ", reqStructName, " struct {").WriteLine().IncreaseIndent()
		for i, param := range params {
			if i == 0 {
				continue
			}
			// P1 int `json:"p1"`
			name := "P" + strconv.Itoa(i)
			writer.WriteString(name, " ", e.formatType(param.Type), " `json:\"", firstLower(name), "\"`").WriteLine()
		}
		writer.DecreaseIndent().WriteString("}").WriteLine()
	}

	// var boxBoomFuncs = garray.New(true)
	// var boxSelFuncs = garray.New(true)
	writer.WriteEmptyLine()
	for _, fun := range e.it.Functions {
		orgFunctionName := string(fun.Name[2:])
		writer.WriteString("var ", firstLower(objectName), orgFunctionName, "Funcs = garray.New(true)").WriteLine()
	}

	// type lBox struct { }
	writer.WriteString("type l", objectName, " struct {}").WriteLine()

	// func (l *lBox) OnBoom(fun func(ctx context.Context, x int, y int) error) {
	// 	boxBoomFuncs.Append(fun)
	// }
	for i, fun := range e.it.Functions {
		orgFunctionName := string(fun.Name[2:])
		params := paramAndResultArr[i].params
		arrayName := firstLower(objectName) + orgFunctionName + "Funcs"
		// results := paramAndResultArr[i].results
		writer.WriteEmptyLine()
		writer.WriteString("func (l *l", objectName, ") On", orgFunctionName, "(fun func(")
		for i, p := range params {
			if i > 0 {
				writer.WriteString(", ")
			}
			writer.WriteString(p.Name, " ", e.formatType(p.Type))
		}
		writer.WriteString(") error) {").WriteLine().IncreaseIndent()
		writer.WriteString(arrayName, ".Append(fun)").WriteLine()
		writer.DecreaseIndent().WriteString("}").WriteLine()
	}

	for i, fun := range e.it.Functions {
		orgFunctionName := string(fun.Name[2:])
		params := paramAndResultArr[i].params
		arrayName := firstLower(objectName) + orgFunctionName + "Funcs"
		writer.WriteEmptyLine()
		writer.WriteString("func (l *l", objectName, ") ", firstLower(orgFunctionName), "(")
		for i, p := range params {
			if i > 0 {
				writer.WriteString(", ")
			}
			writer.WriteString(p.Name, " ", e.formatType(p.Type))
		}
		writer.WriteString(") (err error) {").WriteLine().IncreaseIndent()
		writer.WriteString("if ", arrayName, ".Len() == 0 {").WriteLine().IncreaseIndent()
		writer.WriteString("return").WriteLine()
		writer.DecreaseIndent().WriteString("}").WriteLine()
		// boxSelFuncs.RLockFunc(func(array []interface{}) {
		writer.WriteString(arrayName, ".RLockFunc(func(array []interface{}) {").WriteLine().IncreaseIndent()
		// for _, v := range array {
		writer.WriteString("for _, v := range array {").WriteLine().IncreaseIndent()
		// fun := v.(func(context.Context, *abc.Apple) error)
		writer.WriteString("fun := v.(func(")
		for i, p := range params {
			if i > 0 {
				writer.WriteString(", ")
			}
			writer.WriteString(e.formatType(p.Type))
		}
		writer.WriteString(") error)").WriteLine()
		// err = fun(ctx, apple1)
		writer.WriteString("err = fun(")
		for i, p := range params {
			if i > 0 {
				writer.WriteString(", ")
			}
			writer.WriteString(p.Name)
		}
		writer.WriteString(")").WriteLine()
		// if err != nil {
		// 	return
		// }
		writer.WriteString("if err != nil {").WriteLine().IncreaseIndent()
		writer.WriteString("return").WriteLine()
		writer.DecreaseIndent().WriteString("}").WriteLine()
		// for
		writer.DecreaseIndent().WriteString("}").WriteLine()
		// RLockFunc
		writer.DecreaseIndent().WriteString("})").WriteLine()
		// if err != nil {
		// 	return
		// }
		writer.WriteString("if err != nil {").WriteLine().IncreaseIndent()
		writer.WriteString("return").WriteLine()
		writer.DecreaseIndent().WriteString("}").WriteLine()
		writer.WriteString("return").WriteLine()
		// method
		writer.DecreaseIndent().WriteString("}").WriteLine()
	}

	return nil
}

func (e *listenStructEmiter) formatType(tpe string) string {
	reg := regexp.MustCompile(`\b(\.?[A-Z]\w*\.?)\b`)
	return reg.ReplaceAllStringFunc(tpe, func(s string) string {
		if s[0] != '.' && s[len(s)-1] != '.' {
			return e.target + "." + s
		}
		return s
	})
}
