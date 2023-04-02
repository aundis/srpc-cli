package util

import (
	"bytes"
	"errors"
	"io/ioutil"
	"path"
	"strings"

	"github.com/gogf/gf/v2/os/gfile"
)

func StringEndOf(content string, part string) bool {
	return strings.LastIndex(content, part) == len(content)-len(part)
}

func GetProjectModuleName(dir string) (string, error) {
	fileName := path.Join(dir, "go.mod")
	if !gfile.Exists(fileName) {
		return "", errors.New("not found go.mod")
	}
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
