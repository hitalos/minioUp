//go:build dev

package templates

import (
	"html/template"
	"io"

	"github.com/Masterminds/sprig/v3"
)

func parseTemplates() (*template.Template, error) {
	var err error
	tmpls := template.New("").
		Funcs(sprig.FuncMap()).
		Funcs(funcs)
	tmpls, err = tmpls.ParseGlob("cmd/server/templates/*.html")
	if err != nil {
		return nil, err
	}

	tmpls, err = tmpls.ParseGlob("cmd/server/templates/*/*.html")
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
