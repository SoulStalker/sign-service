package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config — конфигурация сервиса. Читается из YAML-файла.
type Config struct {
	GRPCAddr    string `yaml:"grpc_addr"     env:"GRPC_ADDR"     env-default:"0.0.0.0:50051"`
	TLSCertFile string `yaml:"tls_cert_file" env:"TLS_CERT_FILE"  env-required:"true"`
	TLSKeyFile  string `yaml:"tls_key_file"  env:"TLS_KEY_FILE"   env-required:"true"`
	TLSCAFile   string `yaml:"tls_ca_file"   env:"TLS_CA_FILE"    env-required:"true"`
	LogLevel    string `yaml:"log_level"     env:"LOG_LEVEL"      env-default:"info"`
	AuditLog    string `yaml:"audit_log"     env:"AUDIT_LOG"      env-default:"audit.jsonl"`
}

// Load читает конфиг из YAML-файла по указанному пути.
func Load(path string) (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, fmt.Errorf("ошибка чтения конфига %q: %w", path, err)
	}
	return &cfg, nil
}
