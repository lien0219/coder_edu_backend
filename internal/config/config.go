package config

import (
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Storage  StorageConfig
	Tracing  TracingConfig
	Judge0   Judge0Config
}

type ServerConfig struct {
	Port string
	Mode string
}

type DatabaseConfig struct {
	Host      string
	Port      int
	User      string
	Password  string
	DBName    string
	Charset   string
	ParseTime bool
}

type JWTConfig struct {
	Secret     string
	ExpireTime time.Duration
}

type StorageConfig struct {
	Type          string
	LocalPath     string
	MinioEndpoint string
	MinioAccessID string
	MinioSecret   string
	MinioBucket   string
}
type TracingConfig struct {
	Enabled           bool
	CollectorEndpoint string
}

type Judge0Config struct {
	APIKey string `mapstructure:"api_key"`
	URL    string
	Host   string
}

func LoadConfig(path string) (*Config, error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.SetEnvPrefix("CODER_EDU")
	viper.AutomaticEnv()

	viper.BindEnv("database.host", "DATABASE_HOST")
	viper.BindEnv("database.port", "DATABASE_PORT")
	viper.BindEnv("database.user", "DATABASE_USER")
	viper.BindEnv("database.password", "DATABASE_PASSWORD")
	viper.BindEnv("database.dbname", "DATABASE_NAME")
	viper.BindEnv("jwt.secret", "JWT_SECRET")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	cfg.JWT.ExpireTime = cfg.JWT.ExpireTime * time.Hour

	if cfg.Storage.Type == "local" {
		if _, err := os.Stat(cfg.Storage.LocalPath); os.IsNotExist(err) {
			os.MkdirAll(cfg.Storage.LocalPath, os.ModePerm)
		}
	}

	return &cfg, nil
}
