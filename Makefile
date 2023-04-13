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
debug-run: generate
	ENT_DEBUG=1 \
	ENT_CA_PATH=$(PWD)/resources/ca.crt \
	ENT_KEY=$(PWD)/test_data/test.key \
	ENT_CERT=$(PWD)/test_data/test.cert \
	go run main.go
run: generate
	ENT_CA_PATH=$(PWD)/resources/ca.crt \
	ENT_KEY=$(PWD)/test_data/test.key \
	ENT_CERT=$(PWD)/test_data/test.cert \
	go run main.go
test: generate
	ENT_CA_PATH=$(PWD)/resources/ca.crt \
	ENT_KEY=$(PWD)/test_data/test.key \
	ENT_CERT=$(PWD)/test_data/test.cert \
	go test -v ./...
test-all: generate
	ENT_CA_PATH=$(PWD)/resources/ca.crt \
	ENT_KEY=$(PWD)/test_data/test.key \
	ENT_CERT=$(PWD)/test_data/test.cert \
	go test --race --coverprofile=coverage.out --covermode=atomic ./...
bench: generate
	ENT_CA_PATH=$(PWD)/resources/ca.crt \
	ENT_KEY=$(PWD)/test_data/test.key \
	ENT_CERT=$(PWD)/test_data/test.cert \
	go test -bench=. ./...
