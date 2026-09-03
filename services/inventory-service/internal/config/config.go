package config

import (
	"log"

	"github.com/spf13/viper"
)

// Config holds all configuration for the inventory service.
type Config struct {
	// Server
	GRPCPort string `mapstructure:"GRPC_PORT"`

	// Database
	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     string `mapstructure:"DB_PORT"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBName     string `mapstructure:"DB_NAME"`

	// Redis (for distributed locking)
	RedisAddr     string `mapstructure:"REDIS_ADDR"`
	RedisPassword string `mapstructure:"REDIS_PASSWORD"`

	// Lock settings
	LockTTLSeconds int `mapstructure:"LOCK_TTL_SECONDS"`
}

func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Println("No .env file found, reading from environment variables")
	}

	viper.AutomaticEnv()

	// Defaults
	viper.SetDefault("GRPC_PORT", "50053")
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_PASSWORD", "postgres")
	viper.SetDefault("DB_NAME", "inventory_db")
	viper.SetDefault("REDIS_ADDR", "redis-service:6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("LOCK_TTL_SECONDS", 30)

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
