package main

import (
	"fmt"
	"os"

	_ "httpclient/internal/commands"
	"httpclient/internal/repl"
)

func main() {
	ctx := repl.NewShellContext()
	if err := ctx.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
