all: generate build
build:
	go build -o entitlements-api-go main.go
	go build -o ./bundle-sync bundle_sync/main.go
clean:
	rm entitlements-api-go
	find . -name "*.gen.go" | xargs rm
	go clean -cache
generate:
	go generate ./...

image:
	podman build -t entitlements-api-go .
