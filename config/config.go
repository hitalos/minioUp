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
		Name         string           `yaml:"name" validate:"required"`
		Bucket       string           `yaml:"bucket" validate:"required"`
		Prefix       string           `yaml:"prefix"`
		AllowedRoles []string         `yaml:"allowedRoles"`
		AllowedTypes []string         `yaml:"allowedTypes"`
		Fields       map[string]Field `yaml:"fields" validate:"dive"`
		WebHook      *WebHook         `yaml:"webhook"`
		Model        string           `yaml:"model"`
		modelTmpl    *template.Template
	}

	Field struct {
		Type        string         `yaml:"type"`
		IsRequired  bool           `yaml:"required"`
		Value       string         `yaml:"value"`
		Description string         `yaml:"description" validate:"required"`
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

func (f Field) Validate() bool {
	if f.Pattern == "" {
		return true
	}

	return f.regex.MatchString(f.Value)
}

func (d Destination) MountName(params map[string]string) string {
	if d.Model == "" {
		return filepath.Base(params["originalFilename"])
	}

	str := new(bytes.Buffer)
	if err := d.modelTmpl.Execute(str, params); err != nil {
		slog.Error("error executing template to mount filename", "error", err)
		return filepath.Base(params["originalFilename"])
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

	names := []string{}
	for i, d := range c.Destinations {
		for _, name := range names {
			if name == d.Name {
				return errors.New("duplicate destination name: " + d.Name)
			}
		}
		names = append(names, d.Name)

		if len(d.Fields) == 0 {
			continue
		}

		if d.Model != "" {
			tmpl, err := template.New("").Funcs(sprig.FuncMap()).Parse(d.Model)
			if err != nil {
				return errors.New(err.Error() + ` on model ofdestination "` + d.Name + `"`)
			}
			c.Destinations[i].modelTmpl = tmpl
		}

		for fieldName, f := range d.Fields {
			if f.Pattern != "" {
				reg, err := regexp.Compile(f.Pattern)
				if err != nil {
					return errors.New(err.Error() + ` on pattern of destination "` + d.Name + `"`)
				}
				f.regex = reg
				c.Destinations[i].Fields[fieldName] = f
			}
		}
	}

	return nil
}
