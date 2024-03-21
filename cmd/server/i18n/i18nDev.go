//go:build dev

package i18n

import (
	"os"
)

var (
	locales = os.DirFS("cmd/server/i18n")
)

func Translate(text string) string {
	LoadTranslations()

	t, ok := translations[text]
	if !ok || t == "" {
		return "_" + text + "_"
	}

	return t
}
