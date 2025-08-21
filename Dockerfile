# Manual Build: YYYY-MM-DD
# Use go-toolset as the builder image
# Once built, copys GO executable to a smaller image and runs it from there

FROM registry.access.redhat.com/ubi9/go-toolset:9.6-1752083840 as builder

WORKDIR /go/src/app

COPY go.mod go.sum ./

USER root

RUN go mod download 
COPY . .

RUN make

# Using ubi9-minimal due to its smaller footprint
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.6-1755695350

WORKDIR /

# Copy GO executable file and need directories from the builder image
COPY --from=builder /go/src/app/entitlements-api-go ./entitlements-api-go
COPY --from=builder /go/src/app/bundle-sync ./bundle-sync
COPY resources ./resources
COPY apispec ./apispec
COPY bundles ./bundles

USER 1001

CMD ["/entitlements-api-go"]