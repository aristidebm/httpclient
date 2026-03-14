package commands

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"httpclient/internal/input"
	"httpclient/internal/repl"
)

type loadCmd struct{}

func (c *loadCmd) Name() string      { return "import" }
func (c *loadCmd) Aliases() []string { return []string{"load"} }
func (c *loadCmd) Help() string      { return "Import OpenAPI spec, .http file, or curl commands" }

func (c *loadCmd) Run(ctx *repl.ShellContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: /import <filepath-or-url>")
	}

	target := args[0]

	var data []byte
	var err error

	// Check if it's a URL
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		resp, err := http.Get(target)
		if err != nil {
			return fmt.Errorf("failed to fetch URL: %w", err)
		}
		defer resp.Body.Close()
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
	} else {
		// Read from file
		data, err = os.ReadFile(target)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
	}

	// Detect format
	format, err := input.Detect(data)
	if err != nil {
		return fmt.Errorf("failed to detect format: %w", err)
	}

	switch format {
	case input.FormatOpenAPI:
		spec, err := input.LoadOpenAPI(data)
		if err != nil {
			return fmt.Errorf("failed to load OpenAPI: %w", err)
		}

		ctx.SetSpec(spec)

		repl.PrintSuccess(fmt.Sprintf("Loaded OpenAPI spec: %s (version: %s)", spec.Title, spec.Version))
		fmt.Printf("Found %d routes\n", len(spec.Routes))

	case input.FormatHTTPFile:
		requests, err := input.ParseHTTPFile(data)
		if err != nil {
			return fmt.Errorf("failed to parse HTTP file: %w", err)
		}

		session := ctx.Tree.Current()
		for _, req := range requests {
			session.AddRequest(req)
		}

		repl.PrintSuccess(fmt.Sprintf("Loaded %d requests from HTTP file", len(requests)))

	case input.FormatCurl:
		requests, err := input.ParseCurl(data)
		if err != nil {
			return fmt.Errorf("failed to parse curl: %w", err)
		}

		session := ctx.Tree.Current()
		for _, req := range requests {
			session.AddRequest(req)
		}

		repl.PrintSuccess(fmt.Sprintf("Loaded %d requests from curl commands", len(requests)))

	default:
		return fmt.Errorf("unknown format")
	}

	return nil
}

func (c *loadCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	// Simple file path completion
	if strings.HasPrefix(partial, "/") || strings.HasPrefix(partial, "./") || strings.HasPrefix(partial, "../") {
		// Could add file path completion here
		return nil
	}
	return nil
}

func init() {
	repl.Register(&loadCmd{})
}
