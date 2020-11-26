package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/streadway/amqp"
)

// Amqp はMQとの通信に必要なデータの定義
type Amqp struct {
	Connection *amqp.Connection
	Channel    *amqp.Channel
	Queue      amqp.Queue
	Messages   <-chan amqp.Delivery
}

// Auth はRabbitMQの認証に必要な情報の定義
type Auth struct {
	Username    string
	Password    string
	Host        string
	Port        int
	VirtualHost string
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

	// Queueのbindを定義
	err = mq.Channel.QueueBind(
		mq.Queue.Name, // queue name
		mq.Queue.Name, // routing key
		"test",        // exchange
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		log.Println("Failed to bind a queue")
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

// GetArgsAndEnv は引数と環境変数から認証情報を取得する
func GetArgsAndEnv() *Auth {
	a := Auth{}
	flag.StringVar(&a.Username, "username", "guest", "RabbitMQ username")
	flag.StringVar(&a.Password, "password", "guest", "RabbitMQ password")
	flag.StringVar(&a.Host, "host", "rabbitmq", "RabbitMQ hostname")
	flag.IntVar(&a.Port, "port", 5672, "RabbitMQ port")
	flag.StringVar(&a.VirtualHost, "virtualhost", "/", "RabbitMQ virtualhost")
	flag.VisitAll(func(f *flag.Flag) {
		if s := os.Getenv(strings.ToUpper(f.Name)); s != "" {
			flag.Set(f.Name, s)
		}
	})
	flag.Parse()
	return &a
}

// URI はMQ接続用URIを生成する
func (a *Auth) URI() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d%s", a.Username, a.Password, a.Host, a.Port, a.VirtualHost)
}
