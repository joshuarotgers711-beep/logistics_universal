package mq

import (
	"log"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

type MQ struct{ ch *amqp.Channel }

func Connect() (*MQ, error) {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@rabbitmq:5672/"
	}
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	log.Println("connected to rabbitmq")
	return &MQ{ch: ch}, nil
}

func (m *MQ) Publish(exchange, key string, body []byte) error {
	version := os.Getenv("SCHEMA_VERSION")
	if version == "" {
		version = "1"
	}
	return m.ch.Publish(exchange, key, false, false, amqp.Publishing{
		ContentType: "application/json",
		Headers:     amqp.Table{"x-schema-version": version},
		Body:        body,
	})
}
