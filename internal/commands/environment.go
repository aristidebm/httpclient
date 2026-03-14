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
func (c *envCmd) Help() string      { return "Manage environments: new, set, unset, list, show, copy" }

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
	default:
		return fmt.Errorf("unknown env subcommand: %s", subcmd)
	}
}

func (c *envCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	fields := strings.Fields(partial)

	if len(fields) == 1 {
		return []string{"new", "set", "unset", "list", "show", "copy"}
	}

	subcmd := fields[0]
	lastArg := ""
	if len(fields) > 1 {
		lastArg = fields[len(fields)-1]
	}

	switch subcmd {
	case "set", "unset", "show", "copy":
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
	fmt.Printf("%-15s %-40s %-10s %-10s\n", "NAME", "BASE URL", "HEADERS", "VARS")
	fmt.Println(strings.Repeat("-", 80))

	for _, env := range ctx.Tree.Environments {
		baseURL := env.BaseURL
		if len(baseURL) > 40 {
			baseURL = baseURL[:37] + "..."
		}
		fmt.Printf("%-15s %-40s %-10d %-10d\n",
			env.Name,
			baseURL,
			len(env.Headers),
			len(env.Vars))
	}

	return nil
}

func envShow(ctx *repl.ShellContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: /env show <name>")
	}

	name := args[0]

	env, ok := ctx.Tree.Environments[name]
	if !ok {
		return fmt.Errorf("environment %q not found", name)
	}

	fmt.Printf("Name: %s\n", env.Name)
	fmt.Printf("Base URL: %s\n", env.BaseURL)

	if len(env.Headers) > 0 {
		fmt.Println("\nHeaders:")
		for k, v := range env.Headers {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	if len(env.Vars) > 0 {
		fmt.Println("\nVariables:")
		for k, v := range env.Vars {
			fmt.Printf("  %s: %v\n", k, v)
		}
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

func init() {
	repl.Register(&envCmd{})
}
