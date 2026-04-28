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
	prompt := color.New(color.FgHiBlack).Sprint("› ")

	return "[" + sessionPart + "] " + prompt
}
