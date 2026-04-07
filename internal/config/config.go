package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config — конфигурация сервиса. Читается из YAML-файла.
type Config struct {
	GRPCAddr string `yaml:"grpc_addr" env:"GRPC_ADDR" env-default:"0.0.0.0:50051"`
	LogLevel string `yaml:"log_level" env:"LOG_LEVEL" env-default:"info"`
	AuditLog string `yaml:"audit_log" env:"AUDIT_LOG" env-default:"audit.jsonl"`
}

// Load читает конфиг из YAML-файла по указанному пути.
func Load(path string) (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, fmt.Errorf("ошибка чтения конфига %q: %w", path, err)
	}
	return &cfg, nil
}
