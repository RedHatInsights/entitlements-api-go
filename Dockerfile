# Manual Build: YYYY-MM-DD
# Use go-toolset as the builder image
# Once built, copys GO executable to a smaller image and runs it from there

FROM registry.access.redhat.com/ubi8/go-toolset:1.22.9-1.1736925145 as builder

WORKDIR /go/src/app

COPY go.mod go.sum ./

USER root

RUN go mod download 
COPY . .

RUN make

# Using ubi8-minimal due to its smaller footprint
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.10-1179

WORKDIR /

# Copy GO executable file and need directories from the builder image
COPY --from=builder /go/src/app/entitlements-api-go ./entitlements-api-go
COPY --from=builder /go/src/app/bundle-sync ./bundle-sync
COPY resources ./resources
COPY apispec ./apispec
COPY bundles ./bundles

USER 1001

CMD ["/entitlements-api-go"]