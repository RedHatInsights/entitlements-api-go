# Manual Build: YYYY-MM-DD
# Use go-toolset as the builder image
# Once built, copys GO executable to a smaller image and runs it from there
# FROM registry.redhat.io/ubi8/go-toolset as builder
FROM quay.io/projectquay/golang:1.20 as builder

WORKDIR /go/src/app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

USER 0

RUN make

# Using ubi8-minimal due to its smaller footprint
FROM registry.access.redhat.com/ubi8/ubi-minimal

WORKDIR /

# Copy GO executable file and need directories from the builder image
COPY --from=builder /go/src/app/entitlements-api-go ./entitlements-api-go
COPY --from=builder /go/src/app/bundle-sync ./bundle-sync
COPY resources ./resources
COPY apispec ./apispec
COPY bundles ./bundles

USER 1001

CMD ["/entitlements-api-go"]
