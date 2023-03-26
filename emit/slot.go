package emit

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"path"
	"sr/parse"
	"strconv"
	"strings"
)

var mergeMapsFunc = `// overwriting duplicate keys, you should handle that if there is a need
func mergeMaps(maps ...map[string]srpc.ControllerHandle) map[string]srpc.ControllerHandle {
	result := make(map[string]srpc.ControllerHandle)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}
`

func EmitSlot(root string) error {
	// 取项目模块名
	module, err := getProjectModuleName(root)
	if err != nil {
		return err
	}
	e := &slotEmiter{
		root:   root,
		module: module,
		outDir: path.Join(root, "internal", "srpc", "slot"),
	}
	err = e.emit()
	if err != nil {
		return err
	}
	return nil
}

type slotEmiter struct {
	root          string
	module        string
	outDir        string
	targetStructs []*parse.StructType
}

func (e *slotEmiter) emit() error {
	dirs, err := listDir(path.Join(e.root, "internal", "logic"))
	if err != nil {
		return err
	}
	outDir := path.Join(e.root, "internal", "srpc", "slot")
	// 确保输出目录存在
	err = ensureDirExist(outDir)
	if err != nil {
		return err
	}
	// 清空输出目录下的Go文件
	err = removeDirFiles(outDir)
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		err := e.emitSlotDir(dir)
		if err != nil {
			return err
		}
	}
	// 生成 slot.go
	writer := newTextWriter()
	writer.WriteString("package slot").WriteLine()
	writer.WriteString("import \"github.com/aundis/srpc\"").WriteLine()
	writer.WriteString("import \"github.com/aundis/meta\"").WriteLine()
	writer.WriteString(mergeMapsFunc).WriteLine()
	// 合并所有的controller
	writer.WriteString("var Controllers = mergeMaps(").WriteLine().IncreaseIndent()
	for _, st := range e.targetStructs {
		name := firstLower(st.Name[1:])
		writer.WriteString(name+"Controller", ",").WriteLine()
	}
	writer.DecreaseIndent().WriteString(")").WriteLine()
	// 合并所有的helper
	writer.WriteString("var Helpers = []meta.ObjectMeta{").WriteLine().IncreaseIndent()
	for _, st := range e.targetStructs {
		if !isSlotStruct(st) {
			continue
		}
		name := firstLower(st.Name[1:])
		writer.WriteString(name+"Helper", ",").WriteLine()
	}
	writer.DecreaseIndent().WriteString("}").WriteLine()
	err = ioutil.WriteFile(path.Join(e.root, "internal", "srpc", "slot", "slot.go"), writer.Bytes(), fs.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func (e *slotEmiter) emitSlotDir(dir string) error {
	files, err := listFile(dir)
	if err != nil {
		return err
	}
	// 解析所有Go文件
	var astFiles []*parse.File
	for _, filename := range files {
		astFile, err := parse.ParseFile(filename)
		if err != nil {
			return err
		}
		astFiles = append(astFiles, astFile)
	}
	// 合并结构类型
	structs := parse.CombineStructTypes(astFiles)
	// 提取出 slot 和 listen
	for _, st := range structs {
		// 去掉无类型名称的结构体
		if len(st.Name) == 0 {
			continue
		}
		if isSlotStruct(st) {
			st.Functions = filterNoExport(st.Functions)
			if len(st.Functions) == 0 {
				continue
			}
			e.targetStructs = append(e.targetStructs, st)
			continue
		}
		if isListenStruct(st) {
			st.Functions = filterNoExport(st.Functions)
			if len(st.Functions) == 0 {
				continue
			}
			target := getListenTarget(st)
			if len(target) == 0 {
				return formatError(st.Parent.FileSet, st.Pos, "not set listen object")
			}
			e.targetStructs = append(e.targetStructs, st)
		}
	}
	// 无内容则不生成
	if len(e.targetStructs) == 0 {
		return nil
	}
	// 开始生成代码, 一个结构体对应一个文件
	for _, st := range e.targetStructs {
		writer := newTextWriter()
		writer.WriteString("package slot").WriteLine()
		// 处理 import
		collect := newImportCollect()
		collect.Set("srpc", "github.com/aundis/srpc")
		collect.Set("meta", "github.com/aundis/meta")
		collect.Set("service", e.module+"/internal/service")
		if structNeedImportJson(st) {
			collect.Set("json", "encoding/json")
		}
		err = resolveStructImports(st, collect)
		if err != nil {
			return err
		}
		collect.Emit(writer)
		// emit
		err = emitStruct(writer, st)
		if err != nil {
			return err
		}
		// helper
		err := emitSlotHelper(e.root, e.module, writer, st)
		if err != nil {
			return err
		}
		filename := path.Join(e.outDir, toSnakeCase(st.Name[1:])+".go")
		err = ioutil.WriteFile(filename, writer.Bytes(), fs.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func structNeedImportJson(st *parse.StructType) bool {
	for _, v := range st.Functions {
		if len(v.Params) > 1 {
			return true
		}
	}
	return false
}

func filterNoExport(list []*parse.Function) []*parse.Function {
	var result []*parse.Function
	for _, v := range list {
		if len(v.Name) == 0 {
			continue
		}
		if !(v.Name[0] >= 'A' && v.Name[0] <= 'Z') {
			continue
		}
		// 首个参数必须为 context.Context
		if len(v.Params) == 0 || v.Params[0].Type != "context.Context" {
			fmt.Println("warning: " + formatError(v.Parent.FileSet, v.Pos, "first paramater type not context.Context, ignore method "+v.Name).Error())
			continue
		}
		// 最后一个返回值必须为error
		if len(v.Results) == 0 || v.Results[len(v.Results)-1].Type != "error" {
			fmt.Println("warning: " + formatError(v.Parent.FileSet, v.Pos, "last return value type not error, ignore method "+v.Name).Error())
			continue
		}
		result = append(result, v)
	}
	return result
}

func emitStruct(writer TextWriter, st *parse.StructType) error {
	// 生成结构方法的参数结构体
	for _, f := range st.Functions {
		// 请求结构体
		// type ParamStruct struct {
		// 	P1 int `json:"a"`
		// 	P2 int `json:"B"`
		// }
		if len(f.Params) > 1 {
			reqStructName := firstLower(st.Name[1:]) + f.Name + "Request"
			writer.WriteString("type ", reqStructName, " struct {").WriteLine().IncreaseIndent()
			for i, p := range f.Params {
				if i == 0 {
					continue
				}
				fieldName := "p" + strconv.Itoa(i)
				// Name string `json:"name"`
				writer.WriteString(firstUpper(fieldName), " ", strings.ReplaceAll(p.Type, "...", "[]"), " `json:\"", fieldName, "\"`").WriteLine()
			}
			writer.DecreaseIndent().WriteString("}").WriteLine()
		}
	}

	contollerName := firstLower(st.Name[1:]) + "Controller"
	writer.WriteString("var ", contollerName, " = ", "map[string]srpc.ControllerHandle {").WriteLine().IncreaseIndent()
	// 这里面放请求方法

	for _, f := range st.Functions {
		writer.WriteString(`"`)
		if isListenStruct(st) {
			target := getListenTarget(st)
			writer.WriteString(target + "@" + st.Name[1:] + "." + f.Name)
		} else {
			writer.WriteString(st.Name[1:] + "." + f.Name)
		}
		writer.WriteString(`": `, "func (ctx context.Context, req []byte) (res interface{}, err error) {").WriteLine().IncreaseIndent()

		// 	var params *ParamStruct
		// 	err = json.Unmarshal(req, &params)
		// 	if err != nil {
		// 		return
		// 	}
		if len(f.Params) > 1 {
			reqStructName := firstLower(st.Name[1:]) + f.Name + "Request"
			writer.WriteString("var params *" + reqStructName).WriteLine()
			writer.WriteString("err = json.Unmarshal(req, &params)").WriteLine()
			writer.WriteString("if err != nil {").WriteLine().IncreaseIndent()
			writer.WriteString("return").WriteLine()
			writer.DecreaseIndent().WriteString("}").WriteLine()
		}

		// 	r1, r2, r3, err := target(ctx, params.P1, params.P2)
		// 	if err != nil {
		// 		return
		// 	}
		for i := range f.Results {
			if i == len(f.Results)-1 {
				continue
			}
			fieldName := "r" + strconv.Itoa(i+1)
			writer.WriteString(fieldName, ", ")
		}
		writer.WriteString("err")
		if len(f.Results) == 1 {
			writer.WriteString(" = ")
		} else {
			writer.WriteString(" := ")
		}
		// service.XXX().(ctx
		writer.WriteString("service.", st.Name[1:], "(). ", f.Name, "(ctx")
		paramIndex := 1
		if len(f.Params) > 1 {
			for i, v := range f.Params {
				if i == 0 {
					continue
				}
				writer.WriteString(", ")
				fieldName := "P" + strconv.Itoa(paramIndex)
				paramIndex++
				writer.WriteString("params." + fieldName)
				// 支持省略参数
				if strings.Contains(v.Type, "...") {
					writer.WriteString("...")
				}
			}
		}
		writer.WriteString(")").WriteLine()
		writer.WriteString("if err != nil {").WriteLine().IncreaseIndent()
		writer.WriteString("return").WriteLine()
		writer.DecreaseIndent().WriteString("}").WriteLine()

		// 	res = map[string]interface{} {
		// 		"r1": r1,
		// 		"r2": r2,
		// 		"r3": r3,
		// 	}
		writer.WriteString("res = map[string]interface{} {").WriteLine().IncreaseIndent()
		if len(f.Results) > 0 {
			for i, v := range f.Results {
				if i == len(f.Results)-1 && v.Type == "error" {
					continue
				}
				fieldName := "r" + strconv.Itoa(i+1)
				writer.WriteString(`"`, fieldName, `": `, fieldName, ",").WriteLine()
			}
		}
		writer.DecreaseIndent().WriteString("}").WriteLine()
		writer.WriteString("return").WriteLine()
		writer.DecreaseIndent().WriteString("},").WriteLine()
	}
	writer.DecreaseIndent().WriteString("}").WriteLine()
	return nil
}

func isSlotStruct(tpe *parse.StructType) bool {
	for _, v := range tpe.Fields {
		if v.Type == "meta.Slot" {
			return true
		}
	}
	return false
}

func isListenStruct(tpe *parse.StructType) bool {
	for _, v := range tpe.Fields {
		if v.Type == "meta.Listen" {
			return true
		}
	}
	return false
}

func getListenTarget(tpe *parse.StructType) string {
	for _, v := range tpe.Fields {
		if v.Type == "meta.Listen" {
			return v.Tag.Get("target")
		}
	}
	return ""
}
