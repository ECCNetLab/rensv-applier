package main

import (
	"context"
	"encoding/json"
	"log"

	rensvv1 "github.com/ECCNetLab/rensv-controller/api/v1"
	"github.com/streadway/amqp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

// Rensv is defines of json to receive
type Rensv struct {
	DocumentRoot string `json:"documentRoot"`
	ServerName   string `json:"serverName"`
}

func main() {
	// MQ connect
	conn, err := amqp.Dial("amqp://hoge:****@example.com:5672/")
	failOnError(err)
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err)
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"rensv", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err)

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err)

	// creates the scheme
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(rensvv1.GroupVersion,
		&rensvv1.Rensv{},
		&rensvv1.RensvList{},
	)
	metav1.AddToGroupVersion(scheme, rensvv1.GroupVersion)
	// creates the config
	config, err := rest.InClusterConfig()
	failOnError(err)
	config.GroupVersion = &rensvv1.GroupVersion
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.NewCodecFactory(scheme)
	// creates the client
	client, err := rest.RESTClientFor(config)
	failOnError(err)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var body Rensv
			log.Printf("Received a message: %s", d.Body)
			json.Unmarshal(d.Body, &body)
			log.Printf("DocumentRoot: %s\n", body.DocumentRoot)
			log.Printf("ServerName: %s\n", body.ServerName)

			rensvConfig := &rensvv1.Rensv{
				ObjectMeta: metav1.ObjectMeta{
					Name: body.ServerName,
				},
				Spec: rensvv1.RensvSpec{
					DocumentRoot: body.DocumentRoot,
					ServerName:   body.ServerName,
				},
			}

			result := &rensvv1.Rensv{}
			err := client.
				Post().
				Namespace("default").
				Resource("rensvs").
				Body(rensvConfig).
				Do(context.Background()).
				Into(result)
			if err != nil {
				log.Printf("Error while creating object: %s\n", err)
				err = ch.Publish(
					"",     // exchange
					q.Name, // routing key
					false,  // mandatory
					false,  // immediate
					amqp.Publishing{
						ContentType: "text/plain",
						Body:        d.Body,
					},
				)
				log.Printf(" [x] Requeue %s", d.Body)
				if err != nil {
					log.Printf("Failed to publish a message: %s\n", err)
				}
			} else {
				log.Printf("object created: %v\n", result)
			}
		}
	}()

	log.Printf(" [*] Waiting for messages...")
	<-forever
}

func failOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
