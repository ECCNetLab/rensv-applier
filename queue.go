package main

import "github.com/streadway/amqp"

// Amqp はMQとの通信に必要なデータの定義
type Amqp struct {
	Connection *amqp.Connection
	Channel    *amqp.Channel
	Queue      amqp.Queue
	Messages   <-chan amqp.Delivery
}

// NewClient はConsumerとしての接続までをする
func NewClient(url string) (mq Amqp, err error) {
	mq.Connection, err = amqp.Dial(url)
	if err != nil {
		return mq, err
	}

	mq.Channel, err = mq.Connection.Channel()
	if err != nil {
		return mq, err
	}

	mq.Queue, err = mq.Channel.QueueDeclare(
		"rensv", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		return mq, err
	}

	mq.Messages, err = mq.Channel.Consume(
		mq.Queue.Name, // queue
		"",            // consumer
		true,          // auto-ack
		false,         // exclusive
		false,         // no-local
		false,         // no-wait
		nil,           // args
	)

	return mq, nil
}

// Publish は引数のデータをqueueに投げる
func (a Amqp) Publish(msg []byte) error {
	a.Channel.Publish(
		"",           // exchange
		a.Queue.Name, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        msg,
		},
	)
	return nil
}

// Close はConnectionとChannelを閉じます
func (a *Amqp) Close() {
	a.Connection.Close()
	a.Channel.Close()
}
