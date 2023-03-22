package emit

import (
	"errors"
	"fmt"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"sr/parse"
	"strings"
)

func firstUpper(s string) string {
	if len(s) == 0 {
		return ""
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

func firstLower(s string) string {
	if len(s) == 0 {
		return ""
	}
	return strings.ToLower(string(s[0])) + s[1:]
}

func formatError(fset *token.FileSet, pos token.Pos, message string) error {
	// emit\util.go:21:1: missing return
	p := fset.Position(pos)
	return errors.New(fmt.Sprintf("%s:%d:%d: %s", p.Filename, p.Line, p.Column, message))
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

func listDir(dirname string, deep ...bool) ([]string, error) {
	fileInfos, err := ioutil.ReadDir(dirname)
	if err != nil {
		return nil, err
	}
	var list []string
	for _, fi := range fileInfos {
		filename := path.Join(dirname, fi.Name())
		if fi.IsDir() {
			list = append(list, filename)
			if len(deep) > 0 && deep[0] {
				//继续遍历fi这个目录
				dirs, err := listFile(filename, true)
				if err != nil {
					return nil, err
				}
				list = append(list, dirs...)
			}
		}
	}
	return list, nil
}

func ensureDirExist(dir string) error {
	ok, err := fileExist(dir)
	if err != nil {
		return err
	}
	if !ok {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func removeDirFiles(dir string) error {
	files, err := listFile(dir)
	if err != nil {
		return err
	}
	for _, v := range files {
		err := os.Remove(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func fileExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func stringEndOf(content string, part string) bool {
	return strings.LastIndex(content, part) == len(content)-len(part)
}

func resolveImport(file *parse.File, packageName string) *parse.Import {
	for _, imp := range file.Imports {
		if imp.Export == packageName {
			return imp
		}
	}
	return nil
}

func getPackageName(expr string) string {
	index := strings.Index(expr, ".")
	return strings.ReplaceAll(expr[:index], "*", "")
}

func isUsePackage(expr string) bool {
	return strings.Contains(expr, ".") && expr[0] != '.'
}

func command(arg ...string) error {
	name := "/bin/bash"
	c := "-c"
	// 根据系统设定不同的命令name
	if runtime.GOOS == "windows" {
		name = "cmd"
		c = "/C"
	}
	arg = append([]string{c}, arg...)
	cmd := exec.Command(name, arg...)
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}
