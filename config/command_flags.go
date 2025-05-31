package config

import "flag"

var (
	configFilePath string
)

func init() {
	flag.StringVar(&configFilePath, "config", "./config.yml", "Path to PixivFE's config file")
}

func parseCommandLineArgs() {
	flag.Parse()
}

