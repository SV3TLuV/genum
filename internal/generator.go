package internal

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type TemplateData struct {
	PackageName string
	TypeName    string
	BaseType    string
	Values      []EnumValue
}

type Generator struct {
	tmpl *template.Template
}

func NewGenerator() *Generator {
	funcMap := template.FuncMap{
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
		"title": cases.Title(language.English).String,
		"removePrefix": func(typeName, name string) string {
			return strings.TrimPrefix(name, typeName)
		},
	}
	return &Generator{
		tmpl: template.Must(template.New("enum").Funcs(funcMap).Parse(enumTemplate)),
	}
}

func (g *Generator) Generate(file File) error {
	code, err := g.GenerateFile(file)
	if err != nil {
		return fmt.Errorf("generate %s: %v", file.Output, err)
	}
	if err = g.WriteFile(file.Output, code); err != nil {
		return fmt.Errorf("write %s: %v", file.Output, err)
	}
	return nil
}

func (g *Generator) GenerateFile(file File) (string, error) {
	var buf strings.Builder
	err := g.tmpl.Execute(&buf, file)
	code := buf.String()
	return strings.TrimSpace(code), err
}

func (g *Generator) WriteFile(filename, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}
