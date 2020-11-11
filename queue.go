package main

import "github.com/streadway/amqp"

// Amqp はMQとの通信に必要なデータの定義
type Amqp struct {
	channel  *amqp.Channel
	queue    amqp.Queue
	messages <-chan amqp.Delivery
}

// NewClient はConsumerとしての接続までをする
func NewClient(url string) (mq Amqp, err error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return mq, err
	}

	mq.channel, err = conn.Channel()
	if err != nil {
		return mq, err
	}

	mq.queue, err = mq.channel.QueueDeclare(
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

	mq.messages, err = mq.channel.Consume(
		mq.queue.Name, // queue
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
	a.channel.Publish(
		"",           // exchange
		a.queue.Name, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        msg,
		},
	)
	return nil
}
