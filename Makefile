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
run: generate
	ENT_DEBUG=1 \
	ENT_CA_PATH=$(PWD)/resources/ca.crt \
	ENT_KEY=$(PWD)/test_data/test.key \
	ENT_CERT=$(PWD)/test_data/test.cert \
	go run main.go
test: generate
	ENT_CA_PATH=$(PWD)/resources/ca.crt \
	ENT_KEY=$(PWD)/test_data/test.key \
	ENT_CERT=$(PWD)/test_data/test.cert \
	go test -v ./...
ginkgo: generate
	ENT_CA_PATH=$(PWD)/resources/ca.crt \
	ENT_KEY=$(PWD)/test_data/test.key \
	ENT_CERT=$(PWD)/test_data/test.cert \
	ginkgo --race --coverprofile=coverage.out --covermode=atomic ./...
bench:
	ENT_CA_PATH=$(PWD)/resources/ca.crt \
	ENT_KEY=$(PWD)/test_data/test.key \
	ENT_CERT=$(PWD)/test_data/test.cert \
	go test -bench=. ./...
