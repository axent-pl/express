package main

import (
	"os"
	"testing"
)

func TestUnmarshalEvaluatesExpressFields(t *testing.T) {
	type Config struct {
		User string `yaml:"user"`
		URL  string `yaml:"url" express:"true"`
	}

	var cfg Config

	input := []byte("user: alice\nurl: https://api/${user}\n")

	if err := Unmarshal(input, &cfg); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if cfg.URL != "https://api/alice" {
		t.Fatalf("URL = %q, want %q", cfg.URL, "https://api/alice")
	}
}

func TestUnmarshalWithEnv(t *testing.T) {
	const envKey = "AXENT_EXPRESS_ENV_TEST"
	const envValue = "from-env"

	if err := os.Setenv(envKey, envValue); err != nil {
		t.Fatalf("Setenv() error = %v", err)
	}

	t.Cleanup(func() {
		_ = os.Unsetenv(envKey)
	})

	type Config struct {
		Value string `yaml:"value" express:"true"`
	}

	var cfg Config

	input := []byte("value: ${AXENT_EXPRESS_ENV_TEST}\n")

	if err := Unmarshal(input, &cfg, WithEnv()); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if cfg.Value != envValue {
		t.Fatalf("Value = %q, want %q", cfg.Value, envValue)
	}
}

func TestUnmarshalEvaluatesNestedAndSliceFields(t *testing.T) {
	type Endpoint struct {
		Host string `yaml:"host"`
		URL  string `yaml:"url" express:"true"`
	}

	type Config struct {
		Protocol  string     `yaml:"protocol"`
		Endpoints []Endpoint `yaml:"endpoints"`
	}

	var cfg Config

	input := []byte("protocol: https\nendpoints:\n  - host: service-a\n    url: ${protocol}://${endpoints[0].host}\n")

	if err := Unmarshal(input, &cfg); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if cfg.Endpoints[0].URL != "https://service-a" {
		t.Fatalf("URL = %q, want %q", cfg.Endpoints[0].URL, "https://service-a")
	}
}
