package parse

import (
	"fmt"
	"go/token"
	"regexp"
)

type File struct {
	FileSet        *token.FileSet
	FileName       string
	Content        []byte
	Imports        []*Import
	InterfaceTypes []*InterfaceType
	StructTypes    []*StructType
	Functions      []*Function
}

type InterfaceType struct {
	Parent    *File
	Pos       token.Pos
	End       token.Pos
	Name      string
	Functions []*Function
}

type StructType struct {
	Parent    *File
	Pos       token.Pos
	End       token.Pos
	Name      string
	Fields    []*Field
	Functions []*Function
}

type Tag string

func (t Tag) Get(key string) string {
	reg := regexp.MustCompile(fmt.Sprintf(`\s*%s\s*\:\s*"(.+?)"`, key))
	res := reg.FindAllStringSubmatch(string(t), -1)
	if len(res) > 0 && len(res[0]) == 2 {
		return res[0][1]
	}
	return ""
}

type Function struct {
	Parent       *File
	Pos          token.Pos
	End          token.Pos
	Name         string
	RecvTypeName string
	Params       []*Field
	Results      []*Field
}

type Field struct {
	Parent  *File
	Pos     token.Pos
	Name    string
	Type    string
	TypeRaw interface{}
	Tag     Tag
}

type Import struct {
	Name   string
	Path   string
	Export string
}
