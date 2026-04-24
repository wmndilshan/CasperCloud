package queue

import "context"

// TaskPublisher publishes instance task messages to the primary worker queue.
type TaskPublisher interface {
	PublishTask(ctx context.Context, msg TaskMessage) error
}
