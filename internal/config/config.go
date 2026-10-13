package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Service    Service    `yaml:"service"`
	HTTPServer HTTPConfig `yaml:"http_server"`
	LLM        LLM        `yaml:"llm"`
}

type HTTPConfig struct {
	Port int `yaml:"http_port"`
}

type LLM struct {
	BaseURL       string        `yaml:"base_url"`
	Model         string        `yaml:"model"`
	APIKey        string        `yaml:"api_key"`
	Timeout       time.Duration `yaml:"timeout"`
	StreamTimeout time.Duration `yaml:"stream_timeout"`
}

type Service struct {
	Name string `yaml:"name"`
	Env  string `yaml:"env"`
}

// Load reads config from path (or CONFIG_PATH / default configs/env.prod.yaml),
// then overrides openai.api_key from OPENAI_API_KEY if set.
func Load() (Config, error) {
	path := os.Getenv("CONFIG_PATH")

	if path == "" {
		return Config{}, errors.New("CONFIG_PATH environment variable not set")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file: %w", err)
	}

	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		cfg.LLM.APIKey = key
	}

	if err = cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validate() error {
	if c.HTTPServer.Port <= 0 {
		return fmt.Errorf("invalid http server port: %d", c.HTTPServer.Port)
	}
	if c.LLM.Timeout <= 0 {
		return fmt.Errorf("http.timeout must be > 0")
	}
	if c.LLM.BaseURL == "" {
		return fmt.Errorf("openai.base_url is required")
	}
	if c.LLM.Model == "" {
		return fmt.Errorf("openai.model is required")
	}
	if c.LLM.APIKey == "" {
		return fmt.Errorf("openai.api_key is required (set in yaml or OPENAI_API_KEY)")
	}
	return nil
}
