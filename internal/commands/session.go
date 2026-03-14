package commands

import (
	"fmt"
	"strings"
	"time"

	"httpclient/internal/model"
	"httpclient/internal/repl"
)

type sessionCmd struct{}

func (c *sessionCmd) Name() string      { return "session" }
func (c *sessionCmd) Aliases() []string { return nil }
func (c *sessionCmd) Help() string {
	return "Manage sessions: new, branch, switch, list, rename, drop, move"
}

func (c *sessionCmd) Run(ctx *repl.ShellContext, args []string) error {
	if len(args) == 0 {
		return c.Run(ctx, []string{"list"})
	}

	subcmd := args[0]
	switch subcmd {
	case "new":
		return sessionNew(ctx, args[1:])
	case "branch":
		return sessionBranch(ctx, args[1:])
	case "switch":
		return sessionSwitch(ctx, args[1:])
	case "list":
		return sessionList(ctx)
	case "rename":
		return sessionRename(ctx, args[1:])
	case "drop":
		return sessionDrop(ctx, args[1:])
	case "move":
		return sessionMove(ctx, args[1:])
	default:
		return fmt.Errorf("unknown session subcommand: %s", subcmd)
	}
}

func (c *sessionCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	fields := strings.Fields(partial)

	if len(fields) == 1 {
		return []string{"new", "branch", "switch", "list", "rename", "drop", "move"}
	}

	subcmd := fields[0]
	lastArg := ""
	if len(fields) > 1 {
		lastArg = fields[len(fields)-1]
	}

	switch subcmd {
	case "switch", "rename", "drop", "move":
		// Complete session names
		var names []string
		for _, s := range ctx.Tree.Sessions {
			if strings.HasPrefix(s.Name, lastArg) {
				names = append(names, s.Name)
			}
		}
		return names
	case "new", "branch":
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

func sessionNew(ctx *repl.ShellContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: /session new <name> <env>")
	}

	name := args[0]
	envName := args[1]

	if _, ok := ctx.Tree.Environments[envName]; !ok {
		return fmt.Errorf("environment %q does not exist", envName)
	}

	for _, s := range ctx.Tree.Sessions {
		if s.Name == name {
			return fmt.Errorf("session %q already exists", name)
		}
	}

	id := fmt.Sprintf("sess_%d", time.Now().Unix())
	sess := &model.Session{
		ID:              id,
		Name:            name,
		EnvName:         envName,
		ParentID:        "",
		Requests:        []*model.Request{},
		HeaderOverrides: make(map[string]string),
		VarOverrides:    make(map[string]any),
		CreatedAt:       time.Now(),
	}

	ctx.Tree.Sessions[id] = sess
	ctx.Tree.CurrentID = id

	repl.PrintSuccess(fmt.Sprintf("Created session %q bound to environment %q", name, envName))
	return nil
}

func sessionBranch(ctx *repl.ShellContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: /session branch <name> [env]")
	}

	name := args[0]

	current := ctx.Tree.Current()
	if current == nil {
		return fmt.Errorf("no current session")
	}

	envName := current.EnvName
	if len(args) >= 2 {
		envName = args[1]
	}

	if _, ok := ctx.Tree.Environments[envName]; !ok {
		return fmt.Errorf("environment %q does not exist", envName)
	}

	for _, s := range ctx.Tree.Sessions {
		if s.Name == name {
			return fmt.Errorf("session %q already exists", name)
		}
	}

	id := fmt.Sprintf("sess_%d", time.Now().Unix())
	sess := &model.Session{
		ID:              id,
		Name:            name,
		EnvName:         envName,
		ParentID:        current.ID,
		Requests:        []*model.Request{},
		HeaderOverrides: make(map[string]string),
		VarOverrides:    make(map[string]any),
		CreatedAt:       time.Now(),
	}

	ctx.Tree.Sessions[id] = sess
	ctx.Tree.CurrentID = id

	repl.PrintSuccess(fmt.Sprintf("Branched to new session %q (child of %q)", name, current.Name))
	return nil
}

func sessionSwitch(ctx *repl.ShellContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: /session switch <name>")
	}

	name := args[0]

	// Handle "-" to switch to previous session
	if name == "-" {
		if ctx.Tree.PreviousID == "" || ctx.Tree.Sessions[ctx.Tree.PreviousID] == nil {
			return fmt.Errorf("no previous session")
		}
		prevID := ctx.Tree.PreviousID
		ctx.Tree.PreviousID = ctx.Tree.CurrentID
		ctx.Tree.CurrentID = prevID
		repl.PrintSuccess(fmt.Sprintf("Switched to previous session %q", ctx.Tree.Current().Name))
		return nil
	}

	for id, s := range ctx.Tree.Sessions {
		if s.Name == name {
			ctx.Tree.PreviousID = ctx.Tree.CurrentID
			ctx.Tree.CurrentID = id
			repl.PrintSuccess(fmt.Sprintf("Switched to session %q", name))
			return nil
		}
	}

	return fmt.Errorf("session %q not found", name)
}

func sessionList(ctx *repl.ShellContext) error {
	printSession := func(sess *model.Session, indent int) {
		env := ctx.Tree.Environments[sess.EnvName]
		envName := sess.EnvName
		if env != nil && env.BaseURL != "" {
			envName = fmt.Sprintf("%s (%s)", sess.EnvName, env.BaseURL)
		}

		marker := " "
		if sess.ID == ctx.Tree.CurrentID {
			marker = "←"
		}

		prefix := strings.Repeat("  ", indent)
		if indent > 0 {
			prefix = prefix + "└─ "
		}

		fmt.Printf("%s%s%-15s [%s] %d requests %s %s\n",
			prefix, marker, sess.Name, envName, len(sess.Requests), marker, sess.CreatedAt.Format("2006-01-02 15:04"))
	}

	// Find root sessions (no parent)
	var roots []*model.Session
	for _, s := range ctx.Tree.Sessions {
		if s.ParentID == "" {
			roots = append(roots, s)
		}
	}

	// Print recursively
	var printTree func(sess *model.Session, indent int)
	printTree = func(sess *model.Session, indent int) {
		printSession(sess, indent)
		children := ctx.Tree.Children(sess.ID)
		for _, child := range children {
			printTree(child, indent+1)
		}
	}

	for _, root := range roots {
		printTree(root, 0)
	}

	return nil
}

func sessionRename(ctx *repl.ShellContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: /session rename <old> <new>")
	}

	oldName := args[0]
	newName := args[1]

	var target *model.Session
	for _, s := range ctx.Tree.Sessions {
		if s.Name == oldName {
			target = s
			break
		}
	}

	if target == nil {
		return fmt.Errorf("session %q not found", oldName)
	}

	// Check if new name exists
	for _, s := range ctx.Tree.Sessions {
		if s.Name == newName {
			return fmt.Errorf("session %q already exists", newName)
		}
	}

	// Update parent references
	for _, s := range ctx.Tree.Sessions {
		if s.ParentID == target.ID {
			s.ParentID = newName
		}
	}

	target.Name = newName
	repl.PrintSuccess(fmt.Sprintf("Renamed session %q to %q", oldName, newName))
	return nil
}

func sessionDrop(ctx *repl.ShellContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: /session drop <name>")
	}

	name := args[0]

	var target *model.Session
	var targetID string
	for id, s := range ctx.Tree.Sessions {
		if s.Name == name {
			target = s
			targetID = id
			break
		}
	}

	if target == nil {
		return fmt.Errorf("session %q not found", name)
	}

	// Check for children
	children := ctx.Tree.Children(targetID)
	if len(children) > 0 {
		return fmt.Errorf("refuse to drop session %q: it has %d child session(s)", name, len(children))
	}

	// Can't drop current session without switching
	if targetID == ctx.Tree.CurrentID {
		return fmt.Errorf("cannot drop current session; switch to another first")
	}

	delete(ctx.Tree.Sessions, targetID)
	repl.PrintSuccess(fmt.Sprintf("Dropped session %q", name))
	return nil
}

func sessionMove(ctx *repl.ShellContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: /session move <req-id> <target-session>")
	}

	reqID := args[0]
	targetSessionName := args[1]

	current := ctx.Tree.Current()
	req, ok := current.GetRequest(reqID)
	if !ok {
		return fmt.Errorf("request %q not found in current session", reqID)
	}

	var target *model.Session
	for _, s := range ctx.Tree.Sessions {
		if s.Name == targetSessionName {
			target = s
			break
		}
	}

	if target == nil {
		return fmt.Errorf("target session %q not found", targetSessionName)
	}

	// Remove from current session
	current.RemoveRequest(reqID)

	// Clone request and add to target
	cloned := req.Clone()
	target.AddRequest(cloned)

	repl.PrintSuccess(fmt.Sprintf("Moved request %s to session %q (now %s)", reqID, targetSessionName, cloned.ID))
	return nil
}

func init() {
	repl.Register(&sessionCmd{})
}
