//go:build !dev

package i18n

import (
	"embed"
	"encoding/json"
	"log"
	"os"
)

var (
	defaultLocale = "en"

	//go:embed locales/*.json
	locales embed.FS
)

func init() {
	if lang, ok := os.LookupEnv("LANG"); ok {
		defaultLocale = lang
	}

	f, err := locales.Open("locales/" + defaultLocale + ".json")
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&translations); err != nil {
		log.Println(err)
	}
}

var translations = map[string]string{}

func Translate(text string) string {
	t, ok := translations[text]
	if ok {
		return t
	}

	return text
}
