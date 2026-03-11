package commands

import (
	"fmt"

	"httpclient/internal/repl"
)

type listCmd struct{}

func (c *listCmd) Name() string      { return "list" }
func (c *listCmd) Aliases() []string { return nil }
func (c *listCmd) Help() string      { return "List requests in current session" }

func (c *listCmd) Run(ctx *repl.ShellContext, args []string) error {
	session := ctx.Tree.Current()
	if session == nil {
		return fmt.Errorf("no current session")
	}

	if len(session.Requests) == 0 {
		fmt.Println("No requests in this session")
		return nil
	}

	fmt.Printf("Requests in session %q:\n", session.Name)
	fmt.Println()
	for _, req := range session.Requests {
		status := ""
		if req.Response != nil {
			status = fmt.Sprintf(" → %d %s", req.Response.StatusCode, req.Response.Status)
		} else {
			status = " (not executed)"
		}
		note := ""
		if req.Note != "" {
			note = fmt.Sprintf(" — %s", req.Note)
		}
		fmt.Printf("  [%s] %s %s%s%s\n", req.ID, req.Method, req.URL, status, note)
	}

	return nil
}

func (c *listCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	return nil
}

func init() {
	repl.Register(&listCmd{})
}
