package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	_ "httpclient/internal/commands"
	"httpclient/internal/executor"
	"httpclient/internal/model"
	"httpclient/internal/repl"

	"github.com/spf13/cobra"
)

var (
	methodFlag  string
	userFlag    string
	envFlag     string
	tokenFlag   string
	timeoutFlag int
	paramsFlag  []string
	dataFlag    []string
	headersFlag []string
	acceptFlag  string
	formFlag    bool
	saveFlag    bool
	timeFlag    bool
)

var rootCmd = &cobra.Command{
	Use:   "httpclient",
	Short: "HTTP client with interactive REPL",
	Long:  `An HTTP client with interactive REPL shell and one-shot mode.`,
	Run:   runRoot,
}

func Execute() error {
	return rootCmd.Execute()
}

func runRoot(cmd *cobra.Command, args []string) {
	// No args → interactive shell
	if len(args) == 0 {
		runShell(nil)
		return
	}

	// Single arg that doesn't start with - and is not a URL → login flow
	if len(args) == 1 && !strings.HasPrefix(args[0], "-") && !strings.HasPrefix(args[0], "http") {
		runShell(&args[0])
		return
	}

	// Otherwise: one-shot mode
	if err := runOneShot(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runShell(loginURL *string) {
	ctx := repl.NewShellContext()

	if loginURL != nil {
		// Login flow: create default session with the provided URL
		env := &model.Environment{
			Name:    "default",
			BaseURL: *loginURL,
			Vars:    make(map[string]any),
			Headers: make(map[string]string),
		}
		ctx.Tree.Environments["default"] = env

		sess := ctx.Tree.Current()
		sess.EnvName = "default"
	}

	if err := ctx.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runOneShot(args []string) error {
	// Determine URL and method
	// Handle: httpclient URL or httpclient METHOD URL or httpclient -X METHOD URL
	var url string
	var method string

	// Check if first arg is a method (GET, POST, etc.)
	possibleMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "TRACE", "CONNECT"}
	if len(args) > 0 {
		for _, m := range possibleMethods {
			if strings.EqualFold(args[0], m) {
				method = m
				if len(args) > 1 {
					url = args[1]
				}
				break
			}
		}
	}

	// If no method found, look for URL in args
	if url == "" {
		for _, arg := range args {
			if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
				url = arg
				break
			}
		}
	}

	// Apply method flag
	if methodFlag != "" {
		method = strings.ToUpper(methodFlag)
	}

	if method == "" {
		method = "GET"
	}

	if url == "" {
		return fmt.Errorf("URL is required")
	}

	// Build environment
	env := &model.Environment{
		Name:    "oneshot",
		BaseURL: "",
		Vars:    make(map[string]any),
		Headers: make(map[string]string),
	}

	// Apply flags
	if envFlag != "" {
		if strings.HasPrefix(envFlag, "http://") || strings.HasPrefix(envFlag, "https://") {
			env.BaseURL = envFlag
		} else {
			env.Name = envFlag
		}
	}

	if tokenFlag != "" {
		env.Headers["Authorization"] = fmt.Sprintf("TOKEN %s", tokenFlag)
	}

	if userFlag != "" {
		env.Vars["user"] = userFlag
	}

	// Parse params (-p)
	for _, p := range paramsFlag {
		// key=value or key:value
		parts := strings.SplitN(p, "=", 2)
		if len(parts) != 2 {
			parts = strings.SplitN(p, ":", 2)
		}
		if len(parts) == 2 {
			if env.Vars == nil {
				env.Vars = make(map[string]any)
			}
			env.Vars[parts[0]] = parts[1]
		}
	}

	// Parse headers (-H)
	for _, h := range headersFlag {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			env.Headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	// Parse body (-d)
	var body string
	if len(dataFlag) > 0 {
		formFlag = true // auto-enable form mode when -d is used
	}
	for _, d := range dataFlag {
		body = body + d + "&"
	}
	body = strings.TrimSuffix(body, "&")

	// Apply content type - default to form if -d is used
	contentType := "application/json"
	if formFlag {
		contentType = "application/x-www-form-urlencoded"
	}
	env.Headers["Content-Type"] = contentType

	// Apply accept header shortcut
	if acceptFlag != "" {
		acceptShortcuts := map[string]string{
			"json": "application/json",
			"xml":  "application/xml",
			"pdf":  "application/pdf",
			"csv":  "text/csv",
			"html": "text/html",
			"text": "text/plain",
		}
		if v, ok := acceptShortcuts[acceptFlag]; ok {
			env.Headers["Accept"] = v
		} else {
			env.Headers["Accept"] = acceptFlag
		}
	}

	// Resolve URL with base URL
	fullURL := url
	if env.BaseURL != "" && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		fullURL = strings.TrimSuffix(env.BaseURL, "/") + "/" + strings.TrimPrefix(url, "/")
	}

	// Create request
	req := &model.Request{
		Method:      method,
		URL:         fullURL,
		Headers:     env.Headers,
		Body:        []byte(body),
		ContentType: contentType,
	}

	// Set timeout
	timeout := 30 * time.Second
	if timeoutFlag > 0 {
		timeout = time.Duration(timeoutFlag) * time.Second
	}
	client := executor.NewClient(timeout)

	// Execute
	if err := client.Execute(req, env); err != nil {
		return err
	}

	// Print response
	repl.PrintResponse(req.Response)

	// Auto-save binary if requested
	if saveFlag {
		if err := saveBinary(req.Response); err != nil {
			return err
		}
	}

	// Print time if requested
	if timeFlag {
		fmt.Printf("\nRequest completed in %v\n", req.Duration)
	}

	// Exit code
	if req.Response != nil && req.Response.StatusCode >= 200 && req.Response.StatusCode < 300 {
		os.Exit(0)
	}
	os.Exit(1)
	return nil
}

func saveBinary(resp *model.Response) error {
	if resp == nil {
		return nil
	}

	ct := resp.Headers["Content-Type"]
	if !strings.Contains(ct, "image/") && !strings.Contains(ct, "application/pdf") && !strings.Contains(ct, "application/zip") {
		return nil
	}

	// Determine extension
	ext := ""
	switch {
	case strings.Contains(ct, "pdf"):
		ext = ".pdf"
	case strings.Contains(ct, "png"):
		ext = ".png"
	case strings.Contains(ct, "jpeg") || strings.Contains(ct, "jpg"):
		ext = ".jpg"
	case strings.Contains(ct, "zip"):
		ext = ".zip"
	}

	filename := fmt.Sprintf("httpclient_%d%s", os.Getpid(), ext)
	if err := os.WriteFile(filename, resp.RawBody, 0644); err != nil {
		return err
	}
	fmt.Printf("Saved binary response to: %s\n", filename)
	return nil
}

func init() {
	rootCmd.Flags().StringVarP(&methodFlag, "method", "X", "", "HTTP method")
	rootCmd.Flags().StringVarP(&userFlag, "user", "u", "", "User identifier")
	rootCmd.Flags().StringVar(&envFlag, "env", "", "Environment name or base URL")
	rootCmd.Flags().StringVar(&tokenFlag, "token", "", "Auth token")
	rootCmd.Flags().IntVar(&timeoutFlag, "timeout", 0, "Request timeout in seconds")
	rootCmd.Flags().StringArrayVarP(&paramsFlag, "param", "p", []string{}, "Query params (key=value)")
	rootCmd.Flags().StringArrayVarP(&dataFlag, "data", "d", []string{}, "Request body fields")
	rootCmd.Flags().StringArrayVarP(&headersFlag, "header", "H", []string{}, "Request headers")
	rootCmd.Flags().StringVarP(&acceptFlag, "accept", "a", "", "Accept header shortcut")
	rootCmd.Flags().BoolVar(&formFlag, "form", false, "Send as form-encoded")
	rootCmd.Flags().BoolVar(&saveFlag, "save", false, "Auto-save binary response")
	rootCmd.Flags().BoolVar(&timeFlag, "time", false, "Print execution time")
}
