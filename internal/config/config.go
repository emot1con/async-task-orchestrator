package config

import (
	"os"
)

type Config struct {
	AppName string
	AppEnv  string
	AppPort string

	DB       DBConfig
	Redis    RedisConfig
	RabbitMQ RabbitMQConfig
	JWT      JWTConfig
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type RedisConfig struct {
	Host          string
	Port          string
	RedisPassword string
	RedisDB       string
}

type RabbitMQConfig struct {
	URL string
}

type JWTConfig struct {
	Secret string
}

func Load() *Config {
	return &Config{
		AppName: os.Getenv("APP_NAME"),
		AppEnv:  os.Getenv("APP_ENV"),
		AppPort: os.Getenv("APP_PORT"),

		DB: DBConfig{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     os.Getenv("DB_NAME"),
			SSLMode:  os.Getenv("DB_SSLMODE"),
		},

		Redis: RedisConfig{
			Host:          os.Getenv("REDIS_HOST"),
			Port:          os.Getenv("REDIS_PORT"),
			RedisPassword: os.Getenv("REDIS_PASSWORD"),
			RedisDB:       os.Getenv("REDIS_DB"),
		},

		RabbitMQ: RabbitMQConfig{
			URL: os.Getenv("RABBITMQ_URL"),
		},

		JWT: JWTConfig{
			Secret: os.Getenv("JWT_SECRET"),
		},
	}
}
