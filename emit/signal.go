package emit

import (
	"io/ioutil"
	"os"
	"path"
	"sr/parse"

	"github.com/gogf/gf/v2/os/gfile"
)

func EmitSignal(root string) error {
	// 取项目模块名
	module, err := getProjectModuleName(root)
	if err != nil {
		return err
	}
	emiter := signalEmiter{
		root:   root,
		module: module,
		writer: newTextWriter(),
	}
	err = emiter.emit()
	if err != nil {
		return err
	}
	return nil
}

type signalEmiter struct {
	root   string
	module string
	writer TextWriter
}

func (e *signalEmiter) emit() error {
	dir := path.Join(e.root, "internal", "srpc", "emit")
	err := ensureDirExist(dir)
	if err != nil {
		return err
	}
	// 删除历史生成的文件
	outPath := path.Join(dir, "generate.go")
	if gfile.Exists(outPath) {
		err = os.Remove(outPath)
		if err != nil {
			return err
		}
	}
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
	// 获取所有需要处理的接口类型
	var interfaceTypes []*parse.InterfaceType
	for _, filename := range goFiles {
		astFile, err := parse.ParseFile(filename)
		if err != nil {
			return err
		}
		for _, it := range astFile.InterfaceTypes {
			interfaceTypes = append(interfaceTypes, it)
		}
	}
	// 内容不生成文件
	if len(interfaceTypes) == 0 {
		return nil
	}
	// 生成头部信息
	writer := e.writer
	writer.WriteString("package emit").WriteLine()
	collect := newImportCollect()
	collect.Set("context", "context")
	collect.Set("json", "encoding/json")
	collect.Set("srpc", "github.com/aundis/srpc")
	collect.Set("mate", "github.com/aundis/mate")
	collect.Set("service", e.module+"/internal/service")
	for _, it := range interfaceTypes {
		err = resolveInterfaceImports(it, collect)
		if err != nil {
			return err
		}
	}
	collect.Emit(e.writer)
	// 生成代码内容
	for _, it := range interfaceTypes {
		err = emitInterface(e.writer, "main", it, true)
		if err != nil {
			return err
		}
	}
	// helper 放到文件内容尾部
	for _, it := range interfaceTypes {
		err = emitSignalHelper(e.root, e.module, writer, it)
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
	err = ioutil.WriteFile(outPath, writer.Bytes(), os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}
