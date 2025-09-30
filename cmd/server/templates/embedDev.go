//go:build dev

package templates

import (
	"html/template"
	"io"
	"os"

	"github.com/Masterminds/sprig/v3"
)

func parseTemplates() (*template.Template, error) {
	var (
		err error
		fs  = os.DirFS("./cmd/server/templates")
	)

	tmpls := template.New("").
		Funcs(sprig.FuncMap()).
		Funcs(funcs)
	tmpls, err = tmpls.ParseFS(fs, "*.html", "*/*.html")
	if err != nil {
		return nil, err
	}

	return tmpls, nil
}

func Exec(w io.Writer, name string, data any) error {
	tmpl, err := parseTemplates()
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(w, name, data)
}
