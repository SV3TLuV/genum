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
	Package    string
	Source     string
	Output     string
	TrimPrefix string
	Enums      []Enum
}

type Enum struct {
	TypeName string
	BaseType string
	Values   []EnumValue
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

		if _, ok := files[directive.OutputFile]; !ok {
			files[directive.OutputFile] = &File{
				Package:    env.PackageName(),
				Source:     env.SourceFileName,
				Output:     directive.OutputFile,
				TrimPrefix: directive.TrimPrefix,
				Enums:      []Enum{},
			}
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
		}

		if currentType == targetType {
			for _, name := range valueSpec.Names {
				if ast.IsExported(name.Name) {
					*values = append(*values, EnumValue{
						Name:  name.Name,
						Value: p.ExtractValue(valueSpec),
					})
				}
			}
		}
	}

	return currentType
}

func (p *Parser) ExtractValue(spec *ast.ValueSpec) string {
	if len(spec.Values) == 0 {
		return spec.Names[0].Name
	}

	switch v := spec.Values[0].(type) {
	case *ast.BasicLit:
		return strings.Trim(v.Value, `"`)
	case *ast.Ident:
		return v.Name
	default:
		return spec.Names[0].Name
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
