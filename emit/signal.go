package emit

import (
	"fmt"
	"os"
	"path"
	"sr/parse"
)

func EmitSignal(root string) error {
	// 取项目模块名
	module, err := getProjectModuleName(root)
	if err != nil {
		return err
	}
	dir := path.Join(root, "internal", "srpc", "emit")
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
	var interfaceTypes []*parse.InterfaceType
	for _, filename := range goFiles {
		astFile, err := parse.ParseFile(filename)
		if err != nil {
			return err
		}
		for _, it := range astFile.InterfaceTypes {
			interfaceTypes = append(interfaceTypes, it)
			err = emitInterface(writer, "main", it, true)
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
	importMap["mate"] = "github.com/aundis/mate"
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
				return formatError(it.Parent.FileSet, field.Pos, "cannot found refrence pacakge"+name)
			}
			if len(importMap[imp.Export]) > 0 && imp.Path != importMap[imp.Export] {
				fmt.Printf("warning: package %s exist different import path %s, %s\n", name, imp.Path, importMap[imp.Export])
			}
			importMap[imp.Export] = imp.Path
		}
	}
	headerWriter := newTextWriter()
	headerWriter.WriteString("package emit")
	headerWriter.WriteLine()
	for name, path := range importMap {
		if stringEndOf(path, name) {
			headerWriter.WriteString(`import "` + path + `"`)
		} else {
			headerWriter.WriteString("import " + name + ` "` + path + `"`)
		}
		headerWriter.WriteLine()
	}
	// helper 放到文件内容尾部
	for _, it := range interfaceTypes {
		err = emitSignalHelper(writer, it)
		if err != nil {
			return err
		}
	}
	// helper 合并
	writer.WriteString("var Helpers = []mate.ObjectMate{").WriteLine().IncreaseIndent()
	for _, it := range interfaceTypes {
		writer.WriteString(firstLower(it.Name[1:]), "Helper,").WriteLine()
	}
	writer.DecreaseIndent().WriteString("}").WriteLine()
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
