package commands

import (
	"fmt"
	"strings"

	"httpclient/internal/model"
	"httpclient/internal/repl"
)

type envCmd struct{}

func (c *envCmd) Name() string      { return "env" }
func (c *envCmd) Aliases() []string { return nil }
func (c *envCmd) Help() string      { return "Manage environments: new, list, show, copy, drop, rename" }

func (c *envCmd) Run(ctx *repl.ShellContext, args []string) error {
	if len(args) == 0 {
		return c.Run(ctx, []string{"list"})
	}

	subcmd := args[0]
	switch subcmd {
	case "new":
		return envNew(ctx, args[1:])
	case "set":
		return envSet(ctx, args[1:])
	case "unset":
		return envUnset(ctx, args[1:])
	case "list":
		return envList(ctx)
	case "show":
		return envShow(ctx, args[1:])
	case "copy":
		return envCopy(ctx, args[1:])
	case "drop":
		return envDrop(ctx, args[1:])
	case "rename":
		return envRename(ctx, args[1:])
	default:
		return fmt.Errorf("unknown env subcommand: %s", subcmd)
	}
}

func (c *envCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	fields := strings.Fields(partial)

	if len(fields) == 1 {
		return []string{"new", "set", "unset", "list", "show", "copy", "drop", "rename"}
	}

	subcmd := fields[0]
	lastArg := ""
	if len(fields) > 1 {
		lastArg = fields[len(fields)-1]
	}

	switch subcmd {
	case "set", "unset", "show", "copy", "drop", "rename":
		// Complete environment names
		var names []string
		for name := range ctx.Tree.Environments {
			if strings.HasPrefix(name, lastArg) {
				names = append(names, name)
			}
		}
		return names
	}

	return nil
}

func envNew(ctx *repl.ShellContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: /env new <name> <base-url>")
	}

	name := args[0]
	baseURL := args[1]

	if _, ok := ctx.Tree.Environments[name]; ok {
		return fmt.Errorf("environment %q already exists", name)
	}

	env := &model.Environment{
		Name:    name,
		BaseURL: baseURL,
		Headers: make(map[string]string),
		Vars:    make(model.Variables),
	}
	env.SetBaseURL(baseURL)

	ctx.Tree.Environments[name] = env
	repl.PrintSuccess(fmt.Sprintf("Created environment %q with base URL %q", name, baseURL))
	return nil
}

func envSet(ctx *repl.ShellContext, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: /env set <name> <key> <value>")
	}

	name := args[0]
	key := args[1]
	value := args[2]

	env, ok := ctx.Tree.Environments[name]
	if !ok {
		return fmt.Errorf("environment %q not found", name)
	}

	// Special case: update base URL with "url" or "baseurl" key
	if key == "url" || key == "baseurl" {
		env.SetBaseURL(value)
		repl.PrintSuccess(fmt.Sprintf("Set base URL to %q in environment %q", value, name))
		return nil
	}

	// Check if key starts with uppercase - treat as header
	if strings.HasPrefix(key, strings.ToUpper(key[:1])) {
		env.Headers[key] = value
		repl.PrintSuccess(fmt.Sprintf("Set header %s=%s in environment %q", key, value, name))
	} else {
		env.Vars.Set(key, value, model.VarScopeEnv)
		repl.PrintSuccess(fmt.Sprintf("Set var %s=%s in environment %q", key, value, name))
	}

	return nil
}

func envUnset(ctx *repl.ShellContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: /env unset <name> <key>")
	}

	name := args[0]
	key := args[1]

	env, ok := ctx.Tree.Environments[name]
	if !ok {
		return fmt.Errorf("environment %q not found", name)
	}

	// Check if it's a header
	if _, ok := env.Headers[key]; ok {
		delete(env.Headers, key)
		repl.PrintSuccess(fmt.Sprintf("Removed header %s from environment %q", key, name))
		return nil
	}

	// Check if it's a var
	if _, ok := env.Vars[key]; ok {
		delete(env.Vars, key)
		repl.PrintSuccess(fmt.Sprintf("Removed var %s from environment %q", key, name))
		return nil
	}

	return fmt.Errorf("key %q not found in environment %q", key, name)
}

func envList(ctx *repl.ShellContext) error {
	currentEnv := ""
	current := ctx.Tree.CurrentEnv()
	if current != nil {
		currentEnv = current.Name
	}

	for name, env := range ctx.Tree.Environments {
		envInfo := name
		if env.BaseURL != "" {
			envInfo = fmt.Sprintf("%s (%s)", name, env.BaseURL)
		}

		marker := " "
		if name == currentEnv {
			marker = "←"
		}

		fmt.Printf("%s%s  [%s] %d vars\n",
			marker, envInfo, env.Name, len(env.Vars))
	}
	return nil
}

func envShow(ctx *repl.ShellContext, args []string) error {
	name := ""
	if len(args) >= 1 {
		name = args[0]
	} else {
		env := ctx.Tree.CurrentEnv()
		if env == nil {
			return fmt.Errorf("no current environment")
		}
		name = env.Name
	}

	env, ok := ctx.Tree.Environments[name]
	if !ok {
		return fmt.Errorf("environment %q not found", name)
	}

	fmt.Printf("Name: %s\n", env.Name)
	fmt.Printf("BaseURL: %s\n", env.BaseURL)
	fmt.Printf("Variables: %d\n", len(env.Vars))
	fmt.Printf("Headers: %d\n", len(env.Headers))
	if env.Auth != nil {
		fmt.Printf("Auth: %s\n", env.Auth.Type)
	}

	return nil
}

func envCopy(ctx *repl.ShellContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: /env copy <src> <dst>")
	}

	srcName := args[0]
	dstName := args[1]

	src, ok := ctx.Tree.Environments[srcName]
	if !ok {
		return fmt.Errorf("source environment %q not found", srcName)
	}

	if _, ok := ctx.Tree.Environments[dstName]; ok {
		return fmt.Errorf("destination environment %q already exists", dstName)
	}

	ctx.Tree.Environments[dstName] = src.Clone()
	repl.PrintSuccess(fmt.Sprintf("Copied environment %q to %q", srcName, dstName))
	return nil
}

func envDrop(ctx *repl.ShellContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: /env drop <name>")
	}

	name := args[0]

	// Check if any session is using this env
	for _, s := range ctx.Tree.Sessions {
		if s.EnvName == name {
			return fmt.Errorf("cannot drop environment %q: it is used by session %q", name, s.Name)
		}
	}

	if _, ok := ctx.Tree.Environments[name]; !ok {
		return fmt.Errorf("environment %q not found", name)
	}

	// Don't drop if it's the current environment
	env := ctx.Tree.CurrentEnv()
	if env != nil && env.Name == name {
		return fmt.Errorf("cannot drop current environment; switch to another first")
	}

	delete(ctx.Tree.Environments, name)
	repl.PrintSuccess(fmt.Sprintf("Dropped environment %q", name))
	return nil
}

func envRename(ctx *repl.ShellContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: /env rename <old-name> <new-name>")
	}

	oldName := args[0]
	newName := args[1]

	if _, ok := ctx.Tree.Environments[newName]; ok {
		return fmt.Errorf("environment %q already exists", newName)
	}

	env, ok := ctx.Tree.Environments[oldName]
	if !ok {
		return fmt.Errorf("environment %q not found", oldName)
	}

	// Update env name
	env.Name = newName
	ctx.Tree.Environments[newName] = env
	delete(ctx.Tree.Environments, oldName)

	// Update session references
	for _, s := range ctx.Tree.Sessions {
		if s.EnvName == oldName {
			s.EnvName = newName
		}
	}

	repl.PrintSuccess(fmt.Sprintf("Renamed environment %q to %q", oldName, newName))
	return nil
}

func init() {
	repl.Register(&envCmd{})
}
