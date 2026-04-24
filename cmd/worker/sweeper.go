package main

import (
	"context"
	"log"
	"time"

	"caspercloud/internal/repository"
)

func startStaleTaskSweeper(ctx context.Context, repo *repository.Repository) {
	go func() {
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				n, err := repo.ReconcileStaleRunningTasks(context.Background())
				if err != nil {
					log.Printf("caspercloud sweeper: reconcile failed: %v", err)
					continue
				}
				if n > 0 {
					log.Printf("caspercloud sweeper: reconciled %d stale running task(s)", n)
				}
			}
		}
	}()
}
