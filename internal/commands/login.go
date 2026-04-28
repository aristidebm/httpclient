package commands

import (
	"fmt"
	"strings"

	"httpclient/internal/model"
	"httpclient/internal/repl"
)

type loginCmd struct{}

func (c *loginCmd) Name() string      { return "login" }
func (c *loginCmd) Aliases() []string { return nil }
func (c *loginCmd) Help() string {
	return "Manage authentication: basic, token, oauth, clear, show"
}

func (c *loginCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	fields := strings.Fields(partial)
	if len(fields) == 1 {
		return []string{"basic", "token", "oauth", "clear", "show"}
	}
	return nil
}

func (c *loginCmd) Run(ctx *repl.ShellContext, args []string) error {
	if len(args) == 0 {
		return c.Run(ctx, []string{"show"})
	}

	subcmd := args[0]
	switch subcmd {
	case "basic":
		return loginBasic(ctx, args[1:])
	case "token":
		return loginToken(ctx, args[1:])
	case "oauth":
		return loginOAuth(ctx, args[1:])
	case "clear":
		return loginClear(ctx)
	case "show":
		return loginShow(ctx)
	default:
		return fmt.Errorf("unknown login subcommand: %s (use: basic, token, oauth, clear, show)", subcmd)
	}
}

func loginBasic(ctx *repl.ShellContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: /login basic <username> [password] OR /login basic <username:password>")
	}

	username := args[0]
	password := ""
	if len(args) >= 2 {
		password = args[1]
	}

	// Support username:password format
	if strings.Contains(username, ":") && password == "" {
		parts := strings.SplitN(username, ":", 2)
		username = parts[0]
		password = parts[1]
	}

	sess := ctx.Tree.Current()
	if sess == nil {
		return fmt.Errorf("no current session")
	}

	if sess.Auth == nil {
		sess.Auth = &model.AuthConfig{}
	}

	sess.Auth.Type = "basic"
	sess.Auth.Username = username
	sess.Auth.Password = password

	repl.PrintSuccess(fmt.Sprintf("Basic auth configured for user %q in session %q", username, sess.Name))
	return nil
}

func loginToken(ctx *repl.ShellContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: /login token <token> [--header=X] [--type=Y]")
	}

	token := args[0]
	headerName := "Authorization"
	tokenType := "Bearer"

	// Parse flags
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "--header=") {
			headerName = strings.TrimPrefix(arg, "--header=")
		}
		if strings.HasPrefix(arg, "--type=") {
			tokenType = strings.TrimPrefix(arg, "--type=")
		}
	}

	sess := ctx.Tree.Current()
	if sess == nil {
		return fmt.Errorf("no current session")
	}

	if sess.Auth == nil {
		sess.Auth = &model.AuthConfig{}
	}

	sess.Auth.Type = "token"
	sess.Auth.Token = token
	sess.Auth.TokenType = tokenType
	sess.Auth.HeaderName = headerName

	repl.PrintSuccess(fmt.Sprintf("Token auth configured in session %q", sess.Name))
	return nil
}

func loginOAuth(ctx *repl.ShellContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: /login oauth config <client-id> <token-url> [client-secret] [--auth-url=<url>]")
	}

	subcmd := args[0]
	switch subcmd {
	case "config":
		if len(args) < 3 {
			return fmt.Errorf("usage: /login oauth config <client-id> <token-url> [client-secret] [--auth-url=<url>]")
		}
		clientID := args[1]
		tokenURL := args[2]
		clientSecret := ""

		// Parse optional args
		var authURL string
		for _, arg := range args[3:] {
			if strings.HasPrefix(arg, "--auth-url=") {
				authURL = strings.TrimPrefix(arg, "--auth-url=")
			} else if clientSecret == "" {
				clientSecret = arg
			}
		}

		sess := ctx.Tree.Current()
		if sess == nil {
			return fmt.Errorf("no current session")
		}

		if sess.Auth == nil {
			sess.Auth = &model.AuthConfig{}
		}

		sess.Auth.Type = "oauth"
		sess.Auth.ClientID = clientID
		sess.Auth.TokenURL = tokenURL
		sess.Auth.ClientSecret = clientSecret

		// Set authURL in vars if provided
		if authURL != "" {
			sess.Vars.Set("authURL", authURL, model.VarScopeSession)
		}

		repl.PrintSuccess(fmt.Sprintf("OAuth config set in session %q", sess.Name))
		return nil
	default:
		return fmt.Errorf("unknown oauth subcommand: %s (use: config)", subcmd)
	}
}

func loginClear(ctx *repl.ShellContext) error {
	sess := ctx.Tree.Current()
	if sess == nil {
		return fmt.Errorf("no current session")
	}

	if sess.Auth == nil {
		repl.PrintInfo("No auth configured")
		return nil
	}

	sess.Auth = nil
	repl.PrintSuccess(fmt.Sprintf("Auth cleared from session %q", sess.Name))
	return nil
}

func loginShow(ctx *repl.ShellContext) error {
	sess := ctx.Tree.Current()
	if sess == nil {
		return fmt.Errorf("no current session")
	}

	if sess.Auth == nil {
		fmt.Println("No auth configured for this session.")
		return nil
	}

	fmt.Printf("Type: %s\n", sess.Auth.Type)

	// Show effective auth URL
	effectiveAuthURL := ctx.Tree.GetEffectiveAuthURL(sess.ID)
	if effectiveAuthURL != "" {
		fmt.Printf("AuthURL: %s\n", effectiveAuthURL)
	}

	switch sess.Auth.Type {
	case "basic":
		fmt.Printf("Username: %s\n", sess.Auth.Username)
		if sess.Auth.Password != "" {
			fmt.Println("Password: ***")
		}
	case "token":
		fmt.Printf("Token: ***\n")
		fmt.Printf("TokenType: %s\n", sess.Auth.TokenType)
		fmt.Printf("Header: %s\n", sess.Auth.HeaderName)
	case "oauth":
		fmt.Printf("ClientID: %s\n", sess.Auth.ClientID)
		fmt.Printf("TokenURL: %s\n", sess.Auth.TokenURL)
		if sess.Auth.AccessToken != "" {
			fmt.Println("AccessToken: ***")
		}
	}

	return nil
}

func init() {
	repl.Register(&loginCmd{})
}
