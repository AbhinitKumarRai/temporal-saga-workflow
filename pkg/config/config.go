package config

import (
	"time"

	"github.com/caarlos0/env/v10"
)

type Config struct {
	TemporalAddress           string        `env:"TEMPORAL_ADDRESS" envDefault:"temporal:7233"`
	TemporalNamespace         string        `env:"TEMPORAL_NAMESPACE" envDefault:"default"`
	TemporalTaskQueue         string        `env:"TEMPORAL_TASK_QUEUE" envDefault:"saga-task-queue"`
	TransactionTimeoutSeconds int           `env:"TRANSACTION_TIMEOUT_SECONDS" envDefault:"30"`
	API1BaseURL               string        `env:"API1_BASE_URL" envDefault:"https://crudcrud.com/api/4adaea1377ae42358470ccbd5472cf15"`
	API2BaseURL               string        `env:"API2_BASE_URL" envDefault:"https://crudcrud.com/api/d379fa9d675b4269803fc0f108f5a3eb"`
	API3BaseURL               string        `env:"API3_BASE_URL" envDefault:"https://crudcrud.com/api/4adaea1377ae42358470ccbd5472cf15"`
	MockMode                  bool          `env:"MOCK_MODE" envDefault:"true"`
	HTTPTimeoutSeconds        int           `env:"HTTP_TIMEOUT_SECONDS" envDefault:"10"`
	ServerPort                string        `env:"SERVER_PORT" envDefault:"8080"`
	// Derived
	httpTimeout               time.Duration `env:"-"`
}

func Load() (Config, error) {
	c := Config{}
	if err := env.Parse(&c); err != nil {
		return c, err
	}
	c.httpTimeout = time.Duration(c.HTTPTimeoutSeconds) * time.Second
	return c, nil
}

func (c Config) TransactionTimeout() time.Duration {
	return time.Duration(c.TransactionTimeoutSeconds) * time.Second
}

func (c *Config) HTTPTimeout() time.Duration {
	if c.httpTimeout == 0 {
		c.httpTimeout = time.Duration(c.HTTPTimeoutSeconds) * time.Second
	}
	return c.httpTimeout
}


