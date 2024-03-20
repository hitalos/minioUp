//go:build dev

package i18n

import (
	"encoding/json"
	"log"
	"os"
)

var (
	defaultLocale = "en"
)

var translations = map[string]string{}

func Translate(text string) string {
	f, err := os.Open("cmd/server/i18n/locales/" + defaultLocale + ".json")
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&translations); err != nil {
		log.Println(err)
	}

	t, ok := translations[text]
	if !ok || t == "" {
		return "_" + text + "_"
	}

	return t
}
