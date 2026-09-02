package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const Exchange = "savor.events"

type Conn struct {
	ch  *amqp.Channel
	log *slog.Logger
}

func Dial(url string, log *slog.Logger) (*Conn, error) {
	var last error
	for i := 0; i < 20; i++ {
		c, err := amqp.Dial(url)
		if err != nil {
			last = err
			time.Sleep(time.Second)
			continue
		}
		ch, err := c.Channel()
		if err != nil {
			last = err
			_ = c.Close()
			time.Sleep(time.Second)
			continue
		}
		if err := ch.ExchangeDeclare(Exchange, "topic", true, false, false, false, nil); err != nil {
			return nil, err
		}
		return &Conn{ch: ch, log: log}, nil
	}
	return nil, fmt.Errorf("rabbitmq: %w", last)
}

func (c *Conn) Publish(ctx context.Context, eventType string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.ch.PublishWithContext(ctx, Exchange, eventType, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Type:         eventType,
		Body:         body,
		Timestamp:    time.Now(),
	})
}

func (c *Conn) Consume(queue, key string, handler func([]byte) error) error {
	q, err := c.ch.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return err
	}
	if err := c.ch.QueueBind(q.Name, key, Exchange, false, nil); err != nil {
		return err
	}
	_ = c.ch.Qos(10, 0, false)
	msgs, err := c.ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for d := range msgs {
			if err := handler(d.Body); err != nil {
				c.log.Error("consume", "queue", queue, "err", err)
				_ = d.Nack(false, true)
				continue
			}
			_ = d.Ack(false)
		}
	}()
	return nil
}
