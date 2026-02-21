package config

import (
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	JWT       JWTConfig
	Storage   StorageConfig
	Tracing   TracingConfig
	Judge0    Judge0Config
	Redis     RedisConfig
	AI        AIConfig
	CORS      CORSConfig      `mapstructure:"cors"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

type RateLimitConfig struct {
	MaxRequests   int `mapstructure:"max_requests"`
	WindowMinutes int `mapstructure:"window_minutes"`
}

type AIConfig struct {
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"`
	Model   string `mapstructure:"model"`
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
	Secret     string        `mapstructure:"secret"`
	ExpireTime time.Duration `mapstructure:"expire_hours"`
}

type StorageConfig struct {
	Type          string `mapstructure:"type"`
	LocalPath     string `mapstructure:"local_path"`
	MinioEndpoint string `mapstructure:"minio_endpoint"`
	MinioAccessID string `mapstructure:"minio_access_key"`
	MinioSecret   string `mapstructure:"minio_secret_key"`
	MinioBucket   string `mapstructure:"minio_bucket"`
	OSSEndpoint   string `mapstructure:"oss_endpoint"`
	OSSAccessKey  string `mapstructure:"oss_access_key"`
	OSSSecretKey  string `mapstructure:"oss_secret_key"`
	OSSBucket     string `mapstructure:"oss_bucket"`
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

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

func LoadConfig(path string) (*Config, error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.SetEnvPrefix("CODER_EDU")
	viper.AutomaticEnv()

	// Database
	viper.BindEnv("database.host", "DATABASE_HOST")
	viper.BindEnv("database.port", "DATABASE_PORT")
	viper.BindEnv("database.user", "DATABASE_USER")
	viper.BindEnv("database.password", "DATABASE_PASSWORD")
	viper.BindEnv("database.dbname", "DATABASE_NAME")

	// JWT
	viper.BindEnv("jwt.secret", "JWT_SECRET")

	// Redis
	viper.BindEnv("redis.host", "REDIS_HOST")
	viper.BindEnv("redis.port", "REDIS_PORT")
	viper.BindEnv("redis.password", "REDIS_PASSWORD")

	// Server
	viper.BindEnv("server.mode", "SERVER_MODE")

	// AI
	viper.BindEnv("ai.base_url", "AI_BASE_URL")
	viper.BindEnv("ai.api_key", "AI_API_KEY")
	viper.BindEnv("ai.model", "AI_MODEL")

	// Storage / OSS
	viper.BindEnv("storage.type", "STORAGE_TYPE")
	viper.BindEnv("storage.oss_endpoint", "OSS_ENDPOINT")
	viper.BindEnv("storage.oss_access_key", "OSS_ACCESS_KEY")
	viper.BindEnv("storage.oss_secret_key", "OSS_SECRET_KEY")
	viper.BindEnv("storage.oss_bucket", "OSS_BUCKET")
	viper.BindEnv("storage.minio_endpoint", "MINIO_ENDPOINT")
	viper.BindEnv("storage.minio_access_key", "MINIO_ACCESS_KEY")
	viper.BindEnv("storage.minio_secret_key", "MINIO_SECRET_KEY")
	viper.BindEnv("storage.minio_bucket", "MINIO_BUCKET")

	// Judge0
	viper.BindEnv("judge0.api_key", "JUDGE0_API_KEY")
	viper.BindEnv("judge0.url", "JUDGE0_URL")
	viper.BindEnv("judge0.host", "JUDGE0_HOST")

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
