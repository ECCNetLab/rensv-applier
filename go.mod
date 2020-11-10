module github.com/ECCNetLab/rensv-applyer

go 1.13

require (
	github.com/streadway/amqp v1.0.0
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	github.com/ECCNetLab/rensv-controller v1.0.0
)

replace (
	github.com/ECCNetLab/rensv-controller v1.0.0 => ./rensv-controller
)
