# Image URL to use all building/pushing image targets
IMG ?= rensv-applyer:v1

# Build the app
build:
	GOOS=linux go build -o docker/app .

# Build the docker image
docker-build: build
	docker build -t ${IMG} docker/.

# Run against the configured Kubernetes cluster in ~/.kube/config
run: docker-build
	kubectl apply -f k8s/
