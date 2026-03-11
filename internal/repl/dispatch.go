package repl

import (
	"fmt"
	"strings"
)

type Command interface {
	Name() string
	Aliases() []string
	Run(ctx *ShellContext, args []string) error
	Complete(ctx *ShellContext, partial string) []string
	Help() string
}

var commands = make(map[string]Command)

func Register(cmd Command) {
	commands[cmd.Name()] = cmd
	for _, alias := range cmd.Aliases() {
		commands[alias] = cmd
	}
}

func Dispatch(ctx *ShellContext, line string) error {
	parts := tokenize(line)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	name := strings.ToLower(parts[0])
	args := parts[1:]

	cmd, ok := commands[name]
	if !ok {
		return fmt.Errorf("unknown command: /%s — type /help", name)
	}

	return cmd.Run(ctx, args)
}

func tokenize(line string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := ' '

	for _, r := range line {
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

	return parts
}

func GetCommand(name string) (Command, bool) {
	cmd, ok := commands[strings.ToLower(name)]
	return cmd, ok
}

func ListCommands() []Command {
	seen := make(map[string]bool)
	var result []Command
	for _, cmd := range commands {
		name := cmd.Name()
		if !seen[name] {
			seen[name] = true
			result = append(result, cmd)
		}
	}
	return result
}

type helpCmd struct{}

func (c *helpCmd) Name() string      { return "help" }
func (c *helpCmd) Aliases() []string { return nil }
func (c *helpCmd) Help() string      { return "Show help information" }

func (c *helpCmd) Run(ctx *ShellContext, args []string) error {
	if len(args) > 0 {
		cmd, ok := GetCommand(args[0])
		if !ok {
			return fmt.Errorf("unknown command: %s", args[0])
		}
		fmt.Println(cmd.Help())
		return nil
	}

	fmt.Println("Available commands:")
	for _, cmd := range ListCommands() {
		fmt.Printf("  /%s - %s\n", cmd.Name(), cmd.Help())
	}
	return nil
}

func (c *helpCmd) Complete(ctx *ShellContext, partial string) []string {
	return nil
}

type exitCmd struct{}

func (c *exitCmd) Name() string      { return "exit" }
func (c *exitCmd) Aliases() []string { return []string{"quit", "q"} }
func (c *exitCmd) Help() string      { return "Exit the shell (saves session)" }

func (c *exitCmd) Run(ctx *ShellContext, args []string) error {
	return ctx.Save()
}

func (c *exitCmd) Complete(ctx *ShellContext, partial string) []string {
	return nil
}

type infoCmd struct{}

func (c *infoCmd) Name() string      { return "info" }
func (c *infoCmd) Aliases() []string { return nil }
func (c *infoCmd) Help() string      { return "Show current shell state" }

func (c *infoCmd) Run(ctx *ShellContext, args []string) error {
	session := ctx.Tree.Current()
	env := ctx.Tree.CurrentEnv()

	fmt.Printf("SESSION  : %s\n", session.Name)
	if env != nil {
		fmt.Printf("ENV      : %s (%s)\n", env.Name, env.BaseURL)
	} else {
		fmt.Printf("ENV      : (none)\n")
	}

	if ctx.LastReqID != "" && ctx.LastResp != nil {
		fmt.Printf("LAST REQ : %s → %d (%s)\n", ctx.LastReqID, ctx.LastResp.StatusCode, ctx.LastResp.Status)
	}

	fmt.Printf("VARS     : %d defined\n", len(ctx.Vars))
	return nil
}

func (c *infoCmd) Complete(ctx *ShellContext, partial string) []string {
	return nil
}

func init() {
	Register(&helpCmd{})
	Register(&exitCmd{})
	Register(&infoCmd{})
}
