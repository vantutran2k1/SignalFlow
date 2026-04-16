package config

import "github.com/caarlos0/env/v11"

type Config struct {
	ListenAddr  string `env:"LISTEN_ADDR" envDefault:":8080"`
	DatabaseURL string `env:"DATABASE_URL,required"`
	JWTSecret   string `env:"JWT_SECRET" envDefault:"change-me"`
	LogFormat   string `env:"LOG_FORMAT" envDefault:"text"`
	WorkerCount int    `env:"WORKER_COUNT" envDefault:"10"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	
	return cfg, nil
}
