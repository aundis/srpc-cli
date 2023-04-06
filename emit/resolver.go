package emit

import (
	"fmt"
	"go/token"
	"regexp"
	"sr/parse"
	"sr/util"
	"strings"

	"github.com/aundis/meta"
	"github.com/gogf/gf/v2/util/guid"
)

// map[model1.A]model2.B = > map[{{id1}}]{{id2}}
/*
type XXX struct {
    Name string
    Info map[string]{{id1}}
}
*/
type typeResolver struct {
	module   string
	root     string
	resolved map[string]*meta.TypeMeta
}

func (r *typeResolver) resolve(file *parse.File, compound string, pos token.Pos) (string, error) {
	// 获取包路径
	pkgPath, err := util.GetGoFilePackagePath(r.root, r.module, file.FileName)
	if err != nil {
		return "", err
	}
	var modelType *parse.ModelType
	typeNames := getTypeNames(compound)
	for _, typeName := range typeNames {
		var typeMeta *meta.TypeMeta
		if strings.Contains(typeName, ".") {
			arr := strings.Split(typeName, ".")
			scope := arr[0]
			name := arr[1]
			imp := resolveImport(file, scope)
			if imp == nil {
				return "", formatError(file.FileSet, pos, "not found type scope "+scope, r.root)
			}
			// 判断是否解析过了
			if r.resolved[imp.Path+"@"+name] != nil {
				typeMeta = r.resolved[imp.Path+"@"+name]
			} else {
				typeMeta = &meta.TypeMeta{
					Id:   guid.S(),
					Name: name,
				}
				// 本项目的类型才需要解析
				if !isProjectPackage(r.module, imp.Path) {
					typeMeta.Import = &meta.ImportMeta{
						Path:  imp.Path,
						Alias: imp.Name,
					}
				} else {
					model, err := parse.ParsePackageModel(packagePathToFileName(r.root, imp.Path))
					if err != nil {
						return "", err
					}
					if !model.ContainsType(name) {
						return "", formatError(file.FileSet, pos, fmt.Sprintf("package %s not found type %s", imp.Path, name), r.root)
					}
					modelType = model.GetType(name)
					typeMeta.From = imp.Path
					typeMeta.Code = string(modelType.Content)
				}
				r.resolved[imp.Path+"@"+name] = typeMeta
			}
		} else {
			if r.resolved[pkgPath+"@"+typeName] != nil {
				typeMeta = r.resolved[pkgPath+"@"+typeName]
			} else {
				model, err := parse.ParsePackageModel(packagePathToFileName(r.root, pkgPath))
				if err != nil {
					return "", err
				}
				if !model.ContainsType(typeName) {
					return "", formatError(file.FileSet, pos, fmt.Sprintf("package %s not found type %s", pkgPath, typeName), r.root)
				}
				modelType = model.GetType(typeName)
				typeMeta = &meta.TypeMeta{
					Id:     guid.S(),
					Name:   typeName,
					From:   pkgPath,
					Code:   string(modelType.Content),
					Import: nil,
				}
				r.resolved[pkgPath+"@"+typeName] = typeMeta
			}
		}
		if modelType != nil && parse.IsStructType(modelType.Raw) {
			fields := getStructInnerFields(modelType.Raw)
			structType := modelType.Raw.(*parse.StructType)
			for _, field := range fields {
				fieldType := field.Type
				if !hasCustomerType(fieldType) {
					continue
				}
				template, err := r.resolve(structType.Parent, field.Type, field.Pos)
				if err != nil {
					return "", err
				}
				// 优化替换, 防止出现名称和类型同名的情况
				reg := regexp.MustCompile(fmt.Sprintf(`\b%s\s+%s\b`, field.Name, regexp.QuoteMeta(field.Type)))
				typeMeta.Code = reg.ReplaceAllString(typeMeta.Code, fmt.Sprintf("%s %s", field.Name, template))
			}
		}
		reg := regexp.MustCompile(fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(typeName)))
		compound = reg.ReplaceAllString(compound, fmt.Sprintf("{{%s}}", typeMeta.Id))
	}
	return compound, nil
}

func (r *typeResolver) getTypeMetas() []*meta.TypeMeta {
	var result []*meta.TypeMeta
	for _, v := range r.resolved {
		result = append(result, v)
	}
	return result
}
