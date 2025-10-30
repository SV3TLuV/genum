package internal

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

type File struct {
	Package string
	Source  string
	Output  string
	Enums   []Enum

	NeedStringsPackage bool
}

type Enum struct {
	TypeName   string
	BaseType   string
	TrimPrefix string
	Case       string
	Values     []EnumValue
}

type EnumValue struct {
	Name  string
	Value string
}

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(env *Environment) ([]File, error) {
	directives, err := p.ParseFileDirectives(env)
	if err != nil {
		return nil, err
	}
	if len(directives) == 0 {
		return nil, fmt.Errorf("no genum directives found")
	}

	files := make(map[string]*File)
	for _, directive := range directives {
		enum, err := p.ParseSingleEnum(env.Pkg, &directive)
		if err != nil {
			return nil, fmt.Errorf("failed to parse enum %s: %w", directive.TypeName, err)
		}

		enum.TrimPrefix = directive.TrimPrefix
		enum.Case = string(directive.Case)

		if _, ok := files[directive.OutputFile]; !ok {
			file := &File{
				Package: env.PackageName(),
				Source:  env.SourceFileName,
				Output:  directive.OutputFile,
				Enums:   []Enum{},
			}

			if !file.NeedStringsPackage {
				file.NeedStringsPackage =
					directive.Case != CaseSensitive &&
						enum.BaseType == "string"
			}

			files[directive.OutputFile] = file
		}
		files[directive.OutputFile].Enums = append(files[directive.OutputFile].Enums, *enum)
	}

	out := make([]File, 0, len(files))
	for _, file := range files {
		out = append(out, *file)
	}
	return out, nil
}

func (p *Parser) ParseFileDirectives(env *Environment) ([]Directive, error) {
	var directives []Directive
	ast.Inspect(env.SourceFile, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Doc != nil {
			for _, comment := range genDecl.Doc.List {
				directive, err := ParseFromComment(comment.Text, env.SourceFileName)
				if err != nil {
					return false
				}
				if directive != nil {
					directives = append(directives, *directive)
				}
			}
		}
		return true
	})

	return directives, nil
}

func (p *Parser) ParseSingleEnum(pkg *packages.Package, directive *Directive) (*Enum, error) {
	baseType := p.ParseBaseType(pkg, directive.TypeName)
	if baseType == nil || *baseType == "" {
		return nil, fmt.Errorf("type %s not found", directive.TypeName)
	}

	values := p.ParseConstants(pkg, directive.TypeName)
	if len(values) == 0 {
		return nil, fmt.Errorf("no values found for enum %s", directive.TypeName)
	}

	return &Enum{
		TypeName: directive.TypeName,
		BaseType: *baseType,
		Values:   values,
	}, nil
}

func (p *Parser) ParseConstants(pkg *packages.Package, typeName string) []EnumValue {
	var values []EnumValue
	var currentType string

	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.GenDecl:
				if node.Tok == token.CONST {
					currentType = p.ProcessConstGroup(node, typeName, &values, currentType)
				}
			}
			return true
		})
	}

	return values
}

func (p *Parser) ProcessConstGroup(decl *ast.GenDecl, targetType string, values *[]EnumValue, lastType string) string {
	currentType := lastType

	for _, spec := range decl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok || len(valueSpec.Names) == 0 {
			continue
		}

		if valueSpec.Type != nil {
			if ident, ok := valueSpec.Type.(*ast.Ident); ok {
				currentType = ident.Name
			}
		} else if len(valueSpec.Values) > 0 {
			currentType = p.ExtractTypeFromValue(valueSpec.Values[0])
			if currentType == "" {
				continue
			}
		}

		if currentType == targetType {
			for i, name := range valueSpec.Names {
				if ast.IsExported(name.Name) {
					value := p.ExtractValue(valueSpec, i)
					*values = append(*values, EnumValue{
						Name:  name.Name,
						Value: value,
					})
				}
			}
		}
	}

	return currentType
}

func (p *Parser) ExtractTypeFromValue(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.CallExpr:
		if ident, ok := v.Fun.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

func (p *Parser) ExtractValue(spec *ast.ValueSpec, index int) string {
	if len(spec.Values) <= index {
		return spec.Names[index].Name
	}
	return p.extractValueFromExpr(spec.Values[index], spec.Names[index].Name)
}

func (p *Parser) extractValueFromExpr(expr ast.Expr, defaultValue string) string {
	switch v := expr.(type) {
	case *ast.BasicLit:
		return strings.Trim(v.Value, `"`)
	case *ast.Ident:
		return v.Name
	case *ast.CallExpr:
		if len(v.Args) == 0 {
			return defaultValue
		}
		return p.extractValueFromExpr(v.Args[0], defaultValue)
	default:
		return defaultValue
	}
}

func (p *Parser) ParseBaseType(pkg *packages.Package, typeName string) *string {
	if pkg.TypesInfo == nil {
		return nil
	}

	obj := pkg.Types.Scope().Lookup(typeName)
	if obj == nil {
		return nil
	}

	typeNameObj, ok := obj.(*types.TypeName)
	if !ok {
		return nil
	}

	baseType := p.TypeString(typeNameObj.Type())
	return &baseType
}

func (p *Parser) TypeString(typ types.Type) string {
	switch t := typ.(type) {
	case *types.Basic:
		return t.Name()
	case *types.Named:
		return p.TypeString(t.Underlying())
	case *types.Pointer:
		return "*" + p.TypeString(t.Elem())
	case *types.Struct:
		return "struct{}"
	}

	return "unsupported"
}
