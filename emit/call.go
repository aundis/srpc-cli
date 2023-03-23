package emit

import (
	"io/ioutil"
	"os"
	"path"
	"sr/parse"
	"strconv"
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
		err = emitCallDir(module, dir)
		if err != nil {
			return err
		}
	}
	return nil
}

func emitCallDir(module string, dir string) error {
	base := path.Base(dir)
	// 删除历史生成的文件
	outPath := path.Join(dir, "generate.go")
	exist, err := fileExist(outPath)
	if err != nil {
		return err
	}
	if exist {
		err = os.Remove(outPath)
		if err != nil {
			return err
		}
	}
	//
	writer := newTextWriter()
	// 获取待处理的Go文件
	files, err := listFile(dir)
	if err != nil {
		return err
	}
	var goFiles []string
	for _, v := range files {
		if path.Ext(v) == ".go" {
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
	// 生成头部信息
	writer.WriteString("package ", base).WriteLine()
	collect := newImportCollect()
	collect.Set("context", "context")
	collect.Set("json", "encoding/json")
	collect.Set("srpc", "github.com/aundis/srpc")
	collect.Set("service", module+"/internal/service")
	for _, it := range interfaceTypes {
		err = resolveInterfaceImports(it, collect)
		if err != nil {
			return err
		}
	}
	collect.Emit(writer)
	// 生成内容
	for _, it := range interfaceTypes {
		target := base
		err = emitInterface(writer, target, it, false)
		if err != nil {
			return err
		}
	}
	err = ioutil.WriteFile(outPath, writer.Bytes(), os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func emitInterface(writer TextWriter, target string, it *parse.InterfaceType, signal bool) error {
	// 👉 首先生成接口的结构体
	// 接口的名称需要I开头
	if string(it.Name[0]) != "I" {
		return formatError(it.Parent.FileSet, it.Pos, "接口名称必须以I开头")
	}
	structName := "c" + it.Name[1:]
	writer.WriteString("type ", structName, " struct {}").WriteLine()
	// 写出变量
	writer.WriteString("var ", firstUpper(it.Name[1:]), " ", it.Name, " = ", "&"+structName+"{}").WriteLine()
	for _, fun := range it.Functions {
		// 👉 先生成返回类型的结构体, 如果有返回值的话
		responseStructName := firstLower(fun.Name) + "Response"
		if len(fun.Results) > 1 {
			writer.WriteString("type ", responseStructName, " struct {").WriteLine().IncreaseIndent()
			for i, r := range fun.Results {
				if i == len(fun.Results)-1 {
					continue
				}
				name := "r" + strconv.Itoa(i+1)
				writer.WriteString(firstUpper(name), " ", r.Type, " `json:\"", name, "\"`").WriteLine()
			}
			writer.DecreaseIndent().WriteString("}").WriteLine()
		}

		// writer.WriteString(fmt.Sprintf("func (c *%s) %s (", structName, m.Name))
		writer.WriteString("func (c *", structName, ") ", fun.Name, " (")
		// 👉 写参数
		for i, p := range fun.Params {
			// 首参数校验
			if i == 0 {
				if p.Name != "ctx" {
					return formatError(it.Parent.FileSet, p.Pos, "first param name must is ctx")
				}
				if p.Type != "context.Context" {
					return formatError(it.Parent.FileSet, p.Pos, "first param type must is context.Context")
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
			writer.WriteString(" ", p.Type)
		}
		writer.WriteString(")")
		// 👉 写返回值
		// 必须有返回值
		if len(fun.Results) == 0 {
			return formatError(it.Parent.FileSet, fun.Pos, "method must provide a return value of type error")
		}
		if signal && len(fun.Results) > 1 {
			return formatError(it.Parent.FileSet, fun.Pos, "signal method can only have one return value")
		}
		writer.WriteString(" (")
		for i, r := range fun.Results {
			// 校验最后一个返回类型
			if i == len(fun.Results)-1 {
				if r.Type != "error" {
					return formatError(it.Parent.FileSet, fun.Pos, "method last return value must be error")
				}
			}
			if i != 0 {
				writer.WriteString(", ")
			}
			// 👉 统一设置为命名返回值
			if r.Type == "error" {
				writer.WriteString("err")
			} else {
				writer.WriteString("r" + strconv.Itoa(i+1))
			}
			writer.WriteString(" ", r.Type)
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
		if signal {
			writer.WriteString("Mark: srpc.EmitMark,")
		} else {
			writer.WriteString("Mark: srpc.CallMark,")
		}
		writer.WriteLine()
		writer.WriteString(`Target: "` + target + `",`).WriteLine()
		writer.WriteString(`Action: "`, it.Name[1:], ".", fun.Name, `",`).WriteLine()
		writer.WriteString("Data:   data,").WriteLine()
		writer.DecreaseIndent().WriteString("})").WriteLine()
		writer.WriteString("if err != nil {").WriteLine().IncreaseIndent()
		writer.WriteString("return").WriteLine()
		writer.DecreaseIndent().WriteString("}").WriteLine()
		// 👉 如果没有返回值, 可以直接退出了
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
