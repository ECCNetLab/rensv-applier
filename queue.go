package main

import (
	"log"

	"github.com/streadway/amqp"
)

// Amqp はMQとの通信に必要なデータの定義
type Amqp struct {
	Connection *amqp.Connection
	Channel    *amqp.Channel
	Queue      amqp.Queue
	Messages   <-chan amqp.Delivery
}

// NewClient はConsumerとしての接続までをする
func NewClient(url string) (mq Amqp, err error) {
	// MQへの接続
	mq.Connection, err = amqp.Dial(url)
	if err != nil {
		log.Println("Failed to connect to RabbitMQ")
		return mq, err
	}

	// チャンネル生成
	mq.Channel, err = mq.Connection.Channel()
	if err != nil {
		log.Println("Failed to open a channel")
		return mq, err
	}

	// Exchangeの定義
	err = mq.Channel.ExchangeDeclare(
		"test",   // name
		"direct", // type
		false,    // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		log.Println("Failed to declare an exchange")
		return mq, err
	}

	// Queueの定義
	mq.Queue, err = mq.Channel.QueueDeclare(
		"routing-key", // name
		false,         // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		log.Println("Failed to declare a queue")
		return mq, err
	}

	// 受信する際の定義
	mq.Messages, err = mq.Channel.Consume(
		mq.Queue.Name, // queue
		"",            // consumer
		true,          // auto-ack
		false,         // exclusive
		false,         // no-local
		false,         // no-wait
		nil,           // args
	)
	if err != nil {
		log.Println("Failed to register a consumer")
		return mq, err
	}

	return mq, nil
}

// Publish は引数のデータをqueueに投げる
func (a Amqp) Publish(msg []byte) error {
	err := a.Channel.Publish(
		"test",       // exchange
		a.Queue.Name, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        msg,
		},
	)
	return err
}

// Close はConnectionとChannelを閉じます
func (a *Amqp) Close() {
	a.Connection.Close()
	a.Channel.Close()
}
