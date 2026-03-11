package commands

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"httpclient/internal/repl"
	"github.com/itchyny/gojq"
)

type jqCmd struct{}

func (c *jqCmd) Name() string      { return "jq" }
func (c *jqCmd) Aliases() []string { return nil }
func (c *jqCmd) Help() string      { return "Run jq pattern against last response data" }

func (c *jqCmd) Run(ctx *repl.ShellContext, args []string) error {
	if ctx.LastData == nil {
		return fmt.Errorf("no data to query. Execute a request first.")
	}

	fs := flag.NewFlagSet("jq", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Println("Usage: /jq <pattern> [-f]")
	}
	firstOnly := fs.Bool("f", false, "Return only first result")

	if err := fs.Parse(args); err != nil {
		return err
	}

	pattern := fs.Arg(0)
	if pattern == "" {
		return fmt.Errorf("pattern required")
	}

	query, err := gojq.Parse(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	// Convert lastData to interface{} for gojq
	var data interface{}
	jsonBytes, err := json.Marshal(ctx.LastData)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	json.Unmarshal(jsonBytes, &data)

	var results []any
	iter := query.Run(data)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return fmt.Errorf("query error: %w", err)
		}
		results = append(results, v)
	}

	if len(results) == 0 {
		fmt.Println("No results")
		return nil
	}

	if *firstOnly {
		out, _ := json.MarshalIndent(results[0], "", "  ")
		fmt.Println(string(out))
		ctx.LastData = results[0]
	} else {
		out, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(out))
		ctx.LastData = results
	}

	return nil
}

func (c *jqCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	return nil
}

type updateCmd struct{}

func (c *updateCmd) Name() string      { return "update" }
func (c *updateCmd) Aliases() []string { return nil }
func (c *updateCmd) Help() string      { return "Edit last response data in editor" }

func (c *updateCmd) Run(ctx *repl.ShellContext, args []string) error {
	varName := ""
	if len(args) > 0 {
		varName = args[0]
	}

	data := ctx.LastData
	if data == nil {
		return fmt.Errorf("no data to edit")
	}

	// Get editor
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "nano"
	}

	// Write to temp file
	tmpFile := fmt.Sprintf("/tmp/httpclient_update_%d.json", os.Getpid())
	defer os.Remove(tmpFile)

	var content []byte
	if varName == "" {
		content, _ = json.MarshalIndent(data, "", "  ")
	} else {
		if val, ok := ctx.Vars[varName]; ok {
			content, _ = json.MarshalIndent(val, "", "  ")
		} else {
			content = []byte("{}")
		}
	}

	os.WriteFile(tmpFile, content, 0644)

	// Run editor
	cmd := exec.Command(editor, tmpFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor error: %w", err)
	}

	// Check if changed
	newContent, err := os.ReadFile(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to read temp file: %w", err)
	}

	if string(content) == string(newContent) {
		fmt.Println("No changes")
		return nil
	}

	// Try to parse as JSON
	var parsed any
	if err := json.Unmarshal(newContent, &parsed); err != nil {
		// Store as string if not valid JSON
		parsed = string(newContent)
	}

	if varName == "" {
		ctx.LastData = parsed
	} else {
		ctx.Vars[varName] = parsed
	}

	fmt.Println("Updated")
	return nil
}

func (c *updateCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	var names []string
	for k := range ctx.Vars {
		names = append(names, k)
	}
	return names
}

type editCmd struct{}

func (c *editCmd) Name() string      { return "edit" }
func (c *editCmd) Aliases() []string { return nil }
func (c *editCmd) Help() string      { return "Edit a variable in editor" }

func (c *editCmd) Run(ctx *repl.ShellContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: /edit <var-name>")
	}

	varName := args[0]

	// Get editor
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "nano"
	}

	// Write to temp file
	tmpFile := fmt.Sprintf("/tmp/httpclient_edit_%s.json", varName)
	defer os.Remove(tmpFile)

	var content []byte
	if val, ok := ctx.Vars[varName]; ok {
		content, _ = json.MarshalIndent(val, "", "  ")
	} else {
		content = []byte("{}")
	}

	os.WriteFile(tmpFile, content, 0644)

	// Run editor
	cmd := exec.Command(editor, tmpFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor error: %w", err)
	}

	// Check if changed
	newContent, err := os.ReadFile(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to read temp file: %w", err)
	}

	if string(content) == string(newContent) {
		fmt.Println("No changes")
		return nil
	}

	// Try to parse as JSON
	var parsed any
	if err := json.Unmarshal(newContent, &parsed); err != nil {
		parsed = string(newContent)
	}

	ctx.Vars[varName] = parsed
	fmt.Println("Updated")
	return nil
}

func (c *editCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	var names []string
	for k := range ctx.Vars {
		names = append(names, k)
	}
	return names
}

type clipCmd struct{}

func (c *clipCmd) Name() string      { return "clip" }
func (c *clipCmd) Aliases() []string { return nil }
func (c *clipCmd) Help() string      { return "Copy to clipboard: /clip [var-name|req-id]" }

func (c *clipCmd) Run(ctx *repl.ShellContext, args []string) error {
	var content string

	if len(args) == 0 {
		// Copy last data
		if ctx.LastData == nil {
			return fmt.Errorf("no data to copy")
		}
		out, _ := json.Marshal(ctx.LastData)
		content = string(out)
	} else {
		arg := args[0]
		// Check if it's a variable
		if val, ok := ctx.Vars[arg]; ok {
			out, _ := json.Marshal(val)
			content = string(out)
		} else {
			// Check if it's a request ID
			session := ctx.Tree.Current()
			if session != nil {
				req, ok := session.GetRequest(arg)
				if ok && req.Response != nil {
					content = string(req.Response.RawBody)
				} else {
					return fmt.Errorf("request %s not found or not executed", arg)
				}
			} else {
				return fmt.Errorf("unknown argument: %s", arg)
			}
		}
	}

	// Try clipboard backends
	clipCmd := exec.Command("pbcopy")
	clipCmd.Stdin = bytes.NewReader([]byte(content))
	if clipCmd.Run() == nil {
		fmt.Printf("Copied %d characters\n", len(content))
		return nil
	}

	clipCmd = exec.Command("wl-copy")
	clipCmd.Stdin = bytes.NewReader([]byte(content))
	if clipCmd.Run() == nil {
		fmt.Printf("Copied %d characters\n", len(content))
		return nil
	}

	clipCmd = exec.Command("xclip", "-selection", "clipboard")
	clipCmd.Stdin = bytes.NewReader([]byte(content))
	if clipCmd.Run() == nil {
		fmt.Printf("Copied %d characters\n", len(content))
		return nil
	}

	clipCmd = exec.Command("xsel", "--clipboard", "--input")
	clipCmd.Stdin = bytes.NewReader([]byte(content))
	if clipCmd.Run() == nil {
		fmt.Printf("Copied %d characters\n", len(content))
		return nil
	}

	return fmt.Errorf("no clipboard backend found (tried: pbcopy, wl-copy, xclip, xsel)")
}

func (c *clipCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	var names []string
	for k := range ctx.Vars {
		names = append(names, k)
	}
	session := ctx.Tree.Current()
	if session != nil {
		for _, req := range session.Requests {
			names = append(names, req.ID)
		}
	}
	return names
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
		// Replay all
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
		// Replay specific request
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
		// Try Content-Disposition
		if cd := ctx.LastResp.Headers["Content-Disposition"]; cd != "" {
			// Extract filename from Content-Disposition
			parts := strings.Split(cd, "filename=")
			if len(parts) > 1 {
				filename = strings.Trim(parts[1], "\"")
			}
		}
	}

	if filename == "" {
		// Generate default filename
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

	err := os.WriteFile(filename, ctx.LastResp.RawBody, 0644)
	if err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}

	fmt.Printf("Saved to: %s\n", filename)
	return nil
}

func (c *saveCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	return nil
}

func init() {
	repl.Register(&jqCmd{})
	repl.Register(&updateCmd{})
	repl.Register(&editCmd{})
	repl.Register(&clipCmd{})
	repl.Register(&replayCmd{})
	repl.Register(&watchCmd{})
	repl.Register(&saveCmd{})
}
