package internal

import (
	"fmt"
	"strings"
)

const genumPrefix = "//go:generate genum "

type CaseHandling string

const (
	CaseSensitive CaseHandling = "sensitive"
	CaseIgnore    CaseHandling = "ignore"
	CaseLower     CaseHandling = "lower"
	CaseUpper     CaseHandling = "upper"
)

func (c CaseHandling) IsValid() bool {
	switch c {
	case CaseSensitive, CaseIgnore, CaseLower, CaseUpper:
		return true
	}
	return false
}

type Directive struct {
	TypeName   string
	OutputFile string
	TrimPrefix string
	Case       CaseHandling
}

func ParseFromComment(comment, sourceFile string) (*Directive, error) {
	comment = strings.TrimSpace(comment)
	if !IsGenumDirective(comment) {
		return nil, nil
	}

	flagMap, err := ParseFlags(comment)
	if err != nil {
		return nil, err
	}

	d := &Directive{
		Case: CaseSensitive,
	}
	for k, v := range flagMap {
		switch k {
		case "-type":
			d.TypeName = v
		case "-output":
			d.OutputFile = v
		case "-trimprefix":
			d.TrimPrefix = v
		case "-case":
			d.Case = CaseHandling(v)
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
	if !d.Case.IsValid() {
		return nil, fmt.Errorf("invalid argument -case: %s", d.Case)
	}

	return d, nil
}

func ParseFlags(comment string) (map[string]string, error) {
	parts := strings.Fields(strings.TrimPrefix(comment, genumPrefix))
	out := make(map[string]string, len(parts))

	for _, part := range parts {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("invalid argument: %s", part)
		}
		out[key] = strings.Trim(value, `"'`)
	}

	return out, nil
}

func IsGenumDirective(comment string) bool {
	comment = strings.TrimSpace(comment)
	return strings.HasPrefix(comment, genumPrefix)
}
