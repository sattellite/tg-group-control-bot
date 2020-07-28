package config

import "github.com/caarlos0/env"

// Create parsing environment to config struct
func Create(c *Config) error {
	if err := env.Parse(c); err != nil {
		return err
	}
	return nil
}
