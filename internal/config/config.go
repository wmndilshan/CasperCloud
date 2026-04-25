package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	HTTPAddr         string
	DatabaseURL      string
	RabbitMQURL      string
	JWTSecret        string
	LibvirtURI       string
	StorageDir       string
	VMInstancesDir   string
	VMVolumesDir      string
	LibvirtNetwork   string
	LibvirtBridge    string
	InstanceNetworkCIDR     string
	InstanceNetworkGateway  string
	QEMUImgPath      string
	CloudLocalDSPath string
	GenisoimagePath  string
	VMDefaultRAM     int
	VMDefaultVCPU    int

	// RedisURL is optional; when set, the worker publishes instance metrics and the API serves GET .../stats from Redis.
	RedisURL string
	// WorkerMetricsAddr is the listen address for the worker Prometheus scrape endpoint (e.g. ":9090"). Empty disables it.
	WorkerMetricsAddr string
	// MetricsPollSeconds is how often the worker polls libvirt for running instances (default 12).
	MetricsPollSeconds int
	// IptablesPath is the iptables binary used for floating IP NAT (worker); default "iptables".
	IptablesPath string
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:         getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		RabbitMQURL:      os.Getenv("RABBITMQ_URL"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		LibvirtURI:       getEnv("LIBVIRT_URI", "qemu:///system"),
		StorageDir:       getEnv("VM_STORAGE_DIR", "/var/lib/libvirt/images"),
		VMInstancesDir:   getEnv("VM_INSTANCES_DIR", "/var/lib/caspercloud/instances"),
		VMVolumesDir:     getEnv("VM_VOLUMES_DIR", "/var/lib/caspercloud/volumes"),
		LibvirtNetwork:   getEnv("LIBVIRT_NETWORK", "default"),
		LibvirtBridge:    getEnv("LIBVIRT_BRIDGE", "virbr0"),
		InstanceNetworkCIDR:    getEnv("INSTANCE_NETWORK_CIDR", "192.168.122.0/24"),
		InstanceNetworkGateway: getEnv("INSTANCE_NETWORK_GATEWAY", "192.168.122.1"),
		QEMUImgPath:      getEnv("QEMU_IMG_PATH", "qemu-img"),
		CloudLocalDSPath: os.Getenv("CLOUD_LOCALDS_PATH"),
		GenisoimagePath:  os.Getenv("GENISOIMAGE_PATH"),
		VMDefaultRAM:     getEnvAsInt("VM_DEFAULT_RAM_MB", 2048),
		VMDefaultVCPU:    getEnvAsInt("VM_DEFAULT_VCPU", 2),
		RedisURL:         os.Getenv("REDIS_URL"),
		WorkerMetricsAddr: getEnv("WORKER_METRICS_ADDR", ""),
		MetricsPollSeconds: getEnvAsInt("METRICS_POLL_SECONDS", 12),
		IptablesPath:       getEnv("IPTABLES_PATH", "iptables"),
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

// MetricsPollInterval returns the worker libvirt polling interval.
func (c *Config) MetricsPollInterval() time.Duration {
	if c.MetricsPollSeconds <= 0 {
		return 12 * time.Second
	}
	return time.Duration(c.MetricsPollSeconds) * time.Second
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
