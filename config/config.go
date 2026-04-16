package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ConfigRemote struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
}

type ConfigLocal struct {
	Addr   string `yaml:"addr"`
	Domain string `yaml:"domain"`
}

type Config struct {
	Local         ConfigLocal  `yaml:"local"`
	Remote        ConfigRemote `yaml:"remote"`
	WhitelistPath string       `yaml:"whitelist_path"`
}

type WhitelistEntry struct {
	PasswordHash string `yaml:"password_hash"`
}

type Whitelist struct {
	Entries map[string]WhitelistEntry `yaml:"entries"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	return &cfg, yaml.Unmarshal(data, &cfg)
}

func LoadWhitelist(path string) (*Whitelist, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wl Whitelist
	return &wl, yaml.Unmarshal(data, &wl)
}
