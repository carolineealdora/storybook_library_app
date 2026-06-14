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
	DBName      string `env:"DB_NAME" envDefault:"ms-approval"`
	DBPort      string `env:"DB_PORT" envDefault:"5432"`
	DBSSLMode   string `env:"DB_SSL_MODE" envDefault:"disable"`
	DBTimezone  string `env:"DB_TIMEZONE" envDefault:"UTC"`
	DBDebugMode bool   `env:"DB_DEBUG_MODE" envDefault:"false"`

	//Redis cache
	RedisHost     string `env:"REDIS_HOST" envDefault:"127.0.0.1"`
	RedisPort     string `env:"REDIS_PORT" envDefault:"6379"`
	RedisPassword string `env:"REDIS_PASSWORD" envDefault:"123456"`

	//Mongo
	MongoUri        string `env:"MONGO_URI" envDefault:"mongodb://localhost:27017"`
	MongoDb         string `env:"MONGO_DB" envDefault:"doc"`
	MongoCollection string `env:"MONGO_COLLECTION" envDefault:"logs"`

	//Method
	LoggingAllowedMethod string `env:"LOGGING_ALLOWED_METHOD" envDefault:"POST,PUT,DELETE,PATCH"`
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
