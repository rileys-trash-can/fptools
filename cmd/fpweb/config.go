package main

import (
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"sync"
)

type Config struct {
	PrinterHost  string `yaml:"printer.host"`
	PrinterPort  string `yaml:"printer.port"`
	PrinterCType string `yaml:"printer.type"`

	Listen        string `yaml:"listen"`
	Saves         string `yaml:"saves"`
	MaxPrintCount uint   `yaml:"maxpfcount"`
}

var config *Config
var configOnce sync.Once

func readConfig() {
	f, err := os.Open(*ConfigPath)
	if err != nil {
		log.Fatalf("Failed to open file at %s: %s", *ConfigPath, err)
	}

	defer f.Close()

	config = new(Config)
	dec := yaml.NewDecoder(f)
	err = dec.Decode(config)
	if err != nil {
		log.Fatalf("Failed to decode config at %s: %s", *ConfigPath, err)
	}
}

func GetConfig() *Config {
	configOnce.Do(readConfig)

	return config
}
