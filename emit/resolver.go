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

func newFieldResolver(root, module, exportTo string) *fieldResolver {
	return &fieldResolver{
		module:   module,
		root:     root,
		exportTo: exportTo,
		tResolver: &typeResolver{
			module:   module,
			root:     root,
			resolved: map[string]*meta.TypeMeta{},
		},
		resolved: make(map[*parse.Field]string),
	}
}

type fieldResolver struct {
	module    string
	root      string
	exportTo  string
	tResolver *typeResolver
	resolved  map[*parse.Field]string
}

func (r *fieldResolver) resolve(field *parse.Field) error {
	if !hasCustomerType(field.Type) {
		r.resolved[field] = field.Type
		return nil
	}
	tResolver := r.tResolver
	template, err := tResolver.resolve(field.Parent, field.Type, field.Pos)
	if err != nil {
		return err
	}
	tmetas := tResolver.getTypeMetas()
	real, err := replacePseudocodePart(replacePseudocodePartInput{
		content: template,
		tmetas:  tResolver.getTypeMetas(),
		getExportTo: func(tmetaId string) string {
			return findTypeMetaForId(tmetas, tmetaId).From
		},
		currentPackage: r.exportTo,
	}), nil
	r.resolved[field] = real
	return nil
}

func (r *fieldResolver) getResolvedType(field *parse.Field) string {
	return r.resolved[field]
}

// func redirectTypeReference(fields []*parse.Field, exportTo, module, root string) error {
// 	resolver := &typeResolver{
// 		module:   module,
// 		root:     root,
// 		resolved: map[string]*meta.TypeMeta{},
// 	}
// 	for _, field := range fields {
// 		if !hasCustomerType(field.Type) {
// 			continue
// 		}
// 		template, err := resolver.resolve(field.Parent, field.Type, field.Pos)
// 		if err != nil {
// 			return err
// 		}
// 		tmetas := resolver.getTypeMetas()
// 		field.Type = replacePseudocodePart(replacePseudocodePartInput{
// 			content: template,
// 			tmetas:  resolver.getTypeMetas(),
// 			getExportTo: func(tmetaId string) string {
// 				return findTypeMetaForId(tmetas, tmetaId).From
// 			},
// 			currentPackage: exportTo,
// 		})
// 	}
// 	return nil
// }

type replacePseudocodePartInput struct {
	content        string
	tmetas         []*meta.TypeMeta
	getExportTo    func(string) string
	currentPackage string
}

var regPseudocode = regexp.MustCompile(`\{\{.+?\}\}`)

func replacePseudocodePart(in replacePseudocodePartInput) string {
	return regPseudocode.ReplaceAllStringFunc(in.content, func(s string) string {
		id := s[2 : len(s)-2]
		tmeta := findTypeMetaForId(in.tmetas, id)
		if tmeta.Import != nil {
			return getImportMetaExport(tmeta.Import) + "." + tmeta.Name
		} else {
			if in.getExportTo(tmeta.Id) != in.currentPackage {
				return getImportPathExport(in.getExportTo(tmeta.Id)) + "." + tmeta.Name
			} else {
				return tmeta.Name
			}
		}
	})
}

func hasCustomerType(in string) bool {
	arr := getTypeNames(in)
	for _, v := range arr {
		if !isBuiltin(v) {
			return true
		}
	}
	return false
}

func isBuiltin(in string) bool {
	builtin := []string{
		"int",
		"int8",
		"int16",
		"int32",
		"int64",
		"uint",
		"uint8",
		"uint16",
		"uint32",
		"uint64",
		"bool",
		"interface",
		"any",
		"map",
		"byte",
		"rune",
		"string",
	}
	for _, v := range builtin {
		if v == in {
			return true
		}
	}
	return false
}

func getTypeNames(compound string) []string {
	var results []string
	reg := regexp.MustCompile(`\b([\w\.]+)\b`)
	matchs := reg.FindAllStringSubmatch(compound, -1)
	for _, v := range matchs {
		tar := v[1]
		if strings.Contains(tar, ".") {
			results = append(results, tar)
		} else if tar[0] >= 'A' && tar[0] <= 'Z' {
			results = append(results, tar)
		}
	}
	return results
}

func findTypeMetaForId(arr []*meta.TypeMeta, id string) *meta.TypeMeta {
	for _, v := range arr {
		if v.Id == id {
			return v
		}
	}
	return nil
}

var typeMetaIdReg = regexp.MustCompile(`\{\{(.+?)\}\}`)

func findAllTypeMetaIds(content string) []string {
	var result []string
	matchs := typeMetaIdReg.FindAllStringSubmatch(content, -1)
	for _, v := range matchs {
		result = append(result, v[1])
	}
	return result
}
