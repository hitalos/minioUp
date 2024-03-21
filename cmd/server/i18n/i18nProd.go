//go:build !dev

package i18n

import (
	"embed"
)

var (
	//go:embed locales/*.json
	locales embed.FS
)

func Translate(text string) string {
	t, ok := translations[text]
	if ok {
		return t
	}

	return text
}
