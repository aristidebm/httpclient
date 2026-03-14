package commands

import (
	"fmt"

	"httpclient/internal/repl"
)

type logsCmd struct{}

func (c *logsCmd) Name() string      { return "logs" }
func (c *logsCmd) Aliases() []string { return nil }
func (c *logsCmd) Help() string      { return "View and manage request logs" }

func (c *logsCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	return nil
}

func (c *logsCmd) Run(ctx *repl.ShellContext, args []string) error {
	if len(args) == 0 {
		return logsList(ctx)
	}

	subcmd := args[0]
	switch subcmd {
	case "clear":
		return logsClear(ctx)
	case "remove":
		return logsRemove(ctx, args[1:])
	default:
		return fmt.Errorf("unknown logs subcommand: %s", subcmd)
	}
}

func logsList(ctx *repl.ShellContext) error {
	session := ctx.Tree.Current()
	if session == nil {
		return fmt.Errorf("no current session")
	}

	if len(session.Requests) == 0 {
		fmt.Println("No requests in this session.")
		return nil
	}

	fmt.Printf("%-4s %-6s %-8s %-28s %s\n", "ID", "METHOD", "STATUS", "CREATED AT", "PATH")
	fmt.Println("──────────────────────────────────────────────────────────────────────────────")
	for _, req := range session.Requests {
		method := req.Method
		status := ""
		if req.Response != nil {
			status = req.Response.Status
		}
		created := ""
		if !req.ExecutedAt.IsZero() {
			created = req.ExecutedAt.Format("2006-01-02T15:04:05Z07:00")
		}
		fmt.Printf("%-4s %-6s %-8s %-28s %s\n", req.ID, method, status, created, req.URL)
	}

	return nil
}

func logsClear(ctx *repl.ShellContext) error {
	session := ctx.Tree.Current()
	if session == nil {
		return fmt.Errorf("no current session")
	}

	session.Requests = nil
	repl.PrintSuccess("Cleared all request logs")
	return nil
}

func logsRemove(ctx *repl.ShellContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: /logs remove <req-id>")
	}

	reqID := args[0]
	session := ctx.Tree.Current()
	if session == nil {
		return fmt.Errorf("no current session")
	}

	if !session.RemoveRequest(reqID) {
		return fmt.Errorf("request %q not found", reqID)
	}

	repl.PrintSuccess(fmt.Sprintf("Removed request %s", reqID))
	return nil
}

func init() {
	repl.Register(&logsCmd{})
}
