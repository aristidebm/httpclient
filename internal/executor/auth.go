package executor

import (
	"encoding/base64"
	"net/http"

	"httpclient/internal/model"
)

func ApplyAuth(req *http.Request, session *model.Session, env *model.Environment) {
	// Check session auth first, fall back to environment auth
	auth := session.Auth
	if auth == nil {
		auth = env.Auth
	}

	if auth == nil {
		return
	}

	switch auth.Type {
	case "basic":
		creds := base64.StdEncoding.EncodeToString([]byte(auth.Username + ":" + auth.Password))
		req.Header.Set("Authorization", "Basic "+creds)

	case "token":
		tokenType := auth.TokenType
		if tokenType == "" {
			tokenType = "Bearer"
		}
		headerName := auth.HeaderName
		if headerName == "" {
			headerName = "Authorization"
		}
		req.Header.Set(headerName, tokenType+" "+auth.Token)

	case "oauth":
		if auth.AccessToken != "" {
			req.Header.Set("Authorization", "Bearer "+auth.AccessToken)
		}
	}
}
