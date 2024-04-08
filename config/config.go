package config

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

var (
	ErrWrongFileExt = errors.New("wrong file extension")
)

type (
	Config struct {
		Port         string        `yaml:"port" validate:"required,hostname_port"`
		Endpoint     string        `yaml:"endpoint" validate:"required,hostname|hostname_port"`
		Secure       bool          `yaml:"secure"`
		AccessKey    string        `yaml:"accessKey" validate:"required"`
		SecretKey    string        `yaml:"secretKey" validate:"required"`
		Destinations []Destination `yaml:"destinations" validate:"required,dive"`
		AllowedHosts []string      `yaml:"allowedHosts" validate:"dive,hostname_port|hostname"`
		URLPrefix    string        `yaml:"urlPrefix"`
		Auth         Auth          `yaml:"auth"`
	}

	Auth struct {
		Driver string            `yaml:"driver"`
		Params map[string]string `yaml:"params"`
	}
	Destination struct {
		Name         string    `yaml:"name" validate:"required"`
		Bucket       string    `yaml:"bucket" validate:"required"`
		Prefix       string    `yaml:"prefix"`
		AllowedRoles []string  `yaml:"allowedRoles"`
		AllowedTypes []string  `yaml:"allowedTypes" validate:"min=1"`
		Template     *Template `yaml:"template"`
		WebHook      *WebHook  `yaml:"webhook"`
	}

	Template struct {
		Model       string `yaml:"model"`
		modelTmpl   *template.Template
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

func (t Template) MountName(params []string) string {
	if t.Model == "" {
		return filepath.Base(params[0])
	}

	str := new(bytes.Buffer)
	if err := t.modelTmpl.Execute(str, params); err != nil {
		slog.Error("error executing template to mount filename", err)
		return filepath.Base(params[0])
	}

	return str.String()
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

	if c.Port == "" {
		c.Port = "localhost:8000"
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
		if d.Template == nil {
			continue
		}

		if d.Template.Pattern != "" {
			var err error
			c.Destinations[i].Template.regex, err = regexp.Compile(d.Template.Pattern)
			if err != nil {
				return errors.New(err.Error() + ` on pattern of destination "` + d.Name + `"`)
			}
		}

		if d.Template.Model != "" {
			tmpl, err := template.New("").Funcs(sprig.FuncMap()).Parse(d.Template.Model)
			if err != nil {
				return errors.New(err.Error() + ` on model ofdestination "` + d.Name + `"`)
			}
			d.Template.modelTmpl = tmpl
		}
	}

	return nil
}
