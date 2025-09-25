package i18n

import (
	"encoding/json"
	"log/slog"
	"os"
)

var (
	defaultLocale = "en"
	translations  = map[string]string{}
)

func LoadTranslations() {
	if lang, ok := os.LookupEnv("LANG"); ok {
		defaultLocale = lang
	}

	f, err := locales.Open("locales/" + defaultLocale + ".json")
	if err != nil {
		slog.Error("error reading translations", "error", err)

		return
	}
	defer func() { _ = f.Close() }()

	if err := json.NewDecoder(f).Decode(&translations); err != nil {
		slog.Error("error decoding translations", "error", err)
	}
}
