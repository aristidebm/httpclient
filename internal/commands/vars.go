package commands

import (
	"flag"
	"fmt"
	"strings"

	"httpclient/internal/repl"
)

type varsCmd struct{}

func (c *varsCmd) Name() string      { return "vars" }
func (c *varsCmd) Aliases() []string { return nil }
func (c *varsCmd) Help() string {
	return "Manage variables: /vars [list|set|unset] [flags] [key] [value]"
}

func (c *varsCmd) Run(ctx *repl.ShellContext, args []string) error {
	if len(args) < 1 {
		return c.listVars(ctx, "session")
	}

	subcmd := args[0]

	// Parse remaining args
	fs := flag.NewFlagSet("vars", flag.ContinueOnError)
	_ = fs.Bool("session", false, "Session scope (default)") // default, kept for completeness
	envScope := fs.Bool("env", false, "Environment scope")
	shellScope := fs.Bool("shell", false, "Shell scope")
	fs.Parse(args[1:])

	// Determine scope
	scope := "session" // default
	if *envScope {
		scope = "env"
	} else if *shellScope {
		scope = "shell"
	}

	switch subcmd {
	case "list", "ls", "l":
		return c.listVars(ctx, scope)

	case "set", "s":
		key := fs.Arg(0)
		value := fs.Arg(1)
		if key == "" || value == "" {
			return fmt.Errorf("usage: /vars set [--session|--env|--shell] <key> <value>")
		}
		return c.setVar(ctx, scope, key, value)

	case "unset", "delete", "u":
		key := fs.Arg(0)
		if key == "" {
			return fmt.Errorf("usage: /vars unset [--session|--env|--shell] <key>")
		}
		return c.unsetVar(ctx, scope, key)

	case "get", "g":
		key := fs.Arg(0)
		if key == "" {
			return fmt.Errorf("usage: /vars get <key>")
		}
		return c.getVar(ctx, key)

	default:
		return fmt.Errorf("unknown subcommand: %s (use list, set, unset, or get)", subcmd)
	}
}

func (c *varsCmd) listVars(ctx *repl.ShellContext, scope string) error {
	fmt.Println("=== Shell Variables ===")
	if len(ctx.Vars) == 0 {
		fmt.Println("  (none)")
	} else {
		for k, v := range ctx.Vars {
			fmt.Printf("  %s = %v\n", k, v)
		}
	}

	env := ctx.Tree.CurrentEnv()
	if env != nil && len(env.Vars) > 0 {
		fmt.Printf("\n=== Environment Variables (%s) ===\n", env.Name)
		for k, v := range env.Vars {
			fmt.Printf("  %s = %v\n", k, v)
		}
	}

	session := ctx.Tree.Current()
	if session != nil && len(session.VarOverrides) > 0 {
		fmt.Printf("\n=== Session Variables (%s) ===\n", session.Name)
		for k, v := range session.VarOverrides {
			fmt.Printf("  %s = %v\n", k, v)
		}
	}

	return nil
}

func (c *varsCmd) setVar(ctx *repl.ShellContext, scope, key, value string) error {
	session := ctx.Tree.Current()

	switch scope {
	case "session":
		if session == nil {
			return fmt.Errorf("no current session")
		}
		if session.VarOverrides == nil {
			session.VarOverrides = make(map[string]any)
		}
		session.VarOverrides[key] = value
		repl.PrintSuccess(fmt.Sprintf("Set %s = %s (session)", key, value))

	case "env":
		env := ctx.Tree.CurrentEnv()
		if env == nil {
			return fmt.Errorf("no current environment")
		}
		env.Vars[key] = value
		repl.PrintSuccess(fmt.Sprintf("Set %s = %s (env: %s)", key, value, env.Name))

	case "shell":
		ctx.Vars[key] = value
		repl.PrintSuccess(fmt.Sprintf("Set %s = %s (shell)", key, value))
	}

	return nil
}

func (c *varsCmd) getVar(ctx *repl.ShellContext, key string) error {
	// Check shell vars first
	if v, ok := ctx.Vars[key]; ok {
		fmt.Printf("shell.%s = %v\n", key, v)
		return nil
	}

	// Check session overrides
	session := ctx.Tree.Current()
	if session != nil {
		if v, ok := session.VarOverrides[key]; ok {
			fmt.Printf("session.%s = %v\n", key, v)
			return nil
		}
	}

	// Check environment vars
	env := ctx.Tree.CurrentEnv()
	if env != nil {
		if v, ok := env.Vars[key]; ok {
			fmt.Printf("env.%s = %v\n", key, v)
			return nil
		}
	}

	return fmt.Errorf("variable %q not found", key)
}

func (c *varsCmd) unsetVar(ctx *repl.ShellContext, scope, key string) error {
	session := ctx.Tree.Current()

	switch scope {
	case "session":
		if session != nil && session.VarOverrides != nil {
			if _, ok := session.VarOverrides[key]; ok {
				delete(session.VarOverrides, key)
				repl.PrintSuccess(fmt.Sprintf("Unset %s (session)", key))
				return nil
			}
		}
		return fmt.Errorf("variable %q not found in session", key)

	case "env":
		env := ctx.Tree.CurrentEnv()
		if env != nil {
			if _, ok := env.Vars[key]; ok {
				delete(env.Vars, key)
				repl.PrintSuccess(fmt.Sprintf("Unset %s (env: %s)", key, env.Name))
				return nil
			}
		}
		return fmt.Errorf("variable %q not found in environment", key)

	case "shell":
		if _, ok := ctx.Vars[key]; ok {
			delete(ctx.Vars, key)
			repl.PrintSuccess(fmt.Sprintf("Unset %s (shell)", key))
			return nil
		}
		return fmt.Errorf("variable %q not found in shell", key)
	}

	return nil
}

func (c *varsCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	fields := strings.Fields(partial)

	if len(fields) == 0 {
		return []string{"list", "set", "unset", "get"}
	}

	if len(fields) == 1 {
		return []string{"list", "set", "unset", "get"}
	}

	// Complete flags
	if strings.HasPrefix(fields[len(fields)-1], "-") {
		return []string{"--session", "--env", "--shell"}
	}

	// Complete variable names
	var names []string
	for k := range ctx.Vars {
		names = append(names, k)
	}
	session := ctx.Tree.Current()
	if session != nil {
		for k := range session.VarOverrides {
			names = append(names, k)
		}
	}
	env := ctx.Tree.CurrentEnv()
	if env != nil {
		for k := range env.Vars {
			names = append(names, k)
		}
	}
	return names
}

func init() {
	repl.Register(&varsCmd{})
}
