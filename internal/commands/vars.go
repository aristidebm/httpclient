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
func (c *varsCmd) Help() string      { return "Manage variables: /vars [flags] [key] [value]" }

func (c *varsCmd) Run(ctx *repl.ShellContext, args []string) error {
	fs := flag.NewFlagSet("vars", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Println("Usage: /vars [flags] [key] [value]")
		fmt.Println("Flags:")
		fmt.Println("  --session  Variables scoped to current session (default)")
		fmt.Println("  --env      Variables scoped to current environment")
		fmt.Println("  --shell    Shell-level variables")
		fmt.Println("\nExamples:")
		fmt.Println("  /vars                    # list all variables")
		fmt.Println("  /vars foo                # get value of 'foo'")
		fmt.Println("  /vars foo bar            # set foo=bar in session scope")
		fmt.Println("  /vars --env foo bar      # set foo=bar in environment")
		fmt.Println("  /vars --shell foo bar    # set foo=bar in shell")
		fmt.Println("  /vars --unset foo        # unset foo (from session)")
	}

	sessionScope := fs.Bool("session", false, "Session scope (default)")
	envScope := fs.Bool("env", false, "Environment scope")
	shellScope := fs.Bool("shell", false, "Shell scope")
	unset := fs.Bool("unset", false, "Unset variable")

	_ = sessionScope // default scope, kept for flag completeness

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Determine scope
	scope := "session" // default
	if *envScope {
		scope = "env"
	} else if *shellScope {
		scope = "shell"
	}

	key := fs.Arg(0)
	value := fs.Arg(1)

	// Just list if no arguments
	if key == "" && !*unset {
		return c.listVars(ctx, scope)
	}

	// Unset
	if *unset {
		if key == "" {
			return fmt.Errorf("key required for unset")
		}
		return c.unsetVar(ctx, scope, key)
	}

	// Get
	if value == "" {
		return c.getVar(ctx, key)
	}

	// Set
	return c.setVar(ctx, scope, key, value)
}

func (c *varsCmd) listVars(ctx *repl.ShellContext, defaultScope string) error {
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

	if len(fields) <= 1 {
		return []string{"--session", "--env", "--shell", "--unset"}
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
