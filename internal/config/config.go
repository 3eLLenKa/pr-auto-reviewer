package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	App        App        `yaml:"app"`
	Database   Database   `yaml:"database"`
	PR         PR         `yaml:"pr"`
	Migrations Migrations `yaml:"migrations"`
}

type App struct {
	Port     string `yaml:"port"`
	LogLevel string `yaml:"log_level"`
}

type Database struct {
	Driver          string        `yaml:"driver"`
	Host            string        `yaml:"host"`
	Port            string        `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	DBName          string        `yaml:"dbname"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	SSLMode         string        `env:"SSL_MODE" yaml:"ssl_mode" env-default:"disable"`
}

type PR struct {
	MaxReviewers     int  `yaml:"max_reviewers"`
	AssignOnlyActive bool `yaml:"assign_only_active_users"`
}

type Migrations struct {
	Dir string `yaml:"dir"`
}

func MustLoad() *Config {
	if _, err := os.Stat(".env"); err == nil {
		_ = godotenv.Load(".env")
	}

	cfg := &Config{}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "internal/config/config.yaml"
	}

	if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
		log.Fatalf("failed to read config file: %v", err)
	}

	if err := cleanenv.ReadEnv(cfg); err != nil {
		log.Printf("failed to read env overrides: %v", err)
	}

	return cfg
}
