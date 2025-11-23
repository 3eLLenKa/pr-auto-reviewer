package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	App        App        `yaml:"app"`
	Database   Database   `yaml:"database"`
	PR         PR         `yaml:"pr"`
	Migrations Migrations `yaml:"migrations"`
}

type App struct {
	Port     string `env:"APP_PORT" yaml:"port" env-default:"8080"`
	LogLevel string `env:"LOG_LEVEL" yaml:"log_level" env-default:"info"`
}

type Database struct {
	Driver          string        `env:"DB_DRIVER" yaml:"driver" env-default:"postgres"`
	Host            string        `env:"POSTGRES_HOST" yaml:"host" env-default:"localhost"`
	Port            string        `env:"POSTGRES_PORT" yaml:"port" env-default:"5432"`
	User            string        `env:"POSTGRES_USER" yaml:"user"`
	Password        string        `env:"POSTGRES_PASSWORD" yaml:"password"`
	DBName          string        `env:"POSTGRES_DB" yaml:"dbname"`
	MaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS" yaml:"max_open_conns" env-default:"10"`
	MaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS" yaml:"max_idle_conns" env-default:"5"`
	ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" yaml:"conn_max_lifetime" env-default:"5m"`
	SSLMode         string        `env:"SSL_MODE" yaml:"ssl_mode" env-default:"disable"`
}

type PR struct {
	MaxReviewers     int  `env:"PR_MAX_REVIEWERS" yaml:"max_reviewers" env-default:"2"`
	AssignOnlyActive bool `env:"PR_ASSIGN_ONLY_ACTIVE" yaml:"assign_only_active_users" env-default:"true"`
}

type Migrations struct {
	Dir string `env:"MIGRATIONS_DIR" yaml:"dir" env-default:"/migrations"`
}

func MustLoad() *Config {
	cfg := &Config{}
	path := os.Getenv("CONFIG_PATH")

	if path == "" {
		panic("config path is empty")
	}

	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	if err := cleanenv.ReadEnv(cfg); err != nil {
		log.Fatalf("failed to read env variables: %v", err)
	}

	return cfg
}
