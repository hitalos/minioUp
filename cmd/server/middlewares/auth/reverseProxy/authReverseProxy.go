package reverseProxy

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/hitalos/minioUp/cmd/server/templates"
)

type (
	ReverseProxyAuthenticator struct {
		clientID string
		header   string
	}
)

func (rpAuth ReverseProxyAuthenticator) New(params map[string]string) func(http.Handler) http.Handler {
	rpAuth.header = "X-Forwarded-Groups"
	clientID, ok := params["clientID"]
	if !ok || clientID == "" {
		slog.Error("Missing clientID param of oauth2proxy driver")
		os.Exit(1)
	}
	rpAuth.clientID = clientID

	urlPrefix := ""
	if urlPrefixParam, ok := params["urlPrefix"]; ok && urlPrefixParam != "" {
		urlPrefix = urlPrefixParam
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/auth/logout" {
				http.Redirect(w, r, urlPrefix+"/oauth2/sign_out", http.StatusTemporaryRedirect)
				return
			}

			if r.Header.Get(rpAuth.header) == "" && r.Header.Get("X-Forwarded-Preferred-Username") == "" {
				err := fmt.Errorf("missing header")
				slog.Error("Authorization error", "error", err)
				w.WriteHeader(http.StatusForbidden)

				if err := templates.Exec(w, "error.html", err.Error()); err != nil {
					slog.Error("Error executing template", "error", err)
				}

				return
			}

			rolePrefix := fmt.Sprintf("role:%s:", clientID)
			for _, v := range strings.Split(r.Header.Get(rpAuth.header), ",") {
				if len(v) < len(rolePrefix) {
					continue
				}

				if strings.HasPrefix(v, rolePrefix) {
					r.Header.Add("X-Roles", v[len(rolePrefix):])
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
