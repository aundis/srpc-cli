package parse

import (
	"go/ast"
	"io/ioutil"
	"path"
	"runtime"
	"strings"
)

func formatPath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.ReplaceAll(path, "//", "/")
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
	}
	return path
}

func listFile(dirname string, deep ...bool) ([]string, error) {
	fileInfos, err := ioutil.ReadDir(dirname)
	if err != nil {
		return nil, err
	}
	var list []string
	for _, fi := range fileInfos {
		filename := path.Join(dirname, fi.Name())
		if fi.IsDir() && len(deep) > 0 && deep[0] {
			//继续遍历fi这个目录
			files, err := listFile(filename, true)
			if err != nil {
				return nil, err
			}
			list = append(list, files...)
		} else {
			list = append(list, filename)
		}
	}
	return list, nil
}

func IsFuncType(v interface{}) bool {
	switch v.(type) {
	case *ast.FuncType:
		return true
	}
	return false
}

func ParseFuncType(content []byte, v interface{}) (params []*Field, results []*Field) {
	return parseFunctionParamAndResult(content, v.(*ast.FuncType))
}

func IsStructType(v interface{}) bool {
	switch v.(type) {
	case StructType, *StructType:
		return true
	}
	return false
}
