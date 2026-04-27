package repl

import (
	"github.com/fatih/color"
)

func BuildPrompt(ctx *ShellContext) string {
	session := ctx.Tree.Current()
	if session == nil {
		return color.BlueString("[httpclient] › ")
	}

	env := ctx.Tree.CurrentEnv()
	if env == nil {
		return color.GreenString("[httpclient] › ")
	}

	sessionPart := color.New(color.FgGreen).Sprint(session.Name)
	envPart := color.New(color.FgCyan).Sprint(env.Name)
	prompt := color.New(color.FgHiBlack).Sprint("› ")

	return "[" + sessionPart + "@" + envPart + "] " + prompt
}
