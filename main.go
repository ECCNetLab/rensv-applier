package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"

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
	FailedCount  int    `json:"failedCount"`
}

// Auth はRabbitMQの認証に必要な情報の定義
type Auth struct {
	Username    string
	Password    string
	Host        string
	Port        int
	VirtualHost string
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

func main() {
	// MQ接続
	auth := GetArgsAndEnv()
	queue, err := NewClient(auth.URI())
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
		for d := range queue.Messages {
			// メッセージを受け取った時の処理
			var body Rensv
			log.Printf("Received a message: %s", d.Body)
			json.Unmarshal(d.Body, &body)
			// log.Printf("DocumentRoot: %s\n", body.DocumentRoot)
			// log.Printf("ServerName: %s\n", body.ServerName)
			// log.Printf("FailedCount: %d\n", body.FailedCount)

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
				// 5回以上失敗した場合、破棄する
				if body.FailedCount >= 5 {
					log.Printf("Failed to apply: %s\n", d.Body)
					continue
				}
				// 失敗回数をカウントアップ
				body.FailedCount++
				data, _ := json.Marshal(body)
				// goroutineで待機後、republishする
				go func() {
					s := math.Pow(2, float64(body.FailedCount))
					log.Printf(" [x]  After %.0f seconds, republish: %s\n", s, d.Body)
					time.Sleep(time.Duration(s) * time.Second)
					err = queue.Publish(data)
					if err != nil {
						log.Printf("Failed to republish a message: %s\n", err)
					}
				}()
			} else {
				// apply成功
				log.Printf("Object created: %v\n", result)
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
