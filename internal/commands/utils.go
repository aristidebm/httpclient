package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"httpclient/internal/model"
	"httpclient/internal/repl"
)

type filterCmd struct{}

func (c *filterCmd) Name() string      { return "filter" }
func (c *filterCmd) Aliases() []string { return nil }
func (c *filterCmd) Help() string {
	return "Filter output: /filter <tool> [args...] or /filter --request [req-id]"
}

func (c *filterCmd) Run(ctx *repl.ShellContext, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: /filter <tool> [args...] or /filter --request [req-id]")
	}

	// Check for --request flag
	if args[0] == "--request" {
		var req *model.Request

		if len(args) == 1 {
			// Use last request from session
			session := ctx.Tree.Current()
			if session == nil || len(session.Requests) == 0 {
				return fmt.Errorf("no requests in session")
			}
			req = session.Requests[len(session.Requests)-1]
		} else {
			// Use specific request ID
			reqID := args[1]
			session := ctx.Tree.Current()
			if session == nil {
				return fmt.Errorf("no current session")
			}
			var ok bool
			req, ok = session.GetRequest(reqID)
			if !ok {
				return fmt.Errorf("request %q not found in session", reqID)
			}
		}

		if req.Response == nil {
			return fmt.Errorf("request %q has not been executed", req.ID)
		}

		// Set as last response
		ctx.LastResp = req.Response

		// Run filter tool if additional args provided
		if len(args) > 2 {
			return runFilter(strings.Join(args[2:], " "))
		}
		return nil
	}

	// Run the filter tool on last response data
	tool := args[0]
	toolArgs := args[1:]

	if ctx.LastData == nil && ctx.LastResp == nil {
		return fmt.Errorf("no data to filter. Execute a request first.")
	}

	// Convert last response to JSON
	var data []byte
	var err error

	if ctx.LastData != nil {
		data, err = json.Marshal(ctx.LastData)
	} else if ctx.LastResp != nil {
		data, err = json.Marshal(ctx.LastResp)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal data: %v", err)
	}

	return runToolWithInput(tool, toolArgs, data)
}

func runFilter(filter string) error {
	if filter == "" {
		return nil
	}

	if strings.HasPrefix(filter, ".") || strings.HasPrefix(filter, "try") {
		return runToolWithInput("jq", []string{filter}, nil)
	}

	parts := strings.Fields(filter)
	if len(parts) == 0 {
		return nil
	}
	return runToolWithInput(parts[0], parts[1:], nil)
}

func runToolWithInput(tool string, args []string, input []byte) error {
	cmd := exec.Command(tool, args...)
	cmd.Stdin = bytes.NewReader(input)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *filterCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	return []string{"jq", "fx", "fzf", "--request"}
}

type editorCmd struct{}

func (c *editorCmd) Name() string      { return "editor" }
func (c *editorCmd) Aliases() []string { return nil }
func (c *editorCmd) Help() string      { return "Edit content in $EDITOR" }

func (c *editorCmd) Run(ctx *repl.ShellContext, args []string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	// Get content to pre-populate the temp file with
	var content string
	if ctx.LastData != nil {
		data, err := json.MarshalIndent(ctx.LastData, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal data: %v", err)
		}
		content = string(data)
	} else if ctx.LastResp != nil {
		data, err := json.MarshalIndent(ctx.LastResp, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal data: %v", err)
		}
		content = string(data)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "httpclient-*.txt")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Run editor — must connect stdio so the TUI renders correctly
	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor error: %v", err)
	}

	// Read back what the user wrote
	editedContent, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to read edited file: %v", err)
	}

	// Strip trailing newlines so readline doesn't auto-submit an empty line
	text := strings.TrimRight(string(editedContent), "\r\n")
	if text == "" {
		return nil
	}

	// Inject the text into readline's input buffer WITHOUT a trailing '\n'.
	// readline drains WriteStdin byte-by-byte into its line buffer and redraws
	// the prompt, so the result is:
	//
	//   [user@host] › Something█          ← cursor here, nothing submitted yet
	//
	// The user can still edit the text or just press Enter to execute it.
	if ctx.Readline == nil {
		// Fallback: no readline instance wired up, store for caller to handle
		ctx.Pending = text
		return nil
	}
	if _, err := ctx.Readline.WriteStdin([]byte(text)); err != nil {
		return fmt.Errorf("failed to inject text into readline: %v", err)
	}
	return nil
}

func (c *editorCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	return nil
}

type replayCmd struct{}

func (c *replayCmd) Name() string      { return "replay" }
func (c *replayCmd) Aliases() []string { return nil }
func (c *replayCmd) Help() string      { return "Replay request: /replay [req-id|all]" }

func (c *replayCmd) Run(ctx *repl.ShellContext, args []string) error {
	session := ctx.Tree.Current()
	if session == nil {
		return fmt.Errorf("no session")
	}

	env := ctx.Tree.CurrentEnv()
	if env == nil {
		return fmt.Errorf("no environment")
	}

	target := ""
	if len(args) > 0 {
		target = args[0]
	}

	if target == "" {
		if ctx.LastReqID != "" {
			target = ctx.LastReqID
		} else {
			return fmt.Errorf("specify request id or use /replay all")
		}
	}

	if target == "all" {
		for _, req := range session.Requests {
			if !req.IsExecuted() {
				continue
			}
			cloned := req.Clone()
			if err := ctx.Executor.Execute(cloned, env); err != nil {
				fmt.Fprintf(os.Stderr, "Error replaying %s: %v\n", req.ID, err)
				continue
			}
			session.AddRequest(cloned)
			repl.PrintResponse(cloned.Response)
		}
	} else {
		req, ok := session.GetRequest(target)
		if !ok {
			return fmt.Errorf("request %s not found", target)
		}
		if !req.IsExecuted() {
			return fmt.Errorf("request %s not executed", target)
		}

		cloned := req.Clone()
		if err := ctx.Executor.Execute(cloned, env); err != nil {
			return err
		}
		session.AddRequest(cloned)
		repl.PrintResponse(cloned.Response)
	}

	return nil
}

func (c *replayCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	var ids []string
	session := ctx.Tree.Current()
	if session != nil {
		for _, req := range session.Requests {
			if req.IsExecuted() {
				ids = append(ids, req.ID)
			}
		}
	}
	return ids
}

type watchCmd struct{}

func (c *watchCmd) Name() string      { return "watch" }
func (c *watchCmd) Aliases() []string { return nil }
func (c *watchCmd) Help() string      { return "Watch request: /watch [req-id] [interval]" }

func (c *watchCmd) Run(ctx *repl.ShellContext, args []string) error {
	session := ctx.Tree.Current()
	if session == nil {
		return fmt.Errorf("no session")
	}

	env := ctx.Tree.CurrentEnv()
	if env == nil {
		return fmt.Errorf("no environment")
	}

	reqID := ""
	interval := 2

	if len(args) >= 1 {
		reqID = args[0]
	}
	if len(args) >= 2 {
		fmt.Sscanf(args[1], "%d", &interval)
	}

	if reqID == "" {
		reqID = ctx.LastReqID
	}
	if reqID == "" {
		return fmt.Errorf("specify request id")
	}

	req, ok := session.GetRequest(reqID)
	if !ok {
		return fmt.Errorf("request %s not found", reqID)
	}

	fmt.Printf("Watching %s every %ds (Ctrl-C to stop)\n", reqID, interval)

	for {
		cloned := req.Clone()
		if err := ctx.Executor.Execute(cloned, env); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			session.AddRequest(cloned)
			fmt.Printf("[%s] %d %s\n", cloned.ID, cloned.Response.StatusCode, cloned.Response.Status)
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func (c *watchCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	var ids []string
	session := ctx.Tree.Current()
	if session != nil {
		for _, req := range session.Requests {
			if req.IsExecuted() {
				ids = append(ids, req.ID)
			}
		}
	}
	return ids
}

type saveCmd struct{}

func (c *saveCmd) Name() string      { return "save" }
func (c *saveCmd) Aliases() []string { return nil }
func (c *saveCmd) Help() string      { return "Save last response to file: /save [filename]" }

func (c *saveCmd) Run(ctx *repl.ShellContext, args []string) error {
	if ctx.LastResp == nil {
		return fmt.Errorf("no response to save")
	}

	filename := ""
	if len(args) > 0 {
		filename = args[0]
	}

	if filename == "" {
		if cd := ctx.LastResp.Headers["Content-Disposition"]; cd != "" {
			parts := strings.Split(cd, "filename=")
			if len(parts) > 1 {
				filename = strings.Trim(parts[1], "\"")
			}
		}
	}

	if filename == "" {
		ext := ""
		ct := ctx.LastResp.Headers["Content-Type"]
		switch {
		case strings.Contains(ct, "pdf"):
			ext = ".pdf"
		case strings.Contains(ct, "zip"):
			ext = ".zip"
		case strings.Contains(ct, "json"):
			ext = ".json"
		case strings.Contains(ct, "xml"):
			ext = ".xml"
		}
		filename = fmt.Sprintf("httpclient_%d%s", time.Now().Unix(), ext)
	}

	if err := os.WriteFile(filename, ctx.LastResp.RawBody, 0644); err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}

	fmt.Printf("Saved to: %s\n", filename)
	return nil
}

func (c *saveCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	return nil
}

func init() {
	repl.Register(&filterCmd{})
	repl.Register(&editorCmd{})
	repl.Register(&replayCmd{})
	repl.Register(&watchCmd{})
	repl.Register(&saveCmd{})
}
