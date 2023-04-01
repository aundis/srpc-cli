package parse

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"strings"
)

func ParseFile(filename string) (*File, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return ParseContent(filename, data)
}

// ParseFile 解析文件语法树, 不对Struct的方法整合
func ParseContent(filename string, content []byte) (*File, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, content, 0)
	if err != nil {
		return nil, err
	}
	res := &File{
		FileSet: fset,
		Content: content,
	}
	for _, item := range f.Imports {
		imp := &Import{}
		if item.Name != nil {
			imp.Name = item.Name.Name
		}
		imp.Path = strings.ReplaceAll(item.Path.Value, `"`, ``)
		if len(imp.Name) != 0 {
			imp.Export = imp.Name
		} else {
			index := strings.LastIndex(imp.Path, "/") + 1
			imp.Export = imp.Path[index:]
		}
		res.Imports = append(res.Imports, imp)
	}
	for _, v := range f.Decls {
		switch n := v.(type) {
		case *ast.FuncDecl:
			res.Functions = append(res.Functions, parseFunctionType(content, n))
		case *ast.GenDecl:
			for _, s := range n.Specs {
				if ts, ok := s.(*ast.TypeSpec); ok {
					switch ts.Type.(type) {
					case *ast.StructType:
						res.StructTypes = append(res.StructTypes, parseStructType(content, ts))
					case *ast.InterfaceType:
						res.InterfaceTypes = append(res.InterfaceTypes, parseInterfaceType(content, ts))
					}
				}
			}
		}
	}
	// 服务于错误提示
	for _, f := range res.Functions {
		f.Parent = res
	}
	for _, s := range res.StructTypes {
		s.Parent = res
	}
	for _, i := range res.InterfaceTypes {
		i.Parent = res
	}
	return res, nil
}

func parseStructType(content []byte, spec *ast.TypeSpec) *StructType {
	result := &StructType{
		Pos:  spec.Pos(),
		End:  spec.End(),
		Name: spec.Name.Name,
	}
	structType := spec.Type.(*ast.StructType)
	if structType.Fields != nil && structType.Fields.List != nil {
		for _, cur := range structType.Fields.List {
			field := &Field{}
			field.Pos = cur.Pos()
			if len(cur.Names) > 0 {
				field.Name = cur.Names[0].Name
			}
			if cur.Type != nil {
				field.Type = string(content[cur.Type.Pos()-1 : cur.Type.End()-1])
				field.TypeRaw = cur.Type
			}
			if cur.Tag != nil {
				field.Tag = Tag(cur.Tag.Value)
			}
			result.Fields = append(result.Fields, field)
		}
	}
	return result
}

func parseInterfaceType(content []byte, spec *ast.TypeSpec) *InterfaceType {
	interfaceType := spec.Type.(*ast.InterfaceType)
	tpe := &InterfaceType{
		Pos:  spec.Pos(),
		End:  spec.End(),
		Name: spec.Name.Name,
	}
	for _, m := range interfaceType.Methods.List {
		fun := &Function{
			Pos:  m.Pos(),
			Name: m.Names[0].Name,
		}
		fun.Params, fun.Results = parseFunctionParamAndResult(content, m.Type.(*ast.FuncType))
		tpe.Functions = append(tpe.Functions, fun)
	}
	return tpe
}

func parseFunctionType(content []byte, funcDecl *ast.FuncDecl) *Function {
	fun := &Function{
		Pos: funcDecl.Pos(),
		End: funcDecl.End(),
	}
	fun.Name = funcDecl.Name.Name
	fun.RecvTypeName = getRecvTypeName(content, funcDecl)
	fun.Params, fun.Results = parseFunctionParamAndResult(content, funcDecl.Type)
	return fun
}

func getRecvTypeName(content []byte, funcDecl *ast.FuncDecl) string {
	if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
		return ""
	}
	recv := funcDecl.Recv.List[0]
	recvType := string(content[recv.Type.Pos()-1 : recv.Type.End()-1])
	recvTypeName := strings.ReplaceAll(recvType, "*", "")
	return strings.TrimSpace(recvTypeName)
}

func parseFunctionParamAndResult(content []byte, funType *ast.FuncType) (params []*Field, results []*Field) {
	if funType.Params != nil {
		fparams := funType.Params.List
		for _, p := range fparams {
			tpe := string(content[p.Type.Pos()-1 : p.Type.End()-1])
			for _, n := range p.Names {
				params = append(params, &Field{
					Pos:     n.Pos(),
					Name:    n.Name,
					Type:    tpe,
					TypeRaw: p.Type,
				})
			}
		}
	}
	if funType.Results != nil {
		fresults := funType.Results.List
		for _, r := range fresults {
			tpe := string(content[r.Type.Pos()-1 : r.Type.End()-1])
			if len(r.Names) == 0 {
				results = append(results, &Field{
					Pos:     r.Pos(),
					Type:    tpe,
					TypeRaw: r.Type,
				})
			} else {
				for _, n := range r.Names {
					results = append(results, &Field{
						Pos:     n.Pos(),
						Name:    n.Name,
						Type:    tpe,
						TypeRaw: r.Type,
					})
				}
			}
		}
	}
	return
}

func CombineStructTypes(files []*File) []*StructType {
	var arr []*StructType
	structMap := map[string]*StructType{}
	for _, f := range files {
		for _, s := range f.StructTypes {
			structMap[s.Name] = s
			arr = append(arr, s)
		}
	}
	for _, f := range files {
		for _, fun := range f.Functions {
			if structMap[fun.RecvTypeName] != nil {
				structType := structMap[fun.RecvTypeName]
				structType.Functions = append(structType.Functions, fun)
			}
		}
	}
	return arr
}

func GetSlotStruct([]*StructType) []*StructType {
	return nil
}
