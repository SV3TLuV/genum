package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFromComment(t *testing.T) {
	tests := []struct {
		name          string
		comment       string
		sourceFile    string
		expected      *Directive
		expectedError string
		expectedNil   bool
	}{
		{
			name:       "valid comment with all flags",
			comment:    `//go:generate genum -type=Status -output=status_gen.go -trimprefix=Status_`,
			sourceFile: "types.go",
			expected: &Directive{
				TypeName:   "Status",
				OutputFile: "status_gen.go",
				TrimPrefix: "Status_",
			},
		},
		{
			name:       "valid comment with quotes",
			comment:    `//go:generate genum -type="Status" -output="status_gen.go" -trimprefix="Status_"`,
			sourceFile: "types.go",
			expected: &Directive{
				TypeName:   "Status",
				OutputFile: "status_gen.go",
				TrimPrefix: "Status_",
			},
		},
		{
			name:       "valid comment with single quotes",
			comment:    `//go:generate genum -type='Status' -output='status_gen.go' -trimprefix='Status_'`,
			sourceFile: "types.go",
			expected: &Directive{
				TypeName:   "Status",
				OutputFile: "status_gen.go",
				TrimPrefix: "Status_",
			},
		},
		{
			name:       "valid comment without output - auto generated",
			comment:    `//go:generate genum -type=Color`,
			sourceFile: "color.go",
			expected: &Directive{
				TypeName:   "Color",
				OutputFile: "color_genum.go",
				TrimPrefix: "Color",
			},
		},
		{
			name:       "valid comment without trimprefix",
			comment:    `//go:generate genum -type=Role -output=role_gen.go`,
			sourceFile: "user.go",
			expected: &Directive{
				TypeName:   "Role",
				OutputFile: "role_gen.go",
				TrimPrefix: "Role",
			},
		},
		{
			name:       "valid comment with mixed flags order",
			comment:    `//go:generate genum -output=test.go -trimprefix=Test_ -type=TestType`,
			sourceFile: "test.go",
			expected: &Directive{
				TypeName:   "TestType",
				OutputFile: "test.go",
				TrimPrefix: "Test_",
			},
		},
		{
			name:       "valid comment with extra spaces",
			comment:    `   //go:generate genum   -type=Status   -output=status.go   -trimprefix=Status_   `,
			sourceFile: "types.go",
			expected: &Directive{
				TypeName:   "Status",
				OutputFile: "status.go",
				TrimPrefix: "Status_",
			},
		},
		{
			name:          "missing type flag",
			comment:       `//go:generate genum -output=test.go`,
			sourceFile:    "test.go",
			expected:      nil,
			expectedError: "-type=<type> is required",
		},
		{
			name:          "empty type flag",
			comment:       `//go:generate genum -type=`,
			sourceFile:    "test.go",
			expected:      nil,
			expectedError: "-type=<type> is required",
		},
		{
			name:          "type flag with empty quotes",
			comment:       `//go:generate genum -type=""`,
			sourceFile:    "test.go",
			expected:      nil,
			expectedError: "-type=<type> is required",
		},
		{
			name:        "not a genum directive - different generator",
			comment:     `//go:generate stringer -type=Status`,
			sourceFile:  "types.go",
			expected:    nil,
			expectedNil: true,
		},
		{
			name:        "not a genum directive - regular comment",
			comment:     `// This is a regular comment`,
			sourceFile:  "types.go",
			expected:    nil,
			expectedNil: true,
		},
		{
			name:        "not a genum directive - empty comment",
			comment:     ``,
			sourceFile:  "types.go",
			expected:    nil,
			expectedNil: true,
		},
		{
			name:        "not a genum directive - whitespace only",
			comment:     `   `,
			sourceFile:  "types.go",
			expected:    nil,
			expectedNil: true,
		},
		{
			name:       "valid comment with multiple equals in value",
			comment:    `//go:generate genum -type=Complex=Type -output=file=with=equals.go`,
			sourceFile: "test.go",
			expected: &Directive{
				TypeName:   "Complex=Type",
				OutputFile: "file=with=equals.go",
				TrimPrefix: "Complex=Type",
			},
		},
		{
			name:       "valid comment with special characters in type name",
			comment:    `//go:generate genum -type=MyType123 -output=output_123.go -trimprefix=MyType123_`,
			sourceFile: "types.go",
			expected: &Directive{
				TypeName:   "MyType123",
				OutputFile: "output_123.go",
				TrimPrefix: "MyType123_",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseFromComment(tt.comment, tt.sourceFile)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)

			if tt.expectedNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.TypeName, result.TypeName)
				assert.Equal(t, tt.expected.OutputFile, result.OutputFile)
				assert.Equal(t, tt.expected.TrimPrefix, result.TrimPrefix)
			}
		})
	}
}

func TestIsGenumDirective(t *testing.T) {
	tests := []struct {
		name     string
		comment  string
		expected bool
	}{
		{
			name:     "valid genum directive",
			comment:  "//go:generate genum -type=Status",
			expected: true,
		},
		{
			name:     "valid genum directive with spaces",
			comment:  "   //go:generate genum -type=Status   ",
			expected: true,
		},
		{
			name:     "contains genum but not go:generate",
			comment:  "// genum -type=Status",
			expected: false,
		},
		{
			name:     "contains go:generate but not genum",
			comment:  "//go:generate stringer -type=Status",
			expected: false,
		},
		{
			name:     "empty string",
			comment:  "",
			expected: false,
		},
		{
			name:     "whitespace only",
			comment:  "   ",
			expected: false,
		},
		{
			name:     "regular comment",
			comment:  "// This is a regular comment",
			expected: false,
		},
		{
			name:     "genum in middle of text",
			comment:  "//go:generate somegenum tool",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGenumDirective(tt.comment)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCaseHandling_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		value   CaseHandling
		isValid bool
	}{
		{"valid: CaseSensitive", CaseSensitive, true},
		{"valid: CaseIgnore", CaseIgnore, true},
		{"valid: CaseLower", CaseLower, true},
		{"valid: CaseUpper", CaseUpper, true},
		{"not valid: random", CaseHandling("foobar"), false},
		{"not valid: empty", CaseHandling(""), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isValid, tt.value.IsValid())
		})
	}
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name          string
		comment       string
		expected      map[string]string
		expectError   bool
		errorContains string
	}{
		{
			name:    "all fields",
			comment: "//go:generate genum -type=Role -output=out.go -case=sensitive",
			expected: map[string]string{
				"-type":   "Role",
				"-output": "out.go",
				"-case":   "sensitive",
			},
		},
		{
			name:    "fields with quotes",
			comment: "//go:generate genum -type='Role' -output=\"out.go\"",
			expected: map[string]string{
				"-type":   "Role",
				"-output": "out.go",
			},
		},
		{
			name:          "missing equal sign error",
			comment:       "//go:generate genum -type",
			expectError:   true,
			errorContains: "invalid argument",
		},
		{
			name:    "empty value",
			comment: "//go:generate genum -type=",
			expected: map[string]string{
				"-type": "",
			},
		},
		{
			name:    "multiple equals in value",
			comment: "//go:generate genum -type=Foo=Bar -output=x.go",
			expected: map[string]string{
				"-type":   "Foo=Bar",
				"-output": "x.go",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags, err := ParseFlags(tt.comment)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				return
			}
			require.NoError(t, err)
			for k, v := range tt.expected {
				assert.Equal(t, v, flags[k])
			}
		})
	}
}
