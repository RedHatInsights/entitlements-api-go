# Manual Build: YYYY-MM-DD
# Use go-toolset as the builder image
# Once built, copys GO executable to a smaller image and runs it from there
FROM registry.redhat.io/ubi8/go-toolset as builder

WORKDIR /go/src/app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

USER 0

RUN go build -o entitlements-api-go main.go
RUN go build -o ./bundle-sync bundle_sync/main.go

# Using ubi8-minimal due to its smaller footprint
FROM registry.redhat.io/ubi8/ubi-minimal

WORKDIR /

# Copy GO executable file and need directories from the builder image
COPY --from=builder /go/src/app/entitlements-api-go ./entitlements-api-go
COPY --from=builder /go/src/app/bundle-sync ./bundle-sync
COPY resources ./resources
COPY apispec ./apispec
COPY bundles ./bundles
COPY test_data ./test_data

USER 1001

CMD ["/entitlements-api-go"]
