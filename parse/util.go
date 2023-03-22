package parse

import (
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
