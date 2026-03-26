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
	"caspercloud/internal/libvirt"
	"caspercloud/internal/queue"
	"caspercloud/internal/repository"
	"caspercloud/internal/service"
	"caspercloud/internal/worker"
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
	libvirtAdapter := libvirt.NewVirshAdapter(cfg.LibvirtURI)
	instanceSvc := service.NewInstanceService(repo, queueClient, libvirtAdapter, cfg.VMDefaultRAM, cfg.VMDefaultVCPU)
	w := worker.New(queueClient, instanceSvc)

	log.Println("worker started")
	if err := w.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("worker failed: %v", err)
	}
	log.Println("worker stopped")
	_ = os.Stdout.Sync()
}
