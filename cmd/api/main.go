// @title           CasperCloud API
// @version         1.0
// @description     HTTP API for CasperCloud (multi-tenant projects, images, instances). Authenticated routes expect `Authorization: Bearer <JWT>`. Project-scoped routes require a token whose `active_project_id` matches the path `projectID` (use login with `project_id` or POST /v1/auth/switch-project).
// @host            localhost:8080
// @BasePath        /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description       JWT from POST /v1/auth/login or /v1/auth/register. Prefix value with "Bearer " (e.g. `Bearer eyJhbG...`).

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "caspercloud/api" // swag generated docs

	"caspercloud/internal/auth"
	"caspercloud/internal/config"
	"caspercloud/internal/db"
	"caspercloud/internal/httpapi"
	"caspercloud/internal/libvirt"
	"caspercloud/internal/queue"
	"caspercloud/internal/repository"
	"caspercloud/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
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
	jwtManager := auth.NewJWTManager(cfg.JWTSecret)
	libvirtAdapter := libvirt.NewLibvirtAdapter(cfg)

	authSvc := service.NewAuthService(repo, jwtManager)
	projectSvc := service.NewProjectService(repo)
	imageSvc := service.NewImageService(repo)
	instanceSvc := service.NewInstanceService(repo, queueClient, libvirtAdapter, cfg.VMDefaultRAM, cfg.VMDefaultVCPU)

	server := httpapi.NewServer(authSvc, projectSvc, imageSvc, instanceSvc, jwtManager)
	httpServer := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      server.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("api listening on %s", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
