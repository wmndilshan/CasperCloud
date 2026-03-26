package worker

import (
	"context"
	"encoding/json"
	"log"

	"caspercloud/internal/queue"
	"caspercloud/internal/service"
	"github.com/google/uuid"
)

type Worker struct {
	queueClient *queue.Client
	instanceSvc *service.InstanceService
}

func New(queueClient *queue.Client, instanceSvc *service.InstanceService) *Worker {
	return &Worker{queueClient: queueClient, instanceSvc: instanceSvc}
}

func (w *Worker) Run(ctx context.Context) error {
	msgs, err := w.queueClient.Consume()
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-msgs:
			if !ok {
				return nil
			}
			if err := w.handleMessage(ctx, msg.Body); err != nil {
				log.Printf("worker message failed: %v", err)
				_ = msg.Nack(false, false)
				continue
			}
			_ = msg.Ack(false)
		}
	}
}

func (w *Worker) handleMessage(ctx context.Context, body []byte) error {
	var payload queue.TaskMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return err
	}
	taskID, err := uuid.Parse(payload.TaskID)
	if err != nil {
		return err
	}
	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		return err
	}
	instanceID, err := uuid.Parse(payload.InstanceID)
	if err != nil {
		return err
	}

	switch payload.Type {
	case "instance.create":
		return w.instanceSvc.HandleCreateTask(ctx, taskID, projectID, instanceID)
	default:
		return nil
	}
}
