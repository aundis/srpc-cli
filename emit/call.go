package emit

import (
	"fmt"
	"path"
	"sr/parse"
	"sr/util"
	"strconv"

	"github.com/aundis/meta"
)

func EmitCall(root string) error {
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
		err = emitCallDir(root, module, dir)
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
		has, err := hasGoFile(path.Join(dir, "call"))
		if err != nil {
			return err
		}
		if has {
			writer.WriteString("import _ \"", module, "/internal/srpc/service/", base, `/call"`).WriteLine()
		}
	}
	err = util.WriteGenerateFile(path.Join(root, "internal", "srpc", "call.go"), writer.Bytes(), root)
	if err != nil {
		return err
	}
	return nil
}

func emitCallDir(root, module string, dir string) error {
	base := path.Base(dir)
	// 删除历史生成的目录
	outPath := path.Join(dir, "call")
	err := util.RemoveGenerateFiles(outPath)
	if err != nil {
		return err
	}
	// if gfile.Exists(outPath) {
	// 	err := gfile.Remove(outPath)
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	// 获取待处理的Go文件
	files, err := listFile(dir)
	if err != nil {
		return err
	}
	var goFiles []string
	for _, v := range files {
		if util.StringEndOf(v, ".call.go") {
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
			if !(len(it.Name) > 0 && it.Name[0] == 'I') {
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
		err = emitCallStruct(root, module, target, it)
		if err != nil {
			return err
		}
	}
	return nil
}

func emitCallStruct(root, module, target string, it *parse.InterfaceType) error {
	e := &callStructEmiter{
		writer: util.NewTextWriter(),
		root:   root,
		module: module,
		target: target,
		resolver: &typeResolver{
			module:   module,
			root:     root,
			resolved: map[string]*meta.TypeMeta{},
		},
		exportTo: fmt.Sprintf("%s/internal/srpc/service/%s/call", module, target),
		it:       it,
	}
	return e.emit()
}

type callStructEmiter struct {
	root     string
	module   string
	target   string
	exportTo string
	resolver *typeResolver
	it       *parse.InterfaceType
	writer   util.TextWriter
}

func (e *callStructEmiter) emit() error {
	err := e.emitHeader()
	if err != nil {
		return err
	}
	err = e.emitBody()
	if err != nil {
		return err
	}

	outDir := path.Join(e.root, "internal", "srpc", "service", e.target, "call")
	err = ensureDirExist(outDir)
	if err != nil {
		return err
	}
	outPath := path.Join(e.root, "internal", "srpc", "service", e.target, "call", toSnakeCase(e.it.Name[1:])+".go")
	err = util.WriteGenerateFile(outPath, e.writer.Bytes(), e.root)
	if err != nil {
		return err
	}
	return nil
}

func (e *callStructEmiter) emitHeader() error {
	e.writer.WriteString(generatedHeader).WriteLine()
	e.writer.WriteString("package call").WriteLine()
	e.writer.WriteEmptyLine()
	err := e.emitImports()
	if err != nil {
		return err
	}
	return nil
}

func (e *callStructEmiter) emitImports() error {
	collect := newImportCollect()
	collect.Set("context", "context")
	collect.Set("json", "encoding/json")
	collect.Set("srpc", "github.com/aundis/srpc")
	collect.Set("service", e.module+"/internal/service")
	collect.Set(e.target, e.module+"/internal/srpc/service/"+e.target)
	err := resolveInterfaceImports(e.it, collect, e.exportTo, e.module, e.root)
	if err != nil {
		return err
	}
	collect.Emit(e.writer)
	return nil
}

func (e *callStructEmiter) emitBody() error {
	it := e.it
	writer := e.writer
	// 解析重定向所有的Field
	fResolver := newFieldResolver(e.root, e.module, e.exportTo)
	fields := getInterfaceFields(it)
	for _, v := range fields {
		err := fResolver.resolve(v)
		if err != nil {
			return err
		}
	}
	// 注册
	writer.WriteEmptyLine()
	writer.WriteString("func init() {").WriteLine().IncreaseIndent()
	writer.WriteString(e.target, ".", "Register", e.it.Name[1:], "(&", "c"+it.Name[1:], "{})").WriteLine()
	writer.DecreaseIndent().WriteString("}").WriteLine()
	// 首先生成接口的结构体
	// 接口的名称需要I开头
	if string(it.Name[0]) != "I" {
		return formatError(it.Parent.FileSet, it.Pos, "interface name must start with an \"I\"", e.root)
	}
	structName := "c" + it.Name[1:]
	writer.WriteEmptyLine()
	writer.WriteString("type ", structName, " struct {}").WriteLine()
	for _, fun := range it.Functions {
		// 先生成返回类型的结构体, 如果有返回值的话
		responseStructName := firstLower(e.target) + it.Name[1:] + fun.Name + "Response"
		if len(fun.Results) > 1 {
			writer.WriteEmptyLine()
			writer.WriteString("type ", responseStructName, " struct {").WriteLine().IncreaseIndent()
			for i, r := range fun.Results {
				if i == len(fun.Results)-1 {
					continue
				}
				name := "r" + strconv.Itoa(i+1)
				writer.WriteString(firstUpper(name), " ", fResolver.getResolvedType(r), " `json:\"", name, "\"`").WriteLine()
			}
			writer.DecreaseIndent().WriteString("}").WriteLine()
		}

		// writer.WriteString(fmt.Sprintf("func (c *%s) %s (", structName, m.Name))
		writer.WriteEmptyLine()
		writer.WriteString("func (c *", structName, ") ", fun.Name, " (")
		// 写参数
		for i, p := range fun.Params {
			// 首参数校验
			if i == 0 {
				if p.Name != "ctx" {
					return formatError(it.Parent.FileSet, p.Pos, "first param name must is ctx", e.root)
				}
				if p.Type != "context.Context" {
					return formatError(it.Parent.FileSet, p.Pos, "first param type must is context.Context", e.root)
				}
			}
			if i != 0 {
				writer.WriteString(", ")
			}
			if i == 0 {
				writer.WriteString(p.Name)
			} else {
				name := "p" + strconv.Itoa(i)
				writer.WriteString(name)
			}
			writer.WriteString(" ", fResolver.getResolvedType(p))
		}
		writer.WriteString(")")
		// 写返回值
		if len(fun.Results) == 0 {
			return formatError(it.Parent.FileSet, fun.Pos, "method must provide a return value of type error", e.root)
		}
		writer.WriteString(" (")
		for i, r := range fun.Results {
			// 校验最后一个返回类型
			if i == len(fun.Results)-1 {
				if r.Type != "error" {
					return formatError(it.Parent.FileSet, fun.Pos, "method last return value must be error", e.root)
				}
			}
			if i != 0 {
				writer.WriteString(", ")
			}
			// 统一设置为命名返回值
			if r.Type == "error" {
				writer.WriteString("err")
			} else {
				writer.WriteString("r" + strconv.Itoa(i+1))
			}
			writer.WriteString(" ", fResolver.getResolvedType(r))
		}
		writer.WriteString(")", " {").WriteLine().IncreaseIndent()
		// 方法体内容
		// 	data, err := gjson.Marshal(g.Map{
		// 		"a": a,
		// 		"b": b,
		// 	})
		writer.WriteString("data, err := json.Marshal(map[string]interface{}{").WriteLine().IncreaseIndent()
		if len(fun.Params) > 1 {
			for i := range fun.Params {
				if i == 0 {
					continue
				}
				name := "p" + strconv.Itoa(i)
				writer.WriteString(`"`, name, `": `, name, ",").WriteLine()
			}
		}
		writer.DecreaseIndent().WriteString("})").WriteLine()
		// 	if err != nil {
		// 		return 0, 0, err
		// 	}
		writer.WriteString("if err != nil {").WriteLine().IncreaseIndent()
		writer.WriteString("return").WriteLine()
		writer.DecreaseIndent().WriteString("}").WriteLine()
		// 	res, err := c.Request(ctx, srpc.RequestData{
		// 		Mark:   srpc.CallMark,
		// 		Target: "xxx",
		// 		Action: "Hello",
		// 		Data:   data,
		// 	})
		// 	if err != nil {
		// 		return 0, 0, err
		// 	}
		if len(fun.Results) > 1 {
			writer.WriteString("res, err := ")
		} else {
			writer.WriteString("_, err = ")
		}
		writer.WriteString("service.Srpc().Request(ctx, srpc.RequestData {").IncreaseIndent().WriteLine()
		writer.WriteString("Mark: srpc.CallMark,").WriteLine()
		writer.WriteString(`Target: "` + e.target + `",`).WriteLine()
		writer.WriteString(`Action: "`, it.Name[1:], ".", fun.Name, `",`).WriteLine()
		writer.WriteString("Data:   data,").WriteLine()
		writer.DecreaseIndent().WriteString("})").WriteLine()
		writer.WriteString("if err != nil {").WriteLine().IncreaseIndent()
		writer.WriteString("return").WriteLine()
		writer.DecreaseIndent().WriteString("}").WriteLine()
		// 如果没有返回值, 可以直接退出了
		if len(fun.Results) > 1 {
			//  var rsp *xxxResponse
			// 	jsn, err := json.Unmarshal(res, &rsp)
			// 	if err != nil {
			// 		return 0, 0, err
			// 	}
			writer.WriteString("var rsp *").WriteString(responseStructName).WriteLine()
			writer.WriteString("err = json.Unmarshal(res, &rsp)").WriteLine()
			writer.WriteString("if err != nil {").WriteLine().IncreaseIndent()
			writer.WriteString("return").WriteLine()
			writer.DecreaseIndent().WriteString("}").WriteLine()
			for i := range fun.Results {
				if i == len(fun.Results)-1 {
					continue
				}
				name := "r" + strconv.Itoa(i+1)
				writer.WriteString(name, " = ", "rsp.", firstUpper(name)).WriteLine()
			}
		}
		writer.WriteString("return").WriteLine()
		writer.DecreaseIndent().WriteString("}").WriteLine()
	}
	return nil
}

// func (e *callStructEmiter) formatType(tpe string) string {
// 	reg := regexp.MustCompile(`\b(\.?[A-Z]\w*\.?)\b`)
// 	return reg.ReplaceAllStringFunc(tpe, func(s string) string {
// 		if s[0] != '.' && s[len(s)-1] != '.' {
// 			return e.target + "." + s
// 		}
// 		return s
// 	})
// }
