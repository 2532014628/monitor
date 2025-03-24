package configs

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
)

type Config struct {
	Url    string `yaml:"url"`
	Port   string `yaml:"port"`
	Second int    `yaml:"second"`
}

func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open config file: %v", err)
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("unable to parse config file: %v", err)
	}
	return &config, nil
}
