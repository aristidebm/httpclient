package repl

import (
	"github.com/fatih/color"
)

func BuildPrompt(ctx *ShellContext) string {
	session := ctx.Tree.Current()
	if session == nil {
		return color.BlueString("[cdapi] › ")
	}

	env := ctx.Tree.CurrentEnv()
	if env == nil {
		return color.GreenString("[cdapi] › ")
	}

	user := ""
	if u, ok := env.Vars["user"]; ok {
		user = toString(u)
	}
	if user == "" {
		user = "user"
	}

	cdapi := color.New(color.FgHiBlack).Sprint("cdapi")
	envPart := color.New(color.FgCyan).Sprintf("%s@%s", user, env.Name)
	sessionPart := color.New(color.FgGreen).Sprint(session.Name)
	prompt := color.New(color.FgHiBlack).Sprint("› ")

	return "[cdapi : " + cdapi + " : " + envPart + " : " + sessionPart + "] " + prompt
}

func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	default:
		return ""
	}
}
