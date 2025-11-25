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
	Port     string `yaml:"port" env:"APP_PORT"`
	LogLevel string `yaml:"log_level" env:"APP_LOG_LEVEL" env-default:"debug"`
}

type Database struct {
	Driver          string        `yaml:"driver" env:"POSTGRES_DRIVER" env-default:"postgres"`
	Host            string        `yaml:"host" env:"POSTGRES_HOST"`
	Port            string        `yaml:"port" env:"POSTGRES_PORT"`
	User            string        `yaml:"user" env:"POSTGRES_USER"`
	Password        string        `yaml:"password" env:"POSTGRES_PASSWORD"`
	DBName          string        `yaml:"dbname" env:"POSTGRES_DB"`
	MaxOpenConns    int           `yaml:"max_open_conns" env:"DB_MAX_OPEN_CONNS" env-default:"10"`
	MaxIdleConns    int           `yaml:"max_idle_conns" env:"DB_MAX_IDLE_CONNS" env-default:"5"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env:"DB_CONN_MAX_LIFETIME" env-default:"30m"`
	SSLMode         string        `yaml:"ssl_mode" env:"SSL_MODE" env-default:"disable"`
}

type PR struct {
	MaxReviewers     int  `yaml:"max_reviewers" env:"PR_MAX_REVIEWERS" env-default:"2"`
	AssignOnlyActive bool `yaml:"assign_only_active_users" env:"PR_ASSIGN_ONLY_ACTIVE" env-default:"true"`
}

type Migrations struct {
	Dir string `yaml:"dir" env:"MIGRATIONS_DIR"`
}

func MustLoad() *Config {
	if _, err := os.Stat(".env"); err == nil {
		_ = godotenv.Load(".env")
	}

	cfg := &Config{}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.yaml"
	}

	if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
		log.Fatalf("failed to read config file: %v", err)
	}

	if err := cleanenv.ReadEnv(cfg); err != nil {
		log.Printf("failed to read env overrides: %v", err)
	}

	return cfg
}
