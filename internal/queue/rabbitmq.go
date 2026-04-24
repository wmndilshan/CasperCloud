package queue

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	InstanceTasksQueue = "instance.tasks"
	TasksDLQQueue      = "tasks.dlq"
	// RetryHeaderKey counts prior failures; after 3 failed attempts the message is sent to the DLQ.
	RetryHeaderKey = "x-caspercloud-retry"
)

// MaxConsecutiveFailures is the number of failed processing attempts before routing to the DLQ.
const MaxConsecutiveFailures = 3

type Client struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

type TaskMessage struct {
	TaskID     string `json:"task_id"`
	Type       string `json:"type"`
	ProjectID  string `json:"project_id"`
	InstanceID string `json:"instance_id"`
}

func New(url string) (*Client, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if _, err = ch.QueueDeclare(InstanceTasksQueue, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	if _, err = ch.QueueDeclare(TasksDLQQueue, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	return &Client{conn: conn, channel: ch}, nil
}

func (c *Client) SetPrefetch(prefetchCount int) error {
	return c.channel.Qos(prefetchCount, 0, false)
}

func (c *Client) PublishTask(ctx context.Context, msg TaskMessage) error {
	return c.PublishTaskWithHeaders(ctx, msg, nil)
}

func (c *Client) PublishTaskWithHeaders(ctx context.Context, msg TaskMessage, headers amqp.Table) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	pub := amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
		Headers:      headers,
	}
	return c.channel.PublishWithContext(ctx, "", InstanceTasksQueue, false, false, pub)
}

func (c *Client) PublishToDLQ(ctx context.Context, body []byte, headers amqp.Table) error {
	pub := amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
		Headers:      headers,
	}
	return c.channel.PublishWithContext(ctx, "", TasksDLQQueue, false, false, pub)
}

// Consume starts a consumer with manual ack and returns the delivery channel and consumer tag (for Cancel).
func (c *Client) Consume() (<-chan amqp.Delivery, string, error) {
	tag := "caspercloud-worker"
	msgs, err := c.channel.Consume(InstanceTasksQueue, tag, false, false, false, false, nil)
	if err != nil {
		return nil, "", fmt.Errorf("consume failed: %w", err)
	}
	return msgs, tag, nil
}

func (c *Client) CancelConsume(consumerTag string) error {
	return c.channel.Cancel(consumerTag, false)
}

var _ TaskPublisher = (*Client)(nil)

func (c *Client) Close() error {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// RetryCountFromHeaders returns the number of prior failures encoded on the message (0 if absent).
func RetryCountFromHeaders(h amqp.Table) int {
	if h == nil {
		return 0
	}
	v, ok := h[RetryHeaderKey]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case uint32:
		return int(n)
	case uint64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}
