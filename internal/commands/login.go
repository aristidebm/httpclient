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
		return fmt.Errorf("usage: /login basic <username> [password]")
	}

	username := args[0]
	password := ""
	if len(args) >= 2 {
		password = args[1]
	}

	env := ctx.Tree.CurrentEnv()
	if env == nil {
		return fmt.Errorf("no current environment")
	}

	if env.Auth == nil {
		env.Auth = &model.AuthConfig{}
	}

	env.Auth.Type = "basic"
	env.Auth.Username = username
	env.Auth.Password = password

	repl.PrintSuccess(fmt.Sprintf("Basic auth configured for user %q in environment %q", username, env.Name))
	return nil
}

func loginToken(ctx *repl.ShellContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: /login token <token> [--header=X]")
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

	env := ctx.Tree.CurrentEnv()
	if env == nil {
		return fmt.Errorf("no current environment")
	}

	if env.Auth == nil {
		env.Auth = &model.AuthConfig{}
	}

	env.Auth.Type = "token"
	env.Auth.Token = token
	env.Auth.TokenType = tokenType
	env.Auth.HeaderName = headerName

	repl.PrintSuccess(fmt.Sprintf("Token auth configured in environment %q", env.Name))
	return nil
}

func loginOAuth(ctx *repl.ShellContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: /login oauth config <client-id> <token-url> [client-secret]")
	}

	subcmd := args[0]
	switch subcmd {
	case "config":
		if len(args) < 3 {
			return fmt.Errorf("usage: /login oauth config <client-id> <token-url> [client-secret]")
		}
		clientID := args[1]
		tokenURL := args[2]
		clientSecret := ""
		if len(args) >= 4 {
			clientSecret = args[3]
		}

		env := ctx.Tree.CurrentEnv()
		if env == nil {
			return fmt.Errorf("no current environment")
		}

		if env.Auth == nil {
			env.Auth = &model.AuthConfig{}
		}

		env.Auth.Type = "oauth"
		env.Auth.ClientID = clientID
		env.Auth.TokenURL = tokenURL
		env.Auth.ClientSecret = clientSecret

		repl.PrintSuccess(fmt.Sprintf("OAuth config set in environment %q", env.Name))
		return nil
	default:
		return fmt.Errorf("unknown oauth subcommand: %s (use: config)", subcmd)
	}
}

func loginClear(ctx *repl.ShellContext) error {
	env := ctx.Tree.CurrentEnv()
	if env == nil {
		return fmt.Errorf("no current environment")
	}

	if env.Auth == nil {
		repl.PrintInfo("No auth configured")
		return nil
	}

	env.Auth = nil
	repl.PrintSuccess(fmt.Sprintf("Auth cleared from environment %q", env.Name))
	return nil
}

func loginShow(ctx *repl.ShellContext) error {
	env := ctx.Tree.CurrentEnv()
	if env == nil {
		return fmt.Errorf("no current environment")
	}

	if env.Auth == nil {
		fmt.Println("No auth configured for this environment.")
		return nil
	}

	fmt.Printf("Type: %s\n", env.Auth.Type)
	switch env.Auth.Type {
	case "basic":
		fmt.Printf("Username: %s\n", env.Auth.Username)
		if env.Auth.Password != "" {
			fmt.Println("Password: ***")
		}
	case "token":
		fmt.Printf("Token: ***\n")
		fmt.Printf("TokenType: %s\n", env.Auth.TokenType)
		fmt.Printf("Header: %s\n", env.Auth.HeaderName)
	case "oauth":
		fmt.Printf("ClientID: %s\n", env.Auth.ClientID)
		fmt.Printf("TokenURL: %s\n", env.Auth.TokenURL)
		if env.Auth.AccessToken != "" {
			fmt.Println("AccessToken: ***")
		}
	}

	return nil
}

func init() {
	repl.Register(&loginCmd{})
}
