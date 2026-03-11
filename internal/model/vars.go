package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var varPattern = regexp.MustCompile(`\{([^}]+)\}`)

func ResolveVars(template string, layers ...map[string]any) (string, []string) {
	var unresolved []string

	result := varPattern.ReplaceAllStringFunc(template, func(match string) string {
		key := match[1 : len(match)-1]

		for i := 0; i < len(layers); i++ {
			layer := layers[i]
			if layer == nil {
				continue
			}
			if val, ok := layer[key]; ok {
				return fmt.Sprintf("%v", val)
			}
		}

		unresolved = append(unresolved, key)
		return match
	})

	return result, unresolved
}

func ResolveVarsStrict(template string, layers ...map[string]any) (string, error) {
	result, unresolved := ResolveVars(template, layers...)
	if len(unresolved) > 0 {
		return result, errors.New("unresolved variables: " + strings.Join(unresolved, ", "))
	}
	return result, nil
}
