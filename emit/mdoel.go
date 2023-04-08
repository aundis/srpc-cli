package emit

import (
	"path"
	"sr/parse"
	"sr/util"

	"github.com/gogf/gf/v2/os/gfile"
)

func emitModel(model *parse.Model, out string, root string) error {
	writer := util.NewTextWriter()
	writer.WriteString(generatedHeader).WriteLine()
	writer.WriteString("package ", getPackageNameForFileName(out)).WriteLine()
	if len(model.GetImports()) > 0 {
		writer.WriteEmptyLine()
	}
	for name, path := range model.GetImports() {
		if util.StringEndOf(path, name) {
			writer.WriteString(`import "` + path + `"`)
		} else {
			writer.WriteString("import " + name + ` "` + path + `"`)
		}
		writer.WriteLine()
	}
	for _, v := range model.GetTypes() {
		writer.WriteEmptyLine()
		writer.Write(v.Content)
		writer.WriteLine()
	}
	// 不存在这个文件夹则创建
	if !gfile.Exists(path.Dir(out)) {
		err := gfile.Mkdir(path.Dir(out))
		if err != nil {
			return err
		}
	}
	err := util.WriteGenerateFile(out, writer.Bytes(), root)
	if err != nil {
		return err
	}
	return nil
}
