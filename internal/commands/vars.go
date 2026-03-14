package commands

import (
	"fmt"
	"strings"

	"httpclient/internal/repl"
)

type varsCmd struct{}

func (c *varsCmd) Name() string      { return "vars" }
func (c *varsCmd) Aliases() []string { return nil }
func (c *varsCmd) Help() string {
	return "Manage variables: /vars [list|set|unset|get] [scope] [key] [value]"
}

func (c *varsCmd) Run(ctx *repl.ShellContext, args []string) error {
	if len(args) < 1 {
		return c.listVars(ctx, "")
	}

	subcmd := args[0]

	switch subcmd {
	case "list", "ls", "l":
		scope := ""
		if len(args) > 1 {
			scope = args[1]
		}
		return c.listVars(ctx, scope)

	case "set", "s":
		if len(args) < 3 {
			return fmt.Errorf("usage: /vars set [session|env] <key> <value>")
		}
		scope := "session"
		key := args[1]
		value := args[2]
		if len(args) > 3 {
			scope = args[1]
			key = args[2]
			value = args[3]
		}
		return c.setVar(ctx, scope, key, value)

	case "get", "g":
		if len(args) < 2 {
			return fmt.Errorf("usage: /vars get <key>")
		}
		return c.getVar(ctx, args[1])

	case "unset", "u", "delete", "d":
		if len(args) < 2 {
			return fmt.Errorf("usage: /vars unset <key>")
		}
		return c.unsetVar(ctx, args[1])

	default:
		return fmt.Errorf("unknown subcommand: %s", subcmd)
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

	if scope == "" || scope == "env" {
		env := ctx.Tree.CurrentEnv()
		if env != nil && len(env.Vars) > 0 {
			fmt.Println("\n=== Environment Variables (", env.Name, ") ===")
			for k, v := range env.Vars {
				fmt.Printf("  %s = %v\n", k, v)
			}
		}
	}

	if scope == "" || scope == "session" {
		session := ctx.Tree.Current()
		if session != nil && len(session.VarOverrides) > 0 {
			fmt.Println("\n=== Session Overrides (", session.Name, ") ===")
			for k, v := range session.VarOverrides {
				fmt.Printf("  %s = %v\n", k, v)
			}
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
		repl.PrintSuccess(fmt.Sprintf("Set session variable %s = %s", key, value))

	case "env":
		env := ctx.Tree.CurrentEnv()
		if env == nil {
			return fmt.Errorf("no current environment")
		}
		env.Vars[key] = value
		repl.PrintSuccess(fmt.Sprintf("Set environment variable %s = %s (in %s)", key, value, env.Name))

	default:
		// Shell variable (default)
		ctx.Vars[key] = value
		repl.PrintSuccess(fmt.Sprintf("Set shell variable %s = %s", key, value))
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

func (c *varsCmd) unsetVar(ctx *repl.ShellContext, key string) error {
	session := ctx.Tree.Current()

	// Try shell vars first
	if _, ok := ctx.Vars[key]; ok {
		delete(ctx.Vars, key)
		repl.PrintSuccess(fmt.Sprintf("Unset shell variable %s", key))
		return nil
	}

	// Try session overrides
	if session != nil && session.VarOverrides != nil {
		if _, ok := session.VarOverrides[key]; ok {
			delete(session.VarOverrides, key)
			repl.PrintSuccess(fmt.Sprintf("Unset session variable %s", key))
			return nil
		}
	}

	// Try environment vars
	env := ctx.Tree.CurrentEnv()
	if env != nil {
		if _, ok := env.Vars[key]; ok {
			delete(env.Vars, key)
			repl.PrintSuccess(fmt.Sprintf("Unset environment variable %s (in %s)", key, env.Name))
			return nil
		}
	}

	return fmt.Errorf("variable %q not found", key)
}

func (c *varsCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	fields := strings.Fields(partial)

	if len(fields) == 0 {
		return []string{"list", "set", "get", "unset"}
	}

	if len(fields) == 1 {
		return []string{"list", "set", "get", "unset"}
	}

	if len(fields) == 2 {
		subcmd := fields[0]
		if subcmd == "set" {
			return []string{"session", "env"}
		}
		if subcmd == "get" || subcmd == "unset" {
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
	}

	return nil
}

func init() {
	repl.Register(&varsCmd{})
}
