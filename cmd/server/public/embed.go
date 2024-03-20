//go:build !dev

package public

import (
	"embed"
	"net/http"
)

var (
	//go:embed assets
	fs embed.FS

	Handler = http.FileServer(http.FS(fs))
)
