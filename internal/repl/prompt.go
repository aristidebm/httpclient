package repl

import (
	"github.com/fatih/color"
)

func BuildPrompt(ctx *ShellContext) string {
	session := ctx.Tree.Current()
	if session == nil {
		return color.BlueString("[httpclient] › ")
	}

	sessionPart := color.New(color.FgGreen).Sprint(session.Name)

	// Show parent if this is a child session
	if session.ParentID != "" {
		if parent := ctx.Tree.Sessions[session.ParentID]; parent != nil {
			sessionPart = sessionPart + "@" + color.New(color.FgCyan).Sprint(parent.Name)
		}
	}

	// Show auth indicator if session or inherited auth is set
	if session.Auth != nil || ctx.Tree.GetInheritedAuth(session.ID) != nil {
		sessionPart += color.New(color.FgYellow).Sprint(" [A]")
	}

	prompt := color.New(color.FgHiBlack).Sprint("› ")

	return "[" + sessionPart + "] " + prompt
}
