package internal

import (
	"fmt"
	"strings"
)

type Directive struct {
	TypeName   string
	OutputFile string
	TrimPrefix string
}

func ParseFromComment(comment, sourceFile string) (*Directive, error) {
	comment = strings.TrimSpace(comment)
	if !IsGenumDirective(comment) {
		return nil, nil
	}

	d := &Directive{}
	flagMapping := map[string]*string{
		"-type":       &d.TypeName,
		"-output":     &d.OutputFile,
		"-trimprefix": &d.TrimPrefix,
	}

	parts := strings.Fields(comment)
	for _, part := range parts {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		if field, ok := flagMapping[key]; ok {
			*field = strings.Trim(value, `"'`)
		}
	}

	if d.TypeName == "" {
		return nil, fmt.Errorf("-type=<type> is required")
	}
	if d.OutputFile == "" {
		d.OutputFile = strings.Replace(sourceFile, ".go", "_genum.go", 1)
	}
	if d.TrimPrefix == "" {
		d.TrimPrefix = d.TypeName
	}

	return d, nil
}

func IsGenumDirective(comment string) bool {
	comment = strings.TrimSpace(comment)
	return strings.HasPrefix(comment, "//go:generate genum ")
}
