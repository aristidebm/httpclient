package commands

import (
	"fmt"
	"strings"
	"time"

	"httpclient/internal/model"
	"httpclient/internal/repl"
)

type varsCmd struct{}

func (c *varsCmd) Name() string      { return "vars" }
func (c *varsCmd) Aliases() []string { return nil }
func (c *varsCmd) Help() string {
	return "Manage variables: /vars [list|set|unset|clear] [flags] [key] [value]"
}

func (c *varsCmd) Run(ctx *repl.ShellContext, args []string) error {
	scope := ""
	isPublic := false

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
		} else if flag == "--shell" || flag == "-shell" {
			scope = "shell"
		} else if flag == "--public" || flag == "-public" {
			isPublic = true
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
		} else if flag == "--shell" || flag == "-shell" {
			scope = "shell"
		} else if flag == "--public" || flag == "-public" {
			isPublic = true
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
		if len(nonFlagArgs) < 1 {
			return fmt.Errorf("usage: /vars set [--session|--shell] <key> [value] or <key>=<value>")
		}
		arg := nonFlagArgs[0]
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			key = parts[0]
			value = parts[1]
		} else {
			key = arg
			if len(nonFlagArgs) >= 2 {
				value = nonFlagArgs[1]
			} else {
				value = ""
			}
		}
		// Use scope from flag, or default to session if not specified
		if scope == "" {
			scope = "session"
		}
		return c.setVar(ctx, scope, key, value, isPublic)

	case "unset", "delete", "u":
		if len(nonFlagArgs) < 1 {
			return fmt.Errorf("usage: /vars unset [--session|--shell] <key>")
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

	case "clear", "c":
		// Check for --all flag
		clearAll := false
		for _, arg := range rem {
			if arg == "--all" || arg == "-all" {
				clearAll = true
				break
			}
		}
		return c.clearVars(ctx, scope, clearAll)

	default:
		return fmt.Errorf("unknown subcommand: %s (use list, set, unset, or get)", subcmd)
	}
}

func (c *varsCmd) listVars(ctx *repl.ShellContext, filterScope string) error {
	type varEntry struct {
		key     string
		value   string
		scope   string
		active  bool
		public  bool
		updated time.Time
	}

	var entries []varEntry

	// Shell vars
	if filterScope == "" || filterScope == "shell" {
		for _, v := range ctx.Vars.ListPublic() {
			entries = append(entries, varEntry{
				key:     v.Name,
				value:   fmt.Sprintf("%v", v.Value),
				scope:   "shell",
				active:  true,
				public:  v.Public,
				updated: v.Updated,
			})
		}
	}

	// Inherited vars (from session tree)
	session := ctx.Tree.Current()
	if session != nil && (filterScope == "" || filterScope == "inherited") {
		inheritedVars := ctx.Tree.GetEffectiveVars(session.ID)
		sessionVars := make(map[string]bool)
		for k := range session.Vars {
			sessionVars[k] = true
		}

		for k, v := range inheritedVars {
			active := !sessionVars[k] // inactive if overridden by session
			entries = append(entries, varEntry{
				key:     k,
				value:   fmt.Sprintf("%v", v.Value),
				scope:   "inherited",
				active:  active,
				public:  v.Public,
				updated: v.Updated,
			})
		}
	}

	// Session vars
	if session != nil && (filterScope == "" || filterScope == "session") {
		for _, v := range session.Vars.ListPublic() {
			entries = append(entries, varEntry{
				key:     v.Name,
				value:   fmt.Sprintf("%v", v.Value),
				scope:   "session",
				active:  true,
				public:  v.Public,
				updated: v.Updated,
			})
		}
	}

	// Print all entries
	for _, e := range entries {
		scopeDisplay := e.scope
		if !e.active {
			scopeDisplay += " (shadowed)"
		}
		fmt.Printf("%-15s %-30s %-12s %s\n", e.key, e.value, e.updated.Format("2006-01-02"), scopeDisplay)
	}

	if len(entries) == 0 {
		fmt.Println("(none)")
	}

	return nil
}

func (c *varsCmd) setVar(ctx *repl.ShellContext, scope, key, value string, isPublic bool) error {
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

	// Check session vars
	session := ctx.Tree.Current()
	if session != nil {
		if v, ok := session.Vars[key]; ok {
			fmt.Printf("session.%s = %v\n", key, v)
			return nil
		}
	}

	// Check inherited vars
	if session != nil {
		inheritedVars := ctx.Tree.GetEffectiveVars(session.ID)
		if v, ok := inheritedVars[key]; ok {
			fmt.Printf("inherited.%s = %v\n", key, v.Value)
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

func (c *varsCmd) clearVars(ctx *repl.ShellContext, scope string, clearAll bool) error {
	if scope == "" {
		scope = "session"
	}

	if clearAll {
		session := ctx.Tree.Current()
		if session != nil {
			session.Vars = make(model.Variables)
		}
		ctx.Vars = make(model.Variables)
		repl.PrintSuccess("Cleared all variables (session, shell)")
		return nil
	}

	switch scope {
	case "session":
		session := ctx.Tree.Current()
		if session == nil {
			return fmt.Errorf("no current session")
		}
		session.Vars = make(model.Variables)
		repl.PrintSuccess("Cleared all session variables")

	case "shell":
		ctx.Vars = make(model.Variables)
		repl.PrintSuccess("Cleared all shell variables")

	default:
		return fmt.Errorf("unknown scope: %s (use: session, shell, or --all)", scope)
	}

	return nil
}

func (c *varsCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	fields := strings.Fields(partial)

	if len(fields) == 0 {
		return []string{"list", "set", "unset", "get", "clear"}
	}

	if len(fields) == 1 {
		return []string{"list", "set", "unset", "get", "clear"}
	}

	// Complete flags
	if strings.HasPrefix(fields[len(fields)-1], "-") {
		return []string{"--session", "--shell"}
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
		// Add inherited vars
		inheritedVars := ctx.Tree.GetEffectiveVars(session.ID)
		for k := range inheritedVars {
			names = append(names, k)
		}
	}
	return names
}

func init() {
	repl.Register(&varsCmd{})
}
