package repl

import (
	"strings"

	"github.com/chzyer/readline"
)

type completer struct {
	ctx *ShellContext
}

func (c *completer) Do(line []rune, pos int) (newLine [][]rune, length int) {
	lineStr := string(line)
	parts := tokenizeComplete(lineStr)

	if len(parts) == 0 {
		return nil, 0
	}

	var completions []string
	cmdPart := parts[0]

	// Check if it's a command (starts with /)
	if strings.HasPrefix(cmdPart, "/") {
		cmd := strings.TrimPrefix(cmdPart, "/")

		if cmd == "" {
			// No command yet, complete command names
			for name := range commands {
				completions = append(completions, "/"+name)
			}
		} else {
			// Complete command name
			for name := range commands {
				if strings.HasPrefix(name, cmd) {
					completions = append(completions, "/"+name)
				}
			}

			// If we have a matching command, try to complete its arguments
			if cmdObj, ok := commands[cmd]; ok && len(parts) > 1 {
				args := parts[1:]
				lastArg := ""
				if len(args) > 0 {
					lastArg = args[len(args)-1]
				}
				completions = cmdObj.Complete(c.ctx, lastArg)
			}
		}
	}

	if len(completions) == 0 {
		return nil, 0
	}

	var result [][]rune
	for _, comp := range completions {
		result = append(result, []rune(comp))
	}

	return result, pos
}

func tokenizeComplete(line string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := ' '

	for _, r := range line {
		if !inQuote && (r == '"' || r == '\'') {
			inQuote = true
			quoteChar = r
		} else if inQuote && r == quoteChar {
			inQuote = false
		} else if !inQuote && r == ' ' {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

func NewCompleter(ctx *ShellContext) readline.AutoCompleter {
	return &completer{ctx: ctx}
}
