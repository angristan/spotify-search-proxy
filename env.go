package main

import (
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Env struct {
	SpotifyClientID     string `env:"SPOTIFY_CLIENT_ID" env-required:"true"`
	SpotifyClientSecret string `env:"SPOTIFY_CLIENT_SECRET" env-required:"true"`

	RedisURL string `env:"REDIS_URL" env-required:"true"`

	Port string `env:"PORT" env-default:"1323"`

	LogFormat string `env:"LOG_FORMAT" env-default:"json"`
	LogLevel  string `env:"LOG_LEVEL" env-default:"info"`
}

var env Env

func LoadEnv() error {
	err := godotenv.Load()
	if err != nil {
		logrus.WithError(err).Warn("Failed to load env variables from file")
	}

	return cleanenv.ReadEnv(&env)
}

func GetEnv() *Env {
	return &env
}
