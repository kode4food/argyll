package main

import (
	"flag"
	"os"

	"github.com/kode4food/argyll/mcp"
)

func main() {
	engine := flag.String("engine", "http://localhost:8080", "Engine base URL")
	listen := flag.String("listen", "", "Address for the Streamable HTTP transport")
	flag.Parse()

	server := mcp.NewServer(*engine, nil)
	var err error
	if *listen == "" {
		err = server.Run()
	} else {
		err = server.MCPServer().AsHTTP(*listen).Run()
	}
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
