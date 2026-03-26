package queue

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const InstanceTasksQueue = "instance.tasks"

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
	_, err = ch.QueueDeclare(InstanceTasksQueue, true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	return &Client{conn: conn, channel: ch}, nil
}

func (c *Client) PublishTask(ctx context.Context, msg TaskMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.channel.PublishWithContext(ctx, "", InstanceTasksQueue, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
		DeliveryMode: amqp.Persistent,
	})
}

func (c *Client) Consume() (<-chan amqp.Delivery, error) {
	msgs, err := c.channel.Consume(InstanceTasksQueue, "", false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("consume failed: %w", err)
	}
	return msgs, nil
}

func (c *Client) Close() error {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
