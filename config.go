package main

import (
	"log"

	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
)

type Config struct {
	TBToken string
	DBPath  string

	TBM struct {
		Start string
	}
}

// go run ./examples/demo.go
func CreateConfig() Config {
	config.WithOptions(config.ParseEnv)

	config.AddDriver(yaml.Driver)

	err := config.LoadExists("default.yml", "custom.yml")
	if err != nil {
		log.Panic(err)
	}

	c := Config{}
	config.BindStruct("", &c)
	if err != nil {
		log.Panic(err)
	}

	return c
}
