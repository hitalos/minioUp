package cookie

import (
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"

	"github.com/hitalos/minioUp/cmd/server/handlers"
	"github.com/hitalos/minioUp/cmd/server/templates"
)

type (
	CookieAuthenticator struct {
		store      *sessions.CookieStore
		cookieName string
		users      map[string]struct {
			PasswordHash string   `yaml:"passwordHash"`
			Roles        []string `yaml:"roles"`
		}
		urlPrefix string
		skipPaths []string
	}
)

func (cAuth CookieAuthenticator) New(params map[string]string) func(http.Handler) http.Handler {
	cAuth.store = sessions.NewCookieStore([]byte(params["secret"]))
	cAuth.store.Options = &sessions.Options{
		HttpOnly: true,
		MaxAge:   3600,
		Path:     cAuth.urlPrefix + "/",
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
	}

	if os.Getenv("ENV") == "dev" {
		cAuth.store.Options.SameSite = http.SameSiteLaxMode
		cAuth.store.Options.Secure = false
	}

	if maxAge, ok := params["cookieMaxAge"]; ok {
		if age, err := strconv.Atoi(maxAge); err == nil && age > 0 {
			cAuth.store.Options.MaxAge = age
		}
	}

	cAuth.cookieName = params["cookieName"]

	if cAuth.cookieName == "" {
		cAuth.cookieName = "minioUp-session"
	}

	f, err := os.Open("users.yml")
	if err != nil {
		slog.Error("Error opening users.yml", "error", err)
		os.Exit(1)
	}

	if err := yaml.NewDecoder(f).Decode(&cAuth.users); err != nil {
		slog.Error("Error decoding users.yml", "error", err)
		os.Exit(1)
	}

	if urlPrefixParam, ok := params["urlPrefix"]; ok && urlPrefixParam != "" {
		cAuth.urlPrefix = urlPrefixParam
	}

	if paths, ok := params["skipPaths"]; ok && paths != "" {
		cAuth.skipPaths = strings.Split(paths, ",")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, path := range cAuth.skipPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			if cAuth.isAuthenticated(r) {
				switch r.URL.Path {
				case "/auth/logout":
					cAuth.logout(w, r)
					return
				case "/auth/login":
					w.Header().Set("Location", cAuth.urlPrefix+"/")
					w.WriteHeader(http.StatusSeeOther)
					return
				}

				next.ServeHTTP(w, r)
				return
			}

			if r.URL.Path == "/auth/login" {
				if r.Method == http.MethodPost {
					cAuth.processLogin(w, r)
					return
				}

				cAuth.showLogin(w, r)
				return
			}

			w.Header().Set("Location", cAuth.urlPrefix+"/auth/login")
			w.WriteHeader(http.StatusSeeOther)
		})
	}
}

func (cAuth CookieAuthenticator) showLogin(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
	d := map[string]string{"Title": "Login"}
	if err := templates.Exec(w, "login.html", d); err != nil {
		handlers.ErrorHandler("Error executing template", err, w, http.StatusInternalServerError)
	}
}

func (cAuth CookieAuthenticator) processLogin(w http.ResponseWriter, r *http.Request) {
	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

	if username == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest)
		cAuth.showLogin(w, r)
		return
	}

	u, ok := cAuth.users[username]
	if !ok {
		slog.Error("Error authenticating user")
		w.WriteHeader(http.StatusUnauthorized)
		cAuth.showLogin(w, r)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		slog.Error("Error authenticating user")
		w.WriteHeader(http.StatusUnauthorized)
		cAuth.showLogin(w, r)
		return
	}

	session, err := cAuth.store.New(r, cAuth.cookieName)
	if err != nil {
		slog.Error("Error creating session", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	session.Values["username"] = username
	session.Values["roles"] = u.Roles

	if err := session.Save(r, w); err != nil {
		slog.Error("Error saving session", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", cAuth.urlPrefix+"/")
	w.WriteHeader(http.StatusSeeOther)
}

func (cAuth CookieAuthenticator) logout(w http.ResponseWriter, r *http.Request) {
	session, err := cAuth.store.Get(r, cAuth.cookieName)
	if err != nil {
		slog.Error("Error getting session", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		slog.Error("Error saving session", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", cAuth.urlPrefix+"/auth/login")
	w.WriteHeader(http.StatusSeeOther)
}

func (cAuth CookieAuthenticator) isAuthenticated(r *http.Request) bool {
	session, err := cAuth.store.Get(r, cAuth.cookieName)
	if err != nil {
		slog.Error("Error getting session", "error", err)
		return false
	}

	username, ok := session.Values["username"].(string)
	if ok && username != "" {
		r.Header.Set("X-Forwarded-Preferred-Username", username)
		r.Header.Set("X-Roles", strings.Join(session.Values["roles"].([]string), ","))

		return true
	}

	return false
}
