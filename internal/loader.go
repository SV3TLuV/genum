package internal

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

type Environment struct {
	Pkg            *packages.Package
	SourceFile     *ast.File
	SourceFileName string
}

func (e *Environment) PackageName() string {
	return e.Pkg.Name
}

type Loader struct {
	config *packages.Config
}

func NewLoader() *Loader {
	return &Loader{
		config: &packages.Config{
			Mode: packages.NeedName |
				packages.NeedFiles |
				packages.NeedTypes |
				packages.NeedTypesInfo |
				packages.NeedSyntax,
		},
	}
}

func (l *Loader) Load() (*Environment, error) {
	pkg, err := l.loadPackage()
	if err != nil {
		return nil, err
	}

	file, name, err := l.loadSourceFile(pkg)
	if err != nil {
		return nil, err
	}

	return &Environment{
		Pkg:            pkg,
		SourceFile:     file,
		SourceFileName: name,
	}, nil
}

func (l *Loader) loadPackage() (*packages.Package, error) {
	pkgs, err := packages.Load(l.config, ".")
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("package not found")
	}
	return pkgs[0], nil
}

func (l *Loader) loadSourceFile(pkg *packages.Package) (*ast.File, string, error) {
	sourceFileName := os.Getenv("GOFILE")
	for _, file := range pkg.Syntax {
		path := pkg.Fset.Position(file.Pos()).Filename
		if filepath.Base(path) == sourceFileName {
			return file, sourceFileName, nil
		}
	}
	return nil, "", fmt.Errorf("%s not find in package %s", sourceFileName, pkg.Name)
}
