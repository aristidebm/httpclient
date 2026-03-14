package commands

import (
	"fmt"
	"strings"

	"httpclient/internal/model"
	"httpclient/internal/repl"
)

type varsCmd struct{}

func (c *varsCmd) Name() string      { return "vars" }
func (c *varsCmd) Aliases() []string { return nil }
func (c *varsCmd) Help() string {
	return "Manage variables: /vars [list|set|unset] [flags] [key] [value]"
}

func (c *varsCmd) Run(ctx *repl.ShellContext, args []string) error {
	scope := ""

	// Find subcommand position
	subcmdIdx := -1
	for i, arg := range args {
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			subcmdIdx = i
			break
		}
	}

	// Parse flags before subcommand
	for i := 0; i < subcmdIdx && subcmdIdx > 0; i++ {
		flag := args[i]
		if flag == "--session" || flag == "-session" {
			scope = "session"
		} else if flag == "--env" || flag == "-env" {
			scope = "env"
		} else if flag == "--shell" || flag == "-shell" {
			scope = "shell"
		}
	}

	// If no subcommand found, just list
	if subcmdIdx == -1 || subcmdIdx >= len(args) {
		return c.listVars(ctx, scope)
	}

	subcmd := args[subcmdIdx]
	rem := args[subcmdIdx+1:]

	// Parse flags after subcommand
	for _, flag := range rem {
		if flag == "--session" || flag == "-session" {
			scope = "session"
		} else if flag == "--env" || flag == "-env" {
			scope = "env"
		} else if flag == "--shell" || flag == "-shell" {
			scope = "shell"
		}
	}

	// Filter rem to get only non-flag args
	var nonFlagArgs []string
	for _, arg := range rem {
		if !strings.HasPrefix(arg, "-") {
			nonFlagArgs = append(nonFlagArgs, arg)
		}
	}

	key := ""
	value := ""

	switch subcmd {
	case "list", "ls", "l":
		return c.listVars(ctx, scope)

	case "set", "s":
		if len(nonFlagArgs) < 2 {
			return fmt.Errorf("usage: /vars set [--session|--env|--shell] <key> <value>")
		}
		key = nonFlagArgs[0]
		value = nonFlagArgs[1]
		// Use scope from flag, or default to session if not specified
		if scope == "" {
			scope = "session"
		}
		return c.setVar(ctx, scope, key, value)

	case "unset", "delete", "u":
		if len(nonFlagArgs) < 1 {
			return fmt.Errorf("usage: /vars unset [--session|--env|--shell] <key>")
		}
		key = nonFlagArgs[0]
		if scope == "" {
			scope = "session"
		}
		return c.unsetVar(ctx, scope, key)

	case "get", "g":
		if len(nonFlagArgs) < 1 {
			return fmt.Errorf("usage: /vars get <key>")
		}
		key = nonFlagArgs[0]
		return c.getVar(ctx, key)

	default:
		return fmt.Errorf("unknown subcommand: %s (use list, set, unset, or get)", subcmd)
	}
}

func (c *varsCmd) listVars(ctx *repl.ShellContext, filterScope string) error {
	type varEntry struct {
		key    string
		value  string
		scope  string
		active bool
	}

	var entries []varEntry

	// Shell vars
	if filterScope == "" || filterScope == "shell" {
		for k, v := range ctx.Vars {
			entries = append(entries, varEntry{
				key:    k,
				value:  fmt.Sprintf("%v", v),
				scope:  "shell",
				active: true,
			})
		}
	}

	// Environment vars
	env := ctx.Tree.CurrentEnv()
	envVars := make(map[string]bool)
	if env != nil && (filterScope == "" || filterScope == "env") {
		for k, v := range env.Vars {
			envVars[k] = true
			active := true
			session := ctx.Tree.Current()
			if session != nil {
				if _, ok := session.Vars[k]; ok {
					active = false
				}
			}

			entries = append(entries, varEntry{
				key:    k,
				value:  fmt.Sprintf("%v", v.Value),
				scope:  "env:" + env.Name,
				active: active,
			})
		}
	}

	// Session vars
	session := ctx.Tree.Current()
	if session != nil && (filterScope == "" || filterScope == "session") {
		for k, v := range session.Vars {
			entries = append(entries, varEntry{
				key:    k,
				value:  fmt.Sprintf("%v", v.Value),
				scope:  "session",
				active: true,
			})
		}
	}

	// Print header
	fmt.Println("KEY             VALUE                          SCOPE")
	fmt.Println("───────────────────────────────────────────────────────────────")

	// Print all entries
	for _, e := range entries {
		scopeDisplay := e.scope
		if !e.active {
			scopeDisplay += " (shadowed)"
		}
		fmt.Printf("%-15s %-30s %s\n", e.key, e.value, scopeDisplay)
	}

	if len(entries) == 0 {
		fmt.Println("(none)")
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
		if session.Vars == nil {
			session.Vars = make(model.Variables)
		}
		session.Vars.Set(key, value, model.VarScopeSession)
		repl.PrintSuccess(fmt.Sprintf("Set %s = %s (session)", key, value))

	case "env":
		env := ctx.Tree.CurrentEnv()
		if env == nil {
			return fmt.Errorf("no current environment")
		}
		if env.Vars == nil {
			env.Vars = make(model.Variables)
		}
		env.Vars.Set(key, value, model.VarScopeEnv)
		repl.PrintSuccess(fmt.Sprintf("Set %s = %s (env: %s)", key, value, env.Name))

	case "shell":
		if ctx.Vars == nil {
			ctx.Vars = make(model.Variables)
		}
		ctx.Vars.Set(key, value, model.VarScopeShell)
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
		if v, ok := session.Vars[key]; ok {
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
		if session != nil && session.Vars != nil {
			if _, ok := session.Vars[key]; ok {
				delete(session.Vars, key)
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
		for k := range session.Vars {
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
