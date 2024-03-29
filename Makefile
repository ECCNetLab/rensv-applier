# Image URL to use all building/pushing image targets
IMG ?= rensv-applyer:v1.1

# Build the app
build:
	CGO_ENABLED=0 GOOS=linux go build -o docker/app .

# Build the docker image
docker-build: build
	docker build -t ${IMG} docker/.

# Run against the configured Kubernetes cluster in ~/.kube/config
run: docker-build
	kubectl apply -f k8s/

# Delete the applyer
delete:
	kubectl delete -f k8s/
