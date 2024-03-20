//go:build !dev

package templates

import (
	"embed"
	"html/template"
	"io"

	"github.com/Masterminds/sprig/v3"
)

var (
	//go:embed *.html */*.html
	fs embed.FS

	tmpls = template.Must((template.New("").
		Funcs(sprig.FuncMap()).
		Funcs(funcs).
		ParseFS(fs, "*.html", "**/*.html")))
)

func Exec(w io.Writer, name string, data any) error {
	return tmpls.ExecuteTemplate(w, name, data)
}
