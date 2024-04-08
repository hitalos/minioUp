package auth

import (
	"net/http"

	"github.com/hitalos/minioUp/cmd/server/middlewares/auth/cookie"
	"github.com/hitalos/minioUp/cmd/server/middlewares/auth/reverseProxy"
	"github.com/hitalos/minioUp/config"
)

type (
	Authenticator interface {
		New(map[string]string) func(http.Handler) http.Handler
	}
)

func NewAuthenticator(cfg config.Config) func(http.Handler) http.Handler {
	var authenticator Authenticator

	switch cfg.Auth.Driver {
	case "cookie":
		authenticator = cookie.CookieAuthenticator{}
	case "reverseProxy":
		authenticator = reverseProxy.ReverseProxyAuthenticator{}
	default:
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return authenticator.New(cfg.Auth.Params)
}
