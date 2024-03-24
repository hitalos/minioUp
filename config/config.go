package config

import (
	"errors"
	"os"
	"path/filepath"

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
	}

	Destination struct {
		Name         string    `yaml:"name"`
		Bucket       string    `yaml:"bucket"`
		Prefix       string    `yaml:"prefix"`
		AllowedTypes []string  `yaml:"allowedTypes"`
		Template     *Template `yaml:"template"`
	}

	Template struct {
		Model       string `yaml:"model"`
		Description string `yaml:"description"`
		Regex       string `yaml:"regex"`
		Example     string `yaml:"example"`
	}
)

func (c *Config) Load(configFile string) error {
	ext := filepath.Ext(configFile)
	if ext != ".yml" && ext != ".yaml" {
		return ErrWrongFileExt
	}

	f, err := os.Open(filepath.Clean(configFile))
	if err != nil {
		return err
	}

	return yaml.NewDecoder(f).Decode(c)
}
