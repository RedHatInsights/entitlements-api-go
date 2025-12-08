# Manual Build: YYYY-MM-DD
# Use go-toolset as the builder image
# Once built, copys GO executable to a smaller image and runs it from there

FROM registry.access.redhat.com/ubi9/go-toolset:9.7-1764620329 as builder

LABEL name="entitlements-api-go" \
      summary="Red Hat Entitlements API Service" \
      description="Go-based API service that manages user entitlements for Red Hat products, acting as a proxy to backend services including subscriptions and compliance" \
      io.k8s.description="Go-based API service that manages user entitlements for Red Hat products, acting as a proxy to backend services including subscriptions and compliance" \
      io.k8s.display-name="Red Hat Entitlements API" \
      io.openshift.tags="insights,entitlements,api,subscriptions,compliance" \
      com.redhat.component="entitlements-api-go" \
      version="1.0" \
      release="1" \
      vendor="Red Hat, Inc." \
      url="https://github.com/RedHatInsights/entitlements-go-api" \
      distribution-scope="private" \
      maintainer="platform-accessmanagement@redhat.com"

WORKDIR /go/src/app

COPY go.mod go.sum ./

USER root

ENV GOTOOLCHAIN=auto
RUN go mod download
COPY . .

RUN make

# Using ubi9-minimal due to its smaller footprint
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7-1764794109

WORKDIR /

# Copy GO executable file and need directories from the builder image
COPY --from=builder /go/src/app/entitlements-api-go ./entitlements-api-go
COPY --from=builder /go/src/app/bundle-sync ./bundle-sync
COPY apispec ./apispec
COPY bundles ./bundles

USER 1001

CMD ["/entitlements-api-go"]