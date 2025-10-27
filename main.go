package main

import (
	"fmt"
	"os"

	"github.com/sv3tluv/genum/internal"
)

func main() {
	loader := internal.NewLoader()
	env, err := loader.Load()
	if err != nil {
		fail("%v", err)
	}

	parser := internal.NewParser()
	files, err := parser.Parse(env)
	if err != nil {
		fail("%v", err)
	}

	generator := internal.NewGenerator()
	for _, file := range files {
		if err = generator.Generate(file); err != nil {
			fail("%v", err)
		}
	}
}

func fail(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
