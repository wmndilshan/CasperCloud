package config

import (
	"fmt"
	"os"
)

type Config struct {
	HTTPAddr      string
	DatabaseURL   string
	RabbitMQURL   string
	JWTSecret     string
	LibvirtURI    string
	StorageDir    string
	VMDefaultRAM  int
	VMDefaultVCPU int
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:      getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		RabbitMQURL:   os.Getenv("RABBITMQ_URL"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		LibvirtURI:    getEnv("LIBVIRT_URI", "qemu:///system"),
		StorageDir:    getEnv("VM_STORAGE_DIR", "/var/lib/libvirt/images"),
		VMDefaultRAM:  getEnvAsInt("VM_DEFAULT_RAM_MB", 2048),
		VMDefaultVCPU: getEnvAsInt("VM_DEFAULT_VCPU", 2),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.RabbitMQURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func getEnvAsInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	var out int
	_, err := fmt.Sscanf(val, "%d", &out)
	if err != nil {
		return fallback
	}
	return out
}
