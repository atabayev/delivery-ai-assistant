package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultPath = "cicd/vars/env.prod.yaml"

type Config struct {
	HTTP   HTTPConfig   `yaml:"http"`
	OpenAI OpenAIConfig `yaml:"openai"`
}

type HTTPConfig struct {
	Timeout time.Duration `yaml:"timeout"`
}

type OpenAIConfig struct {
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
	APIKey  string `yaml:"api_key"`
}

// Load reads config from path (or CONFIG_PATH / default configs/env.prod.yaml),
// then overrides openai.api_key from OPENAI_API_KEY if set.
func Load(path string) (Config, error) {
	if path == "" {
		path = os.Getenv("CONFIG_PATH")
	}

	if path == "" {
		path = defaultPath
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
		cfg.OpenAI.APIKey = key
	}

	if err = cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validate() error {
	if c.HTTP.Timeout <= 0 {
		return fmt.Errorf("http.timeout must be > 0")
	}
	if c.OpenAI.BaseURL == "" {
		return fmt.Errorf("openai.base_url is required")
	}
	if c.OpenAI.Model == "" {
		return fmt.Errorf("openai.model is required")
	}
	if c.OpenAI.APIKey == "" {
		return fmt.Errorf("openai.api_key is required (set in yaml or OPENAI_API_KEY)")
	}
	return nil
}
