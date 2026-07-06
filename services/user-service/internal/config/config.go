package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort     string `mapstructure:"SERVER_PORT"`
	DBHost         string `mapstructure:"DB_HOST"`
	DBPort         string `mapstructure:"DB_PORT"`
	DBUser         string `mapstructure:"DB_USER"`
	DBPassword     string `mapstructure:"DB_PASSWORD"`
	DBName         string `mapstructure:"DB_NAME"`
	JWTSecret      string `mapstructure:"JWT_SECRET"`
	JWTExpiryHours int    `mapstructure:"JWT_EXPIRY_HOURS"`
}

func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Println("No .env file found, reading from environment variables")
	}

	viper.AutomaticEnv()

	viper.SetDefault("SERVER_PORT", "8001")
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_PASSWORD", "postgres")
	viper.SetDefault("DB_NAME", "users_db")
	viper.SetDefault("JWT_SECRET", "local-dev-secret-change-in-production")
	viper.SetDefault("JWT_EXPIRY_HOURS", 24)

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}