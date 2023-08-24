gen_files = api/server.gen.go api/types.gen.go

all: generate build
build:
	go build -o entitlements-api-go main.go
	go build -o ./bundle-sync bundle_sync/main.go
clean:
	find . -name "*.gen.go" | xargs rm
	go clean -cache
	rm entitlements-api-go

$(gen_files): apispec/api.spec.json
	go generate ./...

generate: $(gen_files)

image:
	podman build -t entitlements-api-go .
exe: all
	./entitlements-api-go
debug-run: generate
	ENT_DEBUG=1 \
	go run main.go
run: generate
	go run main.go
test: generate
	go test -v ./...
test-all: generate
	go test -v --race --coverprofile=coverage.txt --covermode=atomic ./...
bench: generate
	go test -bench=. ./...
