package main

import (
	"context"
	"encoding/json"
	"log"

	rensvv1 "github.com/ECCNetLab/rensv-controller/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

// Rensv はMQとの通信で使用するjsonの定義
type Rensv struct {
	DocumentRoot string `json:"documentRoot"`
	ServerName   string `json:"serverName"`
}

func main() {
	// MQ接続
	queue, err := NewClient("amqp://hoge:****@example.com:5672/")
	FailOnError(err)
	defer queue.Close()

	// scheme設定
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(rensvv1.GroupVersion,
		&rensvv1.Rensv{},
		&rensvv1.RensvList{},
	)
	metav1.AddToGroupVersion(scheme, rensvv1.GroupVersion)
	// config作成
	config, err := rest.InClusterConfig()
	FailOnError(err)
	config.GroupVersion = &rensvv1.GroupVersion
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.NewCodecFactory(scheme)
	// client作成
	client, err := rest.RESTClientFor(config)
	FailOnError(err)

	forever := make(chan bool)

	go func() {
		for d := range queue.messages {
			// メッセージを受け取った時の処理
			var body Rensv
			log.Printf("Received a message: %s", d.Body)
			json.Unmarshal(d.Body, &body)
			log.Printf("DocumentRoot: %s\n", body.DocumentRoot)
			log.Printf("ServerName: %s\n", body.ServerName)

			// applyするデータ生成
			rensvConfig := &rensvv1.Rensv{
				ObjectMeta: metav1.ObjectMeta{
					Name: body.ServerName,
				},
				Spec: rensvv1.RensvSpec{
					DocumentRoot: body.DocumentRoot,
					ServerName:   body.ServerName,
				},
			}

			// applyする
			result := &rensvv1.Rensv{}
			err := client.
				Post().
				Namespace("default").
				Resource("rensvs").
				Body(rensvConfig).
				Do(context.Background()).
				Into(result)
			if err != nil {
				// apply失敗
				log.Printf("Error while creating object: %s\n", err)
				err = queue.Publish(d.Body)
				log.Printf(" [x] Requeue %s", d.Body)
				if err != nil {
					log.Printf("Failed to publish a message: %s\n", err)
				}
			} else {
				// apply成功
				log.Printf("object created: %v\n", result)
			}
		}
	}()

	log.Printf(" [*] Waiting for messages...")
	<-forever
}

// FailOnError はエラーが出た場合に出力して終了する
func FailOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
