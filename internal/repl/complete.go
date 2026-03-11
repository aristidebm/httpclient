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
	parts := strings.Fields(lineStr)
	if len(parts) == 0 || !strings.HasPrefix(parts[0], "/") {
		return nil, 0
	}

	cmd := strings.TrimPrefix(parts[0], "/")
	var completions []string

	if cmd == "" {
		for name := range commands {
			completions = append(completions, "/"+name)
		}
	} else {
		for name := range commands {
			if strings.HasPrefix(name, cmd) {
				completions = append(completions, "/"+name)
			}
		}
	}

	if len(completions) == 0 {
		return nil, 0
	}

	var result [][]rune
	for _, c := range completions {
		result = append(result, []rune(c))
	}

	return result, pos
}

func NewCompleter(ctx *ShellContext) readline.AutoCompleter {
	return &completer{ctx: ctx}
}
