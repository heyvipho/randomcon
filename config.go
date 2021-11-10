package main

import (
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
)

type Config struct {
	TBToken  string
	DataPath string

	TBMessages Messages
}

// go run ./examples/demo.go
func CreateConfig() (*Config, error) {
	c := Config{}

	config.WithOptions(config.ParseEnv)

	config.AddDriver(yaml.Driver)

	err := config.LoadExists("default.yml", "custom.yml")
	if err != nil {
		return &c, err
	}

	err = config.BindStruct("", &c)

	return &c, err
}
