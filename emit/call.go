package emit

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sr/parse"
	"strconv"
	"strings"
)

func EmitCall(root string) error {
	// 取项目模块名
	module, err := getProjectModuleName(root)
	if err != nil {
		return err
	}
	// 获取待处理的Go目录
	dir := path.Join(root, "internal", "srpc", "call")
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
		target := base
		astFile, err := parse.ParseFile(filename)
		if err != nil {
			return err
		}
		for _, it := range astFile.InterfaceTypes {
			interfaceTypes = append(interfaceTypes, it)
			err = emitInterface(writer, target, it, false)
			if err != nil {
				return err
			}
		}
	}
	// 内容不生成文件
	if len(interfaceTypes) == 0 {
		return nil
	}

	importMap := map[string]string{} // [name]path
	importMap["context"] = "context"
	importMap["json"] = "encoding/json"
	importMap["srpc"] = "github.com/aundis/srpc"
	importMap["service"] = module + "/internal/service"
	for _, it := range interfaceTypes {
		fields := getInterfaceFields(it)
		for _, field := range fields {
			expr := field.Type
			if len(expr) == 0 {
				continue
			}
			if !isUsePackage(expr) {
				continue
			}
			name := getPackageName(expr)
			imp := resolveImport(it.Parent, name)
			if imp == nil {
				return formatError(it.Parent.FileSet, field.Pos, "无法找到引用的模块"+name)
			}
			if len(importMap[imp.Export]) > 0 && imp.Path != importMap[imp.Export] {
				fmt.Printf("警告: 模块%s存在不同的导入路径 %s, %s\n", name, imp.Path, importMap[imp.Export])
			}
			importMap[imp.Export] = imp.Path
		}
	}

	headerWriter := newTextWriter()
	headerWriter.WriteString("package " + base)
	headerWriter.WriteLine()
	for name, path := range importMap {
		if stringEndOf(path, name) {
			headerWriter.WriteString(`import "` + path + `"`)
		} else {
			headerWriter.WriteString("import " + name + ` "` + path + `"`)
		}
		headerWriter.WriteLine()
	}

	// 写出到文件
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()
	// 先写入头部信息
	_, err = out.Write(headerWriter.Bytes())
	if err != nil {
		return err
	}
	// 再写入内容
	_, err = out.Write(writer.Bytes())
	if err != nil {
		return err
	}
	out.Close()
	// 对生成的go文件进行格式化
	err = command("gofmt", "-w", outPath)
	if err != nil {
		return err
	}
	return nil
}

func getInterfaceFields(interfaceType *parse.InterfaceType) []*parse.Field {
	var result []*parse.Field
	for _, fun := range interfaceType.Functions {
		result = append(result, fun.Params...)
		result = append(result, fun.Results...)
	}
	return result
}

func emitInterface(writer TextWriter, target string, it *parse.InterfaceType, signal bool) error {
	// 👉 首先生成接口的结构体
	// 接口的名称需要I开头
	if string(it.Name[0]) != "I" {
		return formatError(it.Parent.FileSet, it.Pos, "接口名称必须以I开头")
	}
	structName := "c" + it.Name[1:]
	writer.WriteString("type ")
	writer.WriteString(structName)
	writer.WriteString(" struct {}")
	writer.WriteLine()
	// 写出变量
	writer.WriteString("var ")
	writer.WriteString(firstUpper(it.Name[1:]))
	writer.WriteString(" ")
	writer.WriteString(it.Name)
	writer.WriteString(" = ")
	writer.WriteString("&" + structName + "{}")
	writer.WriteLine()

	for _, fun := range it.Functions {
		// 👉 先生成返回类型的结构体, 如果有返回值的话
		responseStructName := firstLower(fun.Name) + "Response"
		if len(fun.Results) > 1 {
			writer.WriteString("type ")
			writer.WriteString(responseStructName)
			writer.WriteString(" struct {")
			writer.IncreaseIndent()
			writer.WriteLine()
			for i, r := range fun.Results {
				if i == len(fun.Results)-1 {
					continue
				}
				name := "r" + strconv.Itoa(i+1)
				writer.WriteString(firstUpper(name))
				writer.WriteString(" ")
				writer.WriteString(r.Type)
				writer.WriteString(" `json:\"")
				writer.WriteString(name)
				writer.WriteString("\"`")
				writer.WriteLine()
			}
			writer.WriteLine()
			writer.DecreaseIndent()
			writer.WriteString("}")
			writer.WriteLine()
		}

		// writer.WriteString(fmt.Sprintf("func (c *%s) %s (", structName, m.Name))
		writer.WriteString("func (c *")
		writer.WriteString(structName)
		writer.WriteString(") ")
		writer.WriteString(fun.Name)
		writer.WriteString(" (")
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
			writer.WriteString(" ")
			writer.WriteString(p.Type)

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
			writer.WriteString(" ")
			writer.WriteString(r.Type)
		}
		writer.WriteString(")")
		writer.WriteString(" {")
		writer.WriteLine()
		writer.IncreaseIndent()
		// 方法体内容
		// 	data, err := gjson.Marshal(g.Map{
		// 		"a": a,
		// 		"b": b,
		// 	})
		writer.WriteString("data, err := json.Marshal(map[string]interface{}{")
		writer.WriteLine()
		writer.IncreaseIndent()
		if len(fun.Params) > 1 {
			for i := range fun.Params {
				if i == 0 {
					continue
				}
				name := "p" + strconv.Itoa(i)
				writer.WriteString(`"`)
				writer.WriteString(name)
				writer.WriteString(`": `)
				writer.WriteString(name)
				writer.WriteString(",")
				writer.WriteLine()
			}
		}
		writer.DecreaseIndent()
		writer.WriteLine()
		writer.WriteString("})")
		writer.WriteLine()
		// 	if err != nil {
		// 		return 0, 0, err
		// 	}
		writer.WriteString("if err != nil {")
		writer.WriteLine()
		writer.IncreaseIndent()
		writer.WriteString("return")
		writer.DecreaseIndent()
		writer.WriteLine()
		writer.WriteString("}")
		writer.WriteLine()

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
		writer.WriteString("service.Srpc().Request(ctx, srpc.RequestData {")
		writer.IncreaseIndent()
		writer.WriteLine()
		if signal {
			writer.WriteString("Mark: srpc.EmitMark,")
		} else {
			writer.WriteString("Mark: srpc.CallMark,")
		}
		writer.WriteLine()
		writer.WriteString(`Target: "` + target + `",`)
		writer.WriteLine()
		writer.WriteString(`Action: "`)
		writer.WriteString(it.Name[1:])
		writer.WriteString(".")
		writer.WriteString(fun.Name)
		writer.WriteString(`",`)
		writer.WriteLine()
		writer.WriteString("Data:   data,")
		writer.WriteLine()
		writer.DecreaseIndent()
		writer.WriteString("})")
		writer.WriteLine()
		writer.WriteString("if err != nil {")
		writer.WriteLine()
		writer.IncreaseIndent()
		writer.WriteString("return")
		writer.DecreaseIndent()
		writer.WriteLine()
		writer.WriteString("}")
		writer.WriteLine()

		// 👉 如果没有返回值, 可以直接退出了
		if len(fun.Results) > 1 {
			//  var rsp *xxxResponse
			// 	jsn, err := json.Unmarshal(res, &rsp)
			// 	if err != nil {
			// 		return 0, 0, err
			// 	}
			writer.WriteString("var rsp *")
			writer.WriteString(responseStructName)
			writer.WriteLine()
			writer.WriteString("err = json.Unmarshal(res, &rsp)")
			writer.WriteLine()
			writer.WriteString("if err != nil {")
			writer.WriteLine()
			writer.IncreaseIndent()
			writer.WriteString("return")
			writer.DecreaseIndent()
			writer.WriteLine()
			writer.WriteString("}")
			writer.WriteLine()
			for i := range fun.Results {
				if i == len(fun.Results)-1 {
					continue
				}

				name := "r" + strconv.Itoa(i+1)
				writer.WriteString(name)
				writer.WriteString(" = ")
				writer.WriteString("rsp.")
				writer.WriteString(firstUpper(name))
				writer.WriteLine()
			}
		}

		writer.WriteString("return")
		writer.DecreaseIndent()
		writer.WriteLine()
		writer.WriteString("}")
		writer.WriteLine()
	}

	return nil
}

func getProjectModuleName(dir string) (string, error) {
	fileName := path.Join(dir, "go.mod")
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return "", nil
	}
	index := bytes.IndexByte(data, '\n')
	if index <= 0 {
		index = len(data)
	}
	firstLine := string(data[:index])
	firstLine = strings.ReplaceAll(firstLine, "\r", "")
	firstLine = strings.ReplaceAll(firstLine, "\n", "")
	firstLine = strings.ReplaceAll(firstLine, "module ", "")
	firstLine = strings.TrimSpace(firstLine)
	if len(firstLine) == 0 {
		return "", errors.New("get project module name error")
	}
	return firstLine, nil
}
