package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	_ "httpclient/internal/commands"
	"httpclient/internal/input"
	"httpclient/internal/model"
	"httpclient/internal/repl"

	"github.com/spf13/cobra"
)

var sessionFlag string

var rootCmd = &cobra.Command{
	Use:                   "httpclient [spec]",
	Short:                 "HTTP client with interactive REPL",
	Long:                  `An HTTP client with interactive REPL shell and one-shot mode.`,
	Run:                   runRoot,
	DisableFlagsInUseLine: true,
}

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List available sessions",
	Run:   runSessions,
}

func Execute() error {
	rootCmd.AddCommand(sessionsCmd)
	rootCmd.Flags().StringVar(&sessionFlag, "session", "", "Session name")
	return rootCmd.Execute()
}

func runRoot(cmd *cobra.Command, args []string) {
	ctx := repl.NewShellContext()

	if sessionFlag != "" {
		if err := switchToSession(ctx, sessionFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	if len(args) == 0 {
		if err := ctx.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	spec := args[0]
	if spec == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := importSpec(ctx, data); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := importPath(ctx, spec); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	if err := ctx.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runSessions(cmd *cobra.Command, args []string) {
	ctx := repl.NewShellContext()

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

	var roots []*model.Session
	for _, s := range ctx.Tree.Sessions {
		if s.ParentID == "" {
			roots = append(roots, s)
		}
	}

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
}

func switchToSession(ctx *repl.ShellContext, name string) error {
	for id, sess := range ctx.Tree.Sessions {
		if sess.Name == name {
			ctx.Tree.PreviousID = ctx.Tree.CurrentID
			ctx.Tree.CurrentID = id
			return nil
		}
	}

	envName := "default"
	if len(ctx.Tree.Environments) == 0 {
		ctx.Tree.Environments[envName] = &model.Environment{
			Name:    envName,
			BaseURL: "",
			Vars:    make(model.Variables),
			Headers: make(map[string]string),
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
		Vars:            make(model.Variables),
		CreatedAt:       time.Now(),
	}

	ctx.Tree.Sessions[id] = sess
	ctx.Tree.CurrentID = id

	fmt.Printf("Created new session %q\n", name)
	return nil
}

func importPath(ctx *repl.ShellContext, target string) error {
	var data []byte
	var err error

	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		resp, err := http.Get(target)
		if err != nil {
			return fmt.Errorf("failed to fetch URL: %w", err)
		}
		defer resp.Body.Close()
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
	} else {
		data, err = os.ReadFile(target)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
	}

	return importSpec(ctx, data)
}

func importSpec(ctx *repl.ShellContext, data []byte) error {
	format, err := input.Detect(data)
	if err != nil {
		return fmt.Errorf("failed to detect format: %w", err)
	}

	switch format {
	case input.FormatOpenAPI:
		spec, err := input.LoadOpenAPI(data)
		if err != nil {
			return fmt.Errorf("failed to load OpenAPI: %w", err)
		}

		ctx.SetSpec(spec)

		fmt.Printf("Loaded OpenAPI spec: %s (version: %s)\n", spec.Title, spec.Version)
		fmt.Printf("Found %d routes\n", len(spec.Routes))

	case input.FormatHTTPFile:
		requests, err := input.ParseHTTPFile(data)
		if err != nil {
			return fmt.Errorf("failed to parse HTTP file: %w", err)
		}

		session := ctx.Tree.Current()
		for _, req := range requests {
			session.AddRequest(req)
		}

		fmt.Printf("Loaded %d requests from HTTP file\n", len(requests))

	case input.FormatCurl:
		requests, err := input.ParseCurl(data)
		if err != nil {
			return fmt.Errorf("failed to parse curl: %w", err)
		}

		session := ctx.Tree.Current()
		for _, req := range requests {
			session.AddRequest(req)
		}

		fmt.Printf("Loaded %d requests from curl commands\n", len(requests))

	default:
		return fmt.Errorf("unknown format")
	}

	return nil
}
