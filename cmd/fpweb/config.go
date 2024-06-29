package main

import (
	"flag"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"sync"
)

var (
	ConfigPath = flag.String("config", "config.yml", "config to read")

	ListenAddr = flag.String("listen", "", "specify port to listen on, fallback is [::]:8070")

	PrinterAddressHost = flag.String("host", os.Getenv("IPL_PRINTER"), "Specify printer, can also be set by env IPL_PRINTER (net port)")
	PrinterAddressPort = flag.String("port", os.Getenv("IPL_PORT"), "Specify printer, can also be set by env IPL_PORT (usb port)")

	PrinterAddressType = flag.String("ctype", os.Getenv("IPL_CTYPE"), "Specify printer connection type, can also be set by env IPL_CTYPE")

	OptVerbose = flag.Bool("verbose", false, "toggle verbose logging")
	OptBeep    = flag.Bool("beep", true, "toggle connection-beep")
	OptDryRun  = flag.Bool("dry-run", false, "disables connection to printer; for testing")
)

type Config struct {
	PrinterHost  string `yaml:"printer.host"`
	PrinterPort  string `yaml:"printer.port"`
	PrinterCType string `yaml:"printer.type"`

	Listen        string `yaml:"listen"`
	MaxPrintCount uint   `yaml:"maxpfcount"`
	DB            string `yaml:"databasepath"`
	DBType        string `yaml:"dbtype"`
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
