package config

import (
	"bytes"
	"encoding/json"
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

const (
	MAX_RESULT_LEN = 10
	MAX_SIZE_LIMIT = 100 << 20 // 100 MB
)

var (
	ErrWrongFileExt = errors.New("wrong file extension")
)

type (
	Config struct {
		Port         string        `yaml:"port" json:"port" validate:"required,hostname_port"`
		Endpoint     string        `yaml:"endpoint" json:"endpoint" validate:"required,hostname|hostname_port"`
		Secure       bool          `yaml:"secure" json:"secure"`
		AccessKey    string        `yaml:"accessKey" json:"accessKey" validate:"required"`
		SecretKey    string        `yaml:"secretKey" json:"secretKey" validate:"required"`
		Destinations []Destination `yaml:"destinations" json:"destinations" validate:"required,dive"`
		AllowedHosts []string      `yaml:"allowedHosts,omitempty" json:"allowedHosts,omitempty" validate:"dive,hostname_port|hostname"`
		URLPrefix    string        `yaml:"urlPrefix,omitempty" json:"urlPrefix,omitempty"`
		Auth         Auth          `yaml:"auth" json:"auth"`
	}

	Auth struct {
		Driver string            `yaml:"driver" json:"driver"`
		Params map[string]string `yaml:"params" json:"params"`
	}
	Destination struct {
		Name            string           `yaml:"name" json:"name" validate:"required"`
		Bucket          string           `yaml:"bucket" json:"bucket" validate:"required"`
		Prefix          string           `yaml:"prefix,omitempty" json:"prefix,omitempty"`
		AllowedRoles    []string         `yaml:"allowedRoles,omitempty" json:"allowedRoles,omitempty"`
		AllowedTypes    []string         `yaml:"allowedTypes,omitempty" json:"allowedTypes,omitempty"`
		Fields          map[string]Field `yaml:"fields,omitempty" json:"fields,omitempty" validate:"dive"`
		WebHook         *WebHook         `yaml:"webhook,omitempty" json:"webhook,omitempty"`
		Model           *TemplateString  `yaml:"model,omitempty" json:"model,omitempty"`
		MaxResultLength int              `yaml:"maxResultLength,omitempty" json:"maxResultLength,omitempty" validate:"omitempty,min=1,max=1000"`
		MaxUploadSize   int64            `yaml:"maxUploadSize,omitempty" json:"maxUploadSize,omitempty" validate:"omitempty,min=1"`
	}

	Field struct {
		Type        string `yaml:"type,omitempty" json:"type,omitempty"`
		IsRequired  bool   `yaml:"required,omitempty" json:"required,omitempty"`
		Value       string `yaml:"value,omitempty" json:"value,omitempty"`
		Description string `yaml:"description" json:"description" validate:"required"`
		Pattern     string `yaml:"pattern,omitempty" json:"pattern,omitempty"`
		regex       *regexp.Regexp
		Example     *TemplateString `yaml:"example,omitempty" json:"example,omitempty"`
	}

	TemplateString struct {
		params   any
		template *template.Template
		Value    string `yaml:"value" json:"value" validate:"required"`
	}

	WebHook struct {
		URL     string            `yaml:"url" json:"url" validate:"required,url"`
		Method  string            `yaml:"method,omitempty" json:"method,omitempty"`
		Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
		Fields  map[string]string `yaml:"fields,omitempty" json:"fields,omitempty"`
	}
)

func (t *TemplateString) UnmarshalYAML(v *yaml.Node) error {
	var err error
	t.Value = v.Value
	t.template, err = template.New("").Funcs(sprig.GenericFuncMap()).Parse(t.Value)

	return err
}

func (t TemplateString) String() string {
	buf := new(bytes.Buffer)
	if err := t.template.Execute(buf, t.params); err != nil {
		slog.Error("error executing template", "error", err)
		return t.Value
	}

	return buf.String()
}

func (f Field) Validate() bool {
	if f.Pattern == "" {
		return true
	}

	return f.regex.MatchString(f.Value)
}

func (d Destination) MountName(params map[string]string) string {
	d.Model.params = params
	return d.Model.String()
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

		if c.Destinations[i].MaxResultLength == 0 {
			c.Destinations[i].MaxResultLength = MAX_RESULT_LEN
		}

		if c.Destinations[i].MaxUploadSize == 0 {
			c.Destinations[i].MaxUploadSize = MAX_SIZE_LIMIT
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

func (c Config) ToJSON() string {
	b, err := json.Marshal(c)
	if err != nil {
		return err.Error()
	}

	return string(b)
}

func (c Config) ToYAML() string {
	b, err := yaml.Marshal(c)
	if err != nil {
		return err.Error()
	}

	return string(b)
}

func (c Config) String() string {
	return c.ToYAML()
}
