package emit

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"path"
	"regexp"
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

var globalRoot string
var globalModule string

func EmitSlot(root string) error {
	globalRoot = root
	// 取项目模块名
	module, err := getProjectModuleName(root)
	if err != nil {
		return err
	}
	globalModule = module
	dirs, err := listDir(path.Join(root, "internal", "logic"))
	if err != nil {
		return err
	}
	outDir := path.Join(root, "internal", "srpc", "slot")
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
	var exportNames []string
	for _, dir := range dirs {
		names, err := emitSlotDir(module, dir, outDir)
		if err != nil {
			return err
		}
		exportNames = append(exportNames, names...)
	}
	// 生成 slot.go
	writer := newTextWriter()
	writer.WriteString("package slot").WriteLine()
	writer.WriteString("import \"github.com/aundis/srpc\"").WriteLine()
	writer.WriteString("import \"github.com/aundis/mate\"").WriteLine()
	writer.WriteString(mergeMapsFunc).WriteLine()
	// 合并所有的controller
	writer.WriteString("var Controllers = mergeMaps(").WriteLine().IncreaseIndent()
	for _, name := range exportNames {
		writer.WriteString(name+"Controller", ",").WriteLine()
	}
	writer.DecreaseIndent().WriteString(")").WriteLine()
	// 合并所有的helper
	writer.WriteString("var Helpers = []mate.ObjectMate{").WriteLine().IncreaseIndent()
	for _, name := range exportNames {
		writer.WriteString(name+"Helper", ",").WriteLine()
	}
	writer.DecreaseIndent().WriteString("}").WriteLine()
	err = ioutil.WriteFile(path.Join(root, "internal", "srpc", "slot", "slot.go"), writer.Bytes(), fs.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

type StructEmiter struct {
}

func emitSlotDir(module string, inDir string, outDir string) ([]string, error) {
	files, err := listFile(inDir)
	if err != nil {
		return nil, err
	}
	// 解析所有Go文件
	var astFiles []*parse.File
	for _, filename := range files {
		astFile, err := parse.ParseFile(filename)
		if err != nil {
			return nil, err
		}
		astFiles = append(astFiles, astFile)
	}
	// 合并结构类型
	structs := parse.CombineStructTypes(astFiles)
	// 提取出 slot 和 listen
	var targetStructs []*parse.StructType
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
			targetStructs = append(targetStructs, st)
			continue
		}
		if isListenStruct(st) {
			st.Functions = filterNoExport(st.Functions)
			if len(st.Functions) == 0 {
				continue
			}
			target := getListenTarget(st)
			if len(target) == 0 {
				return nil, formatError(st.Parent.FileSet, st.Pos, "not set listen object")
			}
			targetStructs = append(targetStructs, st)
		}
	}
	// 无内容则不生成
	if len(targetStructs) == 0 {
		return nil, nil
	}
	var names []string
	// 开始生成代码, 一个结构体对应一个文件
	for _, st := range targetStructs {
		names = append(names, firstLower(st.Name[1:]))
		writer := newTextWriter()
		// 生成头部信息
		writer.WriteString("package slot")
		writer.WriteLine()
		importMap := map[string]string{} // [name]path
		importMap["srpc"] = "github.com/aundis/srpc"
		importMap["mate"] = "github.com/aundis/mate"
		// 如果该类型的方法都只有一个ctx参数, 则不需要导入json
		if structNeedImportJson(st) {
			importMap["json"] = "encoding/json"
		}
		importMap["service"] = module + "/internal/service"
		for _, field := range getStructFields(st) {
			expr := field.Type
			if len(expr) == 0 {
				continue
			}
			if !isUsePackage(expr) {
				continue
			}
			name := getPackageName(expr)
			imp := resolveImport(st.Parent, name)
			if imp == nil {
				return nil, formatError(st.Parent.FileSet, field.Pos, "无法找到引用的模块"+name)
			}
			if len(importMap[imp.Export]) > 0 && imp.Path != importMap[imp.Export] {
				fmt.Printf("警告: 模块%s存在不同的导入路径 %s, %s\n", name, imp.Path, importMap[imp.Export])
			}
			importMap[imp.Export] = imp.Path
		}
		for name, path := range importMap {
			if stringEndOf(path, name) {
				writer.WriteString(`import "` + path + `"`)
			} else {
				writer.WriteString("import " + name + ` "` + path + `"`)
			}
			writer.WriteLine()
		}
		// emit
		err = emitStruct(writer, st)
		if err != nil {
			return nil, err
		}
		// helper
		err := emitSlotHelper(writer, st)
		if err != nil {
			return nil, err
		}
		filename := path.Join(outDir, toSnakeCase(st.Name[1:])+".go")
		err = ioutil.WriteFile(filename, writer.Bytes(), fs.ModePerm)
		if err != nil {
			return nil, err
		}
	}
	return names, nil
}

func resolveLocalTypeCode(pkgPath string, typeName string) (string, error) {
	localPath := convertPackagePathToLocalPath(pkgPath)
	modal, err := parse.ParsePackageModal(localPath)
	if err != nil {
		return "", err
	}
	if modal.ContainsType(typeName) {
		return string(modal.Types[typeName]), nil
	}
	return "", fmt.Errorf("package: %s, cannot found type: %s", pkgPath, typeName)
}

func convertPackagePathToLocalPath(pkgPath string) string {
	part := strings.Split(pkgPath, "/")
	return path.Join(globalRoot, strings.Join(part[1:], "/"))
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
			fmt.Println("警告: " + formatError(v.Parent.FileSet, v.Pos, "首参非context.Context, 方法 "+v.Name+" 被忽略").Error())
			continue
		}
		// 最后一个返回值必须为error
		if len(v.Results) == 0 || v.Results[len(v.Results)-1].Type != "error" {
			fmt.Println("警告: " + formatError(v.Parent.FileSet, v.Pos, "最后一个返回值不为error, 方法 "+v.Name+" 被忽略").Error())
			continue
		}
		result = append(result, v)
	}
	return result
}

func getStructFields(structType *parse.StructType) []*parse.Field {
	var result []*parse.Field
	for _, fun := range structType.Functions {
		result = append(result, fun.Params...)
		result = append(result, fun.Results...)
	}
	return result
}

var matchNonAlphaNumeric = regexp.MustCompile(`[^a-zA-Z0-9]+`)
var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	str = matchNonAlphaNumeric.ReplaceAllString(str, "_")     //非常规字符转化为 _
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}") //拆分出连续大写
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")  //拆分单词
	return strings.ToLower(snake)                             //全部转小写
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
			reqStructName := firstLower(f.Name) + "Request"
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
			reqStructName := firstLower(f.Name) + "Request"
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
		if v.Type == "mate.Slot" {
			return true
		}
	}
	return false
}

func isListenStruct(tpe *parse.StructType) bool {
	for _, v := range tpe.Fields {
		if v.Type == "mate.Listen" {
			return true
		}
	}
	return false
}

func getListenTarget(tpe *parse.StructType) string {
	for _, v := range tpe.Fields {
		if v.Type == "mate.Listen" {
			return v.Tag.Get("target")
		}
	}
	return ""
}
