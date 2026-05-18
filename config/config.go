package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	Routes []Route      `yaml:"routes"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type Route struct {
	SlackPath string        `yaml:"slack_path"`
	DingTalk  DingTalkConfig `yaml:"dingtalk"`
}

type DingTalkConfig struct {
	Webhook string `yaml:"webhook"`
	Secret  string `yaml:"secret"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if len(cfg.Routes) == 0 {
		return nil, fmt.Errorf("no routes configured")
	}
	for i, r := range cfg.Routes {
		if r.SlackPath == "" {
			return nil, fmt.Errorf("route[%d]: slack_path is required", i)
		}
		if r.DingTalk.Webhook == "" {
			return nil, fmt.Errorf("route[%d]: dingtalk.webhook is required", i)
		}
	}
	return &cfg, nil
}
