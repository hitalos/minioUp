package i18n

import (
	"encoding/json"
	"log"
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
		log.Println(err)
		return
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&translations); err != nil {
		log.Println(err)
	}
}
