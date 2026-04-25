package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"caspercloud/internal/auth"
	"caspercloud/internal/config"
	"caspercloud/internal/db"
	"caspercloud/internal/instancemetrics"
	"caspercloud/internal/libvirt"
	"caspercloud/internal/queue"
	"caspercloud/internal/repository"
	"caspercloud/internal/service"
	"caspercloud/internal/worker"

	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	queueClient, err := queue.New(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("queue: %v", err)
	}
	defer queueClient.Close()

	repo := repository.New(pool)
	_ = auth.NewJWTManager(cfg.JWTSecret)
	libvirtAdapter := libvirt.NewLibvirtAdapter(cfg)
	volSvc := service.NewVolumeService(repo, libvirtAdapter, cfg.QEMUImgPath, cfg.VMVolumesDir)
	instanceSvc := service.NewInstanceService(repo, queueClient, libvirtAdapter, cfg.VMDefaultRAM, cfg.VMDefaultVCPU, cfg.LibvirtBridge, service.WithVolumes(volSvc))
	w := worker.New(queueClient, instanceSvc)

	memStore := instancemetrics.NewMemoryStore()
	var rdb *redis.Client
	if cfg.RedisURL != "" {
		opt, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			log.Fatalf("redis url: %v", err)
		}
		rdb = redis.NewClient(opt)
		defer func() { _ = rdb.Close() }()
	}
	go worker.RunMetricsPoller(ctx, repo, libvirtAdapter, memStore, rdb, cfg.MetricsPollInterval())
	go worker.RunPrometheusMetricsServer(ctx, cfg.WorkerMetricsAddr, memStore)

	startStaleTaskSweeper(ctx, repo)
	go w.RunStateSync(ctx)

	log.Println("worker started")
	if err := w.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("worker failed: %v", err)
	}
	log.Println("worker stopped")
	_ = os.Stdout.Sync()
}
