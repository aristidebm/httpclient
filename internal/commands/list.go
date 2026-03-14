package commands

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"httpclient/internal/model"
	"httpclient/internal/repl"
)

type listCmd struct{}

func (c *listCmd) Name() string      { return "list" }
func (c *listCmd) Aliases() []string { return nil }
func (c *listCmd) Help() string      { return "List routes from loaded OpenAPI spec: /list [filter]" }

func (c *listCmd) Run(ctx *repl.ShellContext, args []string) error {
	spec := ctx.CurrentSpec()
	if spec == nil {
		fmt.Println("No spec loaded. Use /load <url-or-file>")
		return nil
	}

	filter := ""
	if len(args) > 0 {
		filter = strings.ToLower(args[0])
	}

	routes := spec.Routes
	if filter != "" {
		var filtered []model.Route
		for _, r := range routes {
			pathMatch := strings.Contains(strings.ToLower(r.Path), filter)
			sumMatch := strings.Contains(strings.ToLower(r.Summary), filter)
			tagMatch := false
			for _, t := range r.Tags {
				if strings.Contains(strings.ToLower(t), filter) {
					tagMatch = true
					break
				}
			}
			if pathMatch || sumMatch || tagMatch {
				filtered = append(filtered, r)
			}
		}
		routes = filtered
	}

	// Group by tag
	tagGroups := make(map[string][]model.Route)
	var tags []string
	for _, r := range routes {
		if len(r.Tags) == 0 {
			r.Tags = []string{"(no tag)"}
		}
		for _, tag := range r.Tags {
			if _, ok := tagGroups[tag]; !ok {
				tags = append(tags, tag)
			}
			tagGroups[tag] = append(tagGroups[tag], r)
		}
	}
	sort.Strings(tags)

	// Build output
	var lines []string
	for _, tag := range tags {
		lines = append(lines, fmt.Sprintf("\n## %s\n", tag))
		for _, r := range tagGroups[tag] {
			method := r.Method
			path := r.Path
			summary := r.Summary
			if summary != "" {
				lines = append(lines, fmt.Sprintf("  %-7s %-40s %s", method, path, summary))
			} else {
				lines = append(lines, fmt.Sprintf("  %-7s %s", method, path))
			}
		}
	}

	output := strings.Join(lines, "\n")

	// Use pager if output is long
	if len(output) > 4096 {
		pager := os.Getenv("PAGER")
		if pager == "" {
			pager = "less"
		}
		cmd := exec.Command(pager)
		cmd.Stdin = strings.NewReader(output)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	fmt.Println(output)
	return nil
}

func (c *listCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	return nil
}

func init() {
	repl.Register(&listCmd{})
}
