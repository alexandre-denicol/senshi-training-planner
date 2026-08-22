package config

import (
	"errors"
	"os"
)

const defaultPort = "8080"

type AppEnv string

const (
	AppEnvDevelopment AppEnv = "development"
	AppEnvProduction  AppEnv = "production"
)

var ErrInvalidAppEnv = errors.New("APP_ENV must be development or production")

type Config struct {
	Port        string
	DatabaseURL string
	AppEnv      AppEnv
}

func Load() (Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	appEnv := AppEnv(os.Getenv("APP_ENV"))
	if appEnv == "" {
		appEnv = AppEnvProduction
	}
	if appEnv != AppEnvDevelopment && appEnv != AppEnvProduction {
		return Config{}, ErrInvalidAppEnv
	}

	return Config{
		Port:        port,
		DatabaseURL: os.Getenv("DATABASE_URL"),
		AppEnv:      appEnv,
	}, nil
}

func (c Config) IsProduction() bool {
	return c.AppEnv == AppEnvProduction
}
