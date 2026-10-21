package config

import (
	"backend/internal/adapters/secondary/mailer"
	"backend/internal/adapters/secondary/minio"
	"backend/internal/adapters/secondary/mysql"
	"backend/internal/adapters/secondary/postgres"
	"backend/internal/adapters/secondary/redis"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// "mega-erp-backend-svc/internal/adapters/secondary/postgres"
// "mega-erp-backend-svc/internal/adapters/secondary/redis"

// Config holds the application configuration
type Config struct {
	Server   ServerConfig
	Database mysql.Config
	Postgres postgres.Config
	Redis    redis.Config
	JWT      JWTConfig
	Minio    minio.Config
	Mailer   mailer.Config
}

// ServerConfig holds the HTTP server configuration
type ServerConfig struct {
	Port         string
	ReadTimeout  int
	WriteTimeout int
}

// JWTConfig holds the JWT configuration
type JWTConfig struct {
	Secret        string
	RefreshSecret string
}

var MinioGlobal minio.Config
var JWTGlobal JWTConfig

// LoadConfig loads the application configuration from environment variables and config files
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Set default values
	setDefaults()

	// Read from config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("❌ failed to read config file: %w", err)
		}
		// Config file not found, will use defaults and environment variables
	}

	// Read from environment variables
	viper.SetEnvPrefix("APP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("❌ failed to unmarshal config: %w", err)
	}
	MinioGlobal = config.Minio
	JWTGlobal = config.JWT
	return &config, nil
}

// setDefaults sets the default values for the configuration
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.readTimeout", 10)
	viper.SetDefault("server.writeTimeout", 10)

	// Database defaults
	viper.SetDefault("database.mysql.host", "localhost")
	viper.SetDefault("database.mysql.port", "3306")
	viper.SetDefault("database.mysql.user", "root")
	viper.SetDefault("database.mysql.password", "rootpassword")
	viper.SetDefault("database.mysql.dbname", "go-app")
	viper.SetDefault("database.mysql.sslmode", "disable")
	viper.SetDefault("database.mysql.timezone", "UTC")

	// Postgres defaults
	viper.SetDefault("database.postgres.host", "localhost")
	viper.SetDefault("database.postgres.port", "5432")
	viper.SetDefault("database.postgres.user", "root")
	viper.SetDefault("database.postgres.password", "root")
	viper.SetDefault("database.postgres.dbname", "local")
	viper.SetDefault("database.postgres.sslmode", "disable")
	viper.SetDefault("database.postgres.timezone", "UTC")

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", "6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// JWT defaults
	viper.SetDefault("jwt.secret", "your-secret-key")
	viper.SetDefault("jwt.refreshSecret", "your-refresh-secret-key")

	// Minio defaults
	viper.SetDefault("minio.endpoint", "localhost:2087")
	viper.SetDefault("minio.accessKey", "minio")
	viper.SetDefault("minio.secretKey", "minio")
	viper.SetDefault("minio.bucketName", "sabaix")
	viper.SetDefault("minio.baseUrl", "http://localhost:2087/sabaix")

	// Mailer defaults
	viper.SetDefault("mailer.host", "smtp.example.com")
	viper.SetDefault("mailer.username", "username")
	viper.SetDefault("mailer.password", "password")
	viper.SetDefault("mailer.port", 000)
	viper.SetDefault("mailer.from", "from")
}
