package templates

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/hitalos/minioUp/cmd/server/i18n"
)

var (
	funcs = template.FuncMap{
		"humanize":  HumanizeBytes,
		"i18n":      i18n.Translate,
		"urlPrefix": getURLPrefix,
	}

	urlPrefix string
)

func HumanizeBytes(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

func getURLPrefix() string {
	return urlPrefix
}

func SetURLPrefix(prefix string) {
	urlPrefix = strings.TrimSuffix(prefix, "/")
}
