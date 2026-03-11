package executor

import (
	"net/http"
	"strings"

	"cdapi/internal/model"
)

func ApplyAuth(req *http.Request, env *model.Environment) {
	if auth, ok := env.Headers["Authorization"]; ok {
		if strings.HasPrefix(auth, "TOKEN ") {
			req.Header.Set("Authorization", auth)
		} else {
			req.Header.Set("Authorization", "TOKEN "+auth)
		}
	}
}
