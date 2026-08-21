// Command argyll-gen generates Argyll step adapters for Go functions marked
// with //argyll:step or //argyll:wrap directives
package main

import (
	"fmt"
	"os"

	"github.com/kode4food/argyll/sdk/go/gen/internal/generator"
)

func main() {
	patterns := os.Args[1:]
	if len(patterns) == 0 {
		patterns = []string{"."}
	}

	written, err := generator.Generate(".", patterns...)
	if err != nil {
		fmt.Fprintln(os.Stderr, "argyll-gen:", err)
		os.Exit(1)
	}
	for _, p := range written {
		fmt.Println("argyll-gen:", p)
	}
}
