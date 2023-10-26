gen_files = api/server.gen.go api/types.gen.go
GO = go

all: generate build
build:
	$(GO) build -o entitlements-api-go main.go
	$(GO) build -o ./bundle-sync bundle_sync/main.go
clean:
	find . -name "*.gen.go" | xargs rm
	$(GO) clean -cache
	rm entitlements-api-go

$(gen_files): apispec/api.spec.json
	$(GO) generate ./...

generate: $(gen_files)

image:
	podman build -t entitlements-api-go .
exe: all
	./entitlements-api-go
debug-run: generate
	ENT_DEBUG=1 \
	$(GO) run main.go
run: generate
	$(GO) run main.go
test: generate
	$(GO) test -v ./...
test-all: generate
	$(GO) test -v --race --coverprofile=coverage.txt --covermode=atomic ./...
bench: generate
	$(GO) test -bench=. ./...
