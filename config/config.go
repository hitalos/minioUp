package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type (
	Config struct {
		Endpoint     string        `yaml:"endpoint"`
		Secure       bool          `yaml:"secure"`
		AccessKey    string        `yaml:"accessKey"`
		SecretKey    string        `yaml:"secretKey"`
		Dest         Destination   `yaml:"destination"`
		Destinations []Destination `yaml:"destinations"`
	}

	Destination struct {
		Name     string `yaml:"name"`
		Bucket   string `yaml:"bucket"`
		Prefix   string `yaml:"prefix"`
		Template string `yaml:"template"`
	}
)

func (c *Config) Load(configFile string) error {
	f, err := os.Open(configFile)
	if err != nil {
		return err
	}

	return yaml.NewDecoder(f).Decode(c)
}
