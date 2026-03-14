package repl

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/chzyer/readline"
	openapi3 "github.com/getkin/kin-openapi/openapi3"
	"httpclient/internal/executor"
	"httpclient/internal/input"
	"httpclient/internal/model"
	"httpclient/internal/session"
)

type ShellContext struct {
	Tree      *model.SessionTree
	Executor  *executor.Client
	OpenAPI   *openapi3.T
	Vars      model.Variables
	LastResp  *model.Response
	LastData  any
	LastReqID string
	Pending   string
	Readline  *readline.Instance
}

func (ctx *ShellContext) CurrentSpec() *model.OpenAPISpec {
	session := ctx.Tree.Current()
	if session == nil {
		return nil
	}
	return session.OpenAPISpec
}

func (ctx *ShellContext) SetSpec(spec *input.Spec) {
	session := ctx.Tree.Current()
	if session == nil {
		return
	}
	session.OpenAPISpec = &model.OpenAPISpec{
		Title:   spec.Title,
		Version: spec.Version,
		Routes:  make([]model.Route, len(spec.Routes)),
	}
	for i, r := range spec.Routes {
		session.OpenAPISpec.Routes[i] = model.Route{
			Method:  r.Method,
			Path:    r.Path,
			Summary: r.Summary,
			Tags:    r.Tags,
			Params:  make([]model.Parameter, len(r.Params)),
		}
		for j, p := range r.Params {
			session.OpenAPISpec.Routes[i].Params[j] = model.Parameter{
				Name:     p.Name,
				In:       p.In,
				Required: p.Required,
			}
		}
	}
}

func NewShellContext() *ShellContext {
	tree, err := session.LoadTree()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load tree: %v\n", err)
		tree = model.NewSessionTree()
	}

	return &ShellContext{
		Tree:     tree,
		Executor: executor.NewClient(30 * 1000 * 1000 * 1000), // 30 seconds
		Vars:     make(model.Variables),
	}
}

func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	if path[:1] == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot find home directory: %w", err)
		}
		path = home + path[1:]
	}
	return path, nil
}

func (ctx *ShellContext) Save() error {
	if err := session.SaveTree(ctx.Tree); err != nil {
		return err
	}
	return session.SaveEnvironments(ctx.Tree.Environments)
}

func (ctx *ShellContext) Run() error {
	cfg, _ := session.LoadConfig()
	historyFile, _ := ExpandPath(cfg.HistoryFile)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          BuildPrompt(ctx),
		HistoryFile:     historyFile,
		AutoComplete:    NewCompleter(ctx),
		InterruptPrompt: "^C",
		EOFPrompt:       "^D",
	})
	if err != nil {
		return err
	}
	defer rl.Close()

	ctx.Readline = rl // ← wire it up so editorCmd.Run can call WriteStdin

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				fmt.Println("^C")
				continue
			}
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", r)
				}
			}()
			if err := ctx.handleLine(line); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
		}()

		rl.SetPrompt(BuildPrompt(ctx))
	}

	return ctx.Save()
}

func (ctx *ShellContext) handleLine(line string) error {
	if strings.HasPrefix(line, "!") {
		return ctx.handleShell(line[1:])
	}

	if strings.HasPrefix(line, "/") {
		pending, err := Dispatch(ctx, strings.TrimPrefix(line, "/"))
		if err != nil {
			return err
		}
		if pending != "" {
			ctx.Pending = pending
		}
		return nil
	}

	if idx := strings.Index(line, "="); idx > 0 {
		lhs := strings.TrimSpace(line[:idx])
		rhs := strings.TrimSpace(line[idx+1:])
		if isValidVarName(lhs) {
			return ctx.handleAssignment(lhs, rhs)
		}
	}

	return nil
}

func isValidVarName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_') {
				return false
			}
		} else {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
				return false
			}
		}
	}
	return true
}

func (ctx *ShellContext) handleAssignment(lhs, rhs string) error {
	if rhs == "d" || rhs == "last" {
		ctx.Vars.Set(lhs, ctx.LastData, model.VarScopeShell)
		return nil
	}

	if strings.HasPrefix(rhs, "/") {
		pending, err := Dispatch(ctx, strings.TrimPrefix(rhs, "/"))
		if err != nil {
			return err
		}
		if pending != "" {
			ctx.Pending = pending
		}
		return nil
	}

	ctx.Vars.Set(lhs, rhs, model.VarScopeShell)
	return nil
}

func (ctx *ShellContext) handleShell(cmd string) error {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil
	}

	// Convert Variables to map[string]any for ResolveVars
	varsMap := make(map[string]any)
	for k, v := range ctx.Vars {
		varsMap[k] = v.Value
	}
	resolved, _ := model.ResolveVars(cmd, varsMap)

	parts, err := splitShellCommand(resolved)
	if err != nil {
		return err
	}

	execCmd := exec.Command(parts[0], parts[1:]...)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	return execCmd.Run()
}

func splitShellCommand(cmd string) ([]string, error) {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := ' '

	for _, r := range cmd {
		if !inQuote && (r == '"' || r == '\'') {
			inQuote = true
			quoteChar = r
		} else if inQuote && r == quoteChar {
			inQuote = false
		} else if !inQuote && r == ' ' {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	return parts, nil
}
