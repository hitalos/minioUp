package config

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

var (
	ErrWrongFileExt = errors.New("wrong file extension")
)

type (
	Config struct {
		Endpoint     string        `yaml:"endpoint"`
		Secure       bool          `yaml:"secure"`
		AccessKey    string        `yaml:"accessKey"`
		SecretKey    string        `yaml:"secretKey"`
		Destinations []Destination `yaml:"destinations"`
		AllowedHosts []string      `yaml:"allowedHosts"`
		URLPrefix    string        `yaml:"urlPrefix"`
	}

	Destination struct {
		Name         string    `yaml:"name"`
		Bucket       string    `yaml:"bucket"`
		Prefix       string    `yaml:"prefix"`
		AllowedTypes []string  `yaml:"allowedTypes"`
		Template     *Template `yaml:"template"`
	}

	Template struct {
		Model       string         `yaml:"model"`
		Description string         `yaml:"description"`
		Pattern     string         `yaml:"pattern"`
		regex       *regexp.Regexp `yaml:"-"`
		Example     string         `yaml:"example"`
	}
)

func (t *Template) Validate(s string) bool {
	return t.regex.MatchString(s)
}

func (c *Config) Load(configFile string) error {
	ext := filepath.Ext(configFile)
	if ext != ".yml" && ext != ".yaml" {
		return ErrWrongFileExt
	}

	f, err := os.Open(filepath.Clean(configFile))
	if err != nil {
		return err
	}

	if err := yaml.NewDecoder(f).Decode(c); err != nil {
		return errors.New(`error decoding config: "` + err.Error() + `"`)
	}

	for i, d := range c.Destinations {
		if d.Template.Pattern != "" {
			var err error
			c.Destinations[i].Template.regex, err = regexp.Compile(d.Template.Pattern)
			if err != nil {
				return errors.New(err.Error() + ` on destination "` + d.Name + `"`)
			}
		}
	}

	return nil
}
