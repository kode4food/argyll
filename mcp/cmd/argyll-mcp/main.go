package main

import (
	"flag"
	"os"

	"github.com/kode4food/argyll/mcp"
)

func main() {
	engine := flag.String(
		"engine",
		"http://localhost:8080",
		"Engine base URL",
	)
	flag.Parse()

	server := mcp.NewServer(*engine, nil)
	if err := server.Run(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
