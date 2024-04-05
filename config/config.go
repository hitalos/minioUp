package config

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

var (
	ErrWrongFileExt = errors.New("wrong file extension")
)

type (
	Config struct {
		Endpoint     string        `yaml:"endpoint" validate:"required,hostname"`
		Secure       bool          `yaml:"secure"`
		AccessKey    string        `yaml:"accessKey" validate:"required"`
		SecretKey    string        `yaml:"secretKey" validate:"required"`
		Destinations []Destination `yaml:"destinations" validate:"required,dive"`
		AllowedHosts []string      `yaml:"allowedHosts" validate:"min=1,dive,hostname_port"`
		URLPrefix    string        `yaml:"urlPrefix"`
	}

	Destination struct {
		Name         string    `yaml:"name" validate:"required"`
		Bucket       string    `yaml:"bucket" validate:"required"`
		Prefix       string    `yaml:"prefix"`
		AllowedTypes []string  `yaml:"allowedTypes" validate:"min=1"`
		Template     *Template `yaml:"template"`
		WebHook      *WebHook  `yaml:"webhook"`
	}

	Template struct {
		Model       string         `yaml:"model"`
		Description string         `yaml:"description"`
		Pattern     string         `yaml:"pattern"`
		regex       *regexp.Regexp `yaml:"-"`
		Example     string         `yaml:"example"`
	}

	WebHook struct {
		URL     string            `yaml:"url" validate:"required,url"`
		Method  string            `yaml:"method"`
		Headers map[string]string `yaml:"headers"`
		Fields  map[string]string `yaml:"fields"`
	}
)

func (t *Template) Validate(s string) bool {
	if t.Pattern == "" {
		return true
	}

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

	for i := range c.Destinations {
		if c.Destinations[i].Name == "" {
			c.Destinations[i].Name = c.Destinations[i].Bucket
		}
	}

	return nil
}

func (c *Config) Parse(configFile string) error {
	if err := c.Load(configFile); err != nil {
		return err
	}

	validate := validator.New(validator.WithRequiredStructEnabled())

	if err := validate.Struct(c); err != nil {
		return errors.New(`error validating config: "` + err.Error() + `"`)
	}

	for i, d := range c.Destinations {
		if d.Template != nil && d.Template.Pattern != "" {
			var err error
			c.Destinations[i].Template.regex, err = regexp.Compile(d.Template.Pattern)
			if err != nil {
				return errors.New(err.Error() + ` on destination "` + d.Name + `"`)
			}
		}
	}

	return nil
}
