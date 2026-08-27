package config

import (
	"log"

	"github.com/spf13/viper"
)

// Config holds all configuration for the product service.
type Config struct {
	// Server
	ServerPort string `mapstructure:"SERVER_PORT"`
	GRPCPort   string `mapstructure:"GRPC_PORT"`

	// MongoDB
	MongoURI string `mapstructure:"MONGO_URI"`
	MongoDB  string `mapstructure:"MONGO_DB"`

	// Redis
	RedisAddr     string `mapstructure:"REDIS_ADDR"`
	RedisPassword string `mapstructure:"REDIS_PASSWORD"`

	// JWT (for validating incoming requests)
	JWTSecret string `mapstructure:"JWT_SECRET"`

	// User Service gRPC address
	UserServiceAddr string `mapstructure:"USER_SERVICE_ADDR"`
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
	viper.SetDefault("SERVER_PORT", "8002")
	viper.SetDefault("GRPC_PORT", "50052")
	viper.SetDefault("MONGO_URI", "mongodb://mongo:mongo@mongodb-service:27017")
	viper.SetDefault("MONGO_DB", "products_db")
	viper.SetDefault("REDIS_ADDR", "redis-service:6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("JWT_SECRET", "local-dev-secret-change-in-production")
	viper.SetDefault("USER_SERVICE_ADDR", "user-service:50051")

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
