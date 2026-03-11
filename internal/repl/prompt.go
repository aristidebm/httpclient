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

	user := ""
	if u, ok := env.Vars["user"]; ok {
		user = toString(u)
	}
	if user == "" {
		user = "user"
	}

	// httpclient := color.New(color.FgHiBlack).Sprint("httpclient")
	envPart := color.New(color.FgCyan).Sprintf("%s@%s", user, env.Name)
	sessionPart := color.New(color.FgGreen).Sprint(session.Name)
	prompt := color.New(color.FgHiBlack).Sprint("› ")

	// return "[httpclient : " + httpclient + " : " + envPart + " : " + sessionPart + "] " + prompt
	return "[httpclient : " + envPart + " : " + sessionPart + "] " + prompt
}

func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	default:
		return ""
	}
}
