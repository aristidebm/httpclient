package executor

import (
	"encoding/base64"
	"net/http"

	"httpclient/internal/model"
)

func ApplyAuth(req *http.Request, env *model.Environment) {
	if env.Auth == nil {
		return
	}

	switch env.Auth.Type {
	case "basic":
		creds := base64.StdEncoding.EncodeToString([]byte(env.Auth.Username + ":" + env.Auth.Password))
		req.Header.Set("Authorization", "Basic "+creds)

	case "token":
		tokenType := env.Auth.TokenType
		if tokenType == "" {
			tokenType = "Bearer"
		}
		headerName := env.Auth.HeaderName
		if headerName == "" {
			headerName = "Authorization"
		}
		req.Header.Set(headerName, tokenType+" "+env.Auth.Token)

	case "oauth":
		if env.Auth.AccessToken != "" {
			req.Header.Set("Authorization", "Bearer "+env.Auth.AccessToken)
		}
	}
}
