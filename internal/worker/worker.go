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
				log.Printf("caspercloud worker: message processing failed (nack, no requeue): %v", err)
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
		log.Printf("caspercloud worker: invalid json payload len=%d err=%v", len(body), err)
		return err
	}
	log.Printf("caspercloud worker: dequeue task type=%s task_id=%s project_id=%s instance_id=%s", payload.Type, payload.TaskID, payload.ProjectID, payload.InstanceID)
	taskID, err := uuid.Parse(payload.TaskID)
	if err != nil {
		log.Printf("caspercloud worker: bad task_id %q: %v", payload.TaskID, err)
		return err
	}
	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		log.Printf("caspercloud worker: bad project_id %q: %v", payload.ProjectID, err)
		return err
	}
	instanceID, err := uuid.Parse(payload.InstanceID)
	if err != nil {
		log.Printf("caspercloud worker: bad instance_id %q: %v", payload.InstanceID, err)
		return err
	}

	switch payload.Type {
	case "instance.create":
		return w.instanceSvc.HandleCreateTask(ctx, taskID, projectID, instanceID)
	default:
		log.Printf("caspercloud worker: unknown task type %q (ack)", payload.Type)
		return nil
	}
}
