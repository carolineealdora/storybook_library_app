package config

import (
	"fmt"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

type SystemConfig struct {
	Env      string `env:"ENV" envDefault:"development"`
	Port     string `env:"PORT" envDefault:"9999"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"INFO"`

	//DB
	DBHost      string `env:"DB_HOST" envDefault:"localhost"`
	DBUsername  string `env:"DB_USERNAME" envDefault:"postgres"`
	DBPassword  string `env:"DB_PASSWORD" envDefault:"postgres"`
	DBName      string `env:"DB_NAME" envDefault:"storybook"`
	DBPort      string `env:"DB_PORT" envDefault:"5432"`
	DBSSLMode   string `env:"DB_SSL_MODE" envDefault:"disable"`
	DBTimezone  string `env:"DB_TIMEZONE" envDefault:"UTC"`
	DBDebugMode bool   `env:"DB_DEBUG_MODE" envDefault:"false"`

	// Auth — no default on purpose: the service refuses to start without it.
	JWTSecret string `env:"JWT_SECRET"`

	// Browser clients (Library app + Book Management app) live on other origins.
	CORSAllowOrigins []string `env:"CORS_ALLOW_ORIGINS" envDefault:"*" envSeparator:","`
}

func LoadEnv() *SystemConfig {
	cfg := SystemConfig{}
	err := godotenv.Load()
	if err != nil {
		fmt.Println(err, ", getting config from ENV")
	}
	err = env.Parse(&cfg)
	if err != nil {
		fmt.Println(err, ", getting config from ENV")
	}
	return &cfg
}
