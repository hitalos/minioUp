//go:build dev

package public

import "net/http"

var (
	Handler = http.FileServer(http.Dir("./cmd/server/public"))
)
