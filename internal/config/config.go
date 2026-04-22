// Package config
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/henrywhitaker3/windowframe/config"
)

type Endpoint struct {
	Path string `yaml:"path"`
	// Supports go templating with a .Request object
	// e.g. .Request.body.field
	Response string `yaml:"response"`

	StatusCode int `yaml:"status_code"`

	// When both are set, a random delay is added before responding.
	// 95% of delays will fall at or below P95Delay, with MaxDelay
	// as the absolute ceiling.
	MinDelay time.Duration `yaml:"minDelay"`
	MaxDelay time.Duration `yaml:"maxDelay"`
	P95Delay time.Duration `yaml:"p95Delay"`
}

type Config struct {
	Endpoints []Endpoint `yaml:"endpoints"`
}

func Load(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	conf, err := config.NewParser[Config]().WithExtractors(
		config.NewYamlExtractor[Config](file),
	).Parse()
	if err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	return &conf, err
}
