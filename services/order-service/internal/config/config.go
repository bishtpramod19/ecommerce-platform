package config

import (
	"log"

	"github.com/spf13/viper"
)

// Config holds all configuration for the order service.
type Config struct {
	// Server
	ServerPort string `mapstructure:"SERVER_PORT"`
	GRPCPort   string `mapstructure:"GRPC_PORT"`

	// Database
	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     string `mapstructure:"DB_PORT"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBName     string `mapstructure:"DB_NAME"`

	// JWT (for validating incoming requests)
	JWTSecret string `mapstructure:"JWT_SECRET"`

	// Downstream service addresses (gRPC)
	UserServiceAddr      string `mapstructure:"USER_SERVICE_ADDR"`
	ProductServiceAddr   string `mapstructure:"PRODUCT_SERVICE_ADDR"`
	InventoryServiceAddr string `mapstructure:"INVENTORY_SERVICE_ADDR"`
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
	viper.SetDefault("SERVER_PORT", "8004")
	viper.SetDefault("GRPC_PORT", "50054")
	viper.SetDefault("DB_HOST", "postgres-service")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_PASSWORD", "postgres")
	viper.SetDefault("DB_NAME", "orders_db")
	viper.SetDefault("JWT_SECRET", "local-dev-secret-change-in-production")
	viper.SetDefault("USER_SERVICE_ADDR", "user-service:50051")
	viper.SetDefault("PRODUCT_SERVICE_ADDR", "product-service:50052")
	viper.SetDefault("INVENTORY_SERVICE_ADDR", "inventory-service:50053")

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
