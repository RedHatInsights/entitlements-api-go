# Manual Build: YYYY-MM-DD
# Use go-toolset as the builder image
# Once built, copys GO executable to a smaller image and runs it from there

FROM registry.access.redhat.com/ubi9/go-toolset:9.8-1793107762@sha256:c8947b60e6568dcdc5f66d382e18fa1fb89039918f59783e45b8c90847579955 as builder

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

# Install Go 1.25.5 to address CVE-2025-61729 (Until new ubi9 minimal image supports this go version)
RUN curl -LO https://go.dev/dl/go1.25.5.linux-amd64.tar.gz && \
    rm -rf /usr/local/go && \
    tar -C /usr/local -xzf go1.25.5.linux-amd64.tar.gz && \
    rm go1.25.5.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"

RUN go mod download
COPY . .

RUN make

# Using ubi9-minimal due to its smaller footprint
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8-1793050346@sha256:6f3a78179e0b30b930a186e7f68a4af52e51bb82406c9653be10bddd624e40f5

WORKDIR /

# Copy GO executable file and need directories from the builder image
COPY --from=builder /go/src/app/entitlements-api-go ./entitlements-api-go
COPY --from=builder /go/src/app/bundle-sync ./bundle-sync
COPY apispec ./apispec
COPY bundles ./bundles

USER 1001

CMD ["/entitlements-api-go"]