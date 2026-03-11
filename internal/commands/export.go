package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"httpclient/internal/export"
	"httpclient/internal/model"
	"httpclient/internal/repl"

	_ "httpclient/internal/export"
)

type exportCmd struct{}

func (c *exportCmd) Name() string      { return "export" }
func (c *exportCmd) Aliases() []string { return nil }
func (c *exportCmd) Help() string      { return "Export session: /export [format] [session] [--out path]" }

func (c *exportCmd) Run(ctx *repl.ShellContext, args []string) error {
	format := "json"
	sessionName := ""
	var outPath string

	// Parse arguments
	for i, arg := range args {
		if arg == "--out" && i+1 < len(args) {
			outPath = args[i+1]
			break
		}
		if !argStartsWithDash(arg) {
			if format == "json" && (arg == "json" || arg == "curl" || arg == "har" || arg == "http" || arg == "bruno") {
				format = arg
			} else {
				sessionName = arg
			}
		}
	}

	// Find session
	session := ctx.Tree.Current()
	if sessionName != "" {
		found := false
		for _, s := range ctx.Tree.Sessions {
			if s.Name == sessionName {
				session = s
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("session %q not found", sessionName)
		}
	}

	if session == nil {
		return fmt.Errorf("no session to export")
	}

	// Get environment
	var env *model.Environment
	if session.EnvName != "" {
		env = ctx.Tree.Environments[session.EnvName]
	}

	// Export
	exporter, err := export.Get(format)
	if err != nil {
		return err
	}

	data, err := exporter.Export(session, env)
	if err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	// Handle bruno directory export
	if format == "bruno" && outPath != "" {
		err = export.ExportBrunoDir(session, env, outPath)
		if err != nil {
			return fmt.Errorf("failed to export bruno: %w", err)
		}
		repl.PrintSuccess(fmt.Sprintf("Exported to %s/", outPath))
		return nil
	}

	// Output
	if outPath != "" {
		dir := filepath.Dir(outPath)
		if dir != "." {
			os.MkdirAll(dir, 0755)
		}
		err = os.WriteFile(outPath, data, 0644)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		repl.PrintSuccess(fmt.Sprintf("Exported to %s", outPath))
	} else {
		fmt.Print(string(data))
	}

	return nil
}

func (c *exportCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	return nil
}

func argStartsWithDash(s string) bool {
	return len(s) > 0 && s[0] == '-'
}

func init() {
	repl.Register(&exportCmd{})
}
