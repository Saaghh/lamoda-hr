package config

import (
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	BindAddress string `env:"BIND_ADDR" env-default:":8080"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"debug"`

	PGHost     string `env:"PG_HOST" env-default:"localhost"`
	PGPort     string `env:"PG_PORT" env-default:"5432"`
	PGDatabase string `env:"PG_DATABASE" env-default:"postgres"`
	PGUser     string `env:"PG_USER" env-default:"user"`
	PGPassword string `env:"PG_PASSWORD" env-default:"secret"`

	DeactivatorPeriod string `env:"DEACTIVATOR_PERIOD" env-default:"5m"`
}

func New() *Config {
	cfg := Config{}

	err := cleanenv.ReadEnv(&cfg)
	if err != nil {
		panic("error getting config")
	}

	return &cfg
}
