package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"caspercloud/internal/queue"
	"caspercloud/internal/service"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Worker struct {
	queueClient *queue.Client
	instanceSvc *service.InstanceService
}

func New(queueClient *queue.Client, instanceSvc *service.InstanceService) *Worker {
	return &Worker{queueClient: queueClient, instanceSvc: instanceSvc}
}

// Run consumes messages until ctx is cancelled, then stops the consumer and finishes any in-flight delivery.
func (w *Worker) Run(ctx context.Context) error {
	if err := w.queueClient.SetPrefetch(1); err != nil {
		return fmt.Errorf("qos: %w", err)
	}
	deliveries, consumerTag, err := w.queueClient.Consume()
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			if err := w.queueClient.CancelConsume(consumerTag); err != nil {
				log.Printf("caspercloud worker: cancel consume: %v", err)
			}
			select {
			case d := <-deliveries:
				w.processDelivery(ctx, d)
			default:
			}
			log.Println("caspercloud worker: shutdown after draining in-flight delivery")
			return ctx.Err()

		case d, ok := <-deliveries:
			if !ok {
				return nil
			}
			w.processDelivery(ctx, d)
		}
	}
}

func (w *Worker) processDelivery(parent context.Context, d amqp.Delivery) {
	workCtx := context.WithoutCancel(parent)
	err := w.safeHandleMessage(workCtx, d.Body)
	w.finalizeDelivery(d, err)
}

func (w *Worker) safeHandleMessage(ctx context.Context, body []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return w.handleMessageBody(ctx, body)
}

func (w *Worker) finalizeDelivery(d amqp.Delivery, err error) {
	if err == nil {
		if ackErr := d.Ack(false); ackErr != nil {
			log.Printf("caspercloud worker: ack failed: %v", ackErr)
		}
		return
	}

	log.Printf("caspercloud worker: task failed: %v", err)

	prev := queue.RetryCountFromHeaders(d.Headers)
	if prev >= queue.MaxConsecutiveFailures-1 {
		hdr := amqp.Table{}
		if d.Headers != nil {
			for k, v := range d.Headers {
				hdr[k] = v
			}
		}
		hdr["x-caspercloud-dlq-reason"] = err.Error()
		if pubErr := w.queueClient.PublishToDLQ(context.Background(), d.Body, hdr); pubErr != nil {
			log.Printf("caspercloud worker: dlq publish failed: %v", pubErr)
			_ = d.Nack(false, true)
			return
		}
		if ackErr := d.Ack(false); ackErr != nil {
			log.Printf("caspercloud worker: ack after dlq failed: %v", ackErr)
		}
		return
	}

	var payload queue.TaskMessage
	if uerr := json.Unmarshal(d.Body, &payload); uerr != nil {
		log.Printf("caspercloud worker: republish skipped, invalid json: %v", uerr)
		_ = d.Nack(false, false)
		return
	}

	next := int32(prev + 1)
	headers := amqp.Table{queue.RetryHeaderKey: next}
	if pubErr := w.queueClient.PublishTaskWithHeaders(context.Background(), payload, headers); pubErr != nil {
		log.Printf("caspercloud worker: republish failed: %v", pubErr)
		_ = d.Nack(false, true)
		return
	}
	if ackErr := d.Ack(false); ackErr != nil {
		log.Printf("caspercloud worker: ack after republish failed: %v", ackErr)
	}
}

func (w *Worker) handleMessageBody(ctx context.Context, body []byte) error {
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
	case service.TaskTypeInstanceCreate:
		return w.instanceSvc.HandleCreateTask(ctx, taskID, projectID, instanceID)
	case service.TaskTypeInstanceStart:
		return w.instanceSvc.HandleStartTask(ctx, taskID, projectID, instanceID)
	case service.TaskTypeInstanceStop:
		return w.instanceSvc.HandleStopTask(ctx, taskID, projectID, instanceID)
	case service.TaskTypeInstanceReboot:
		return w.instanceSvc.HandleRebootTask(ctx, taskID, projectID, instanceID)
	case service.TaskTypeInstanceDestroy:
		return w.instanceSvc.HandleDestroyTask(ctx, taskID, projectID, instanceID)
	default:
		log.Printf("caspercloud worker: unknown task type %q (ack)", payload.Type)
		return nil
	}
}

// RunStateSync periodically reconciles Postgres instance state with libvirt domain state.
func (w *Worker) RunStateSync(ctx context.Context) {
	t := time.NewTicker(20 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := w.instanceSvc.SyncHypervisorWithDB(ctx); err != nil {
				log.Printf("caspercloud worker: state sync: %v", err)
			}
		}
	}
}
