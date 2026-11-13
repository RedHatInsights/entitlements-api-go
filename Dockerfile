# Manual Build: YYYY-MM-DD
# Use go-toolset as the builder image
# Once built, copys GO executable to a smaller image and runs it from there

FROM registry.access.redhat.com/ubi9/go-toolset:9.8-1793736700@sha256:c2bd0309bf78ae317024fdfc5602a31704e0dba9572f6d4b0bd4b681033dc965 as builder

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

# Install Go 1.27.3 to address CVE-2026-111943 and GO-2026-6809 (Until new ubi9 go-toolset image supports this go version)
RUN curl -LO https://go.dev/dl/go1.27.3.linux-amd64.tar.gz && \
    rm -rf /usr/local/go && \
    tar -C /usr/local -xzf go1.27.3.linux-amd64.tar.gz && \
    rm go1.27.3.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"

RUN go mod download
COPY . .

RUN make

# Using ubi9-minimal due to its smaller footprint
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.9-1794467051@sha256:baafce2d34783a28b4a472779f304e97e00f9d35c99553bc53b068b14344d9a9

WORKDIR /

# Copy GO executable file and need directories from the builder image
COPY --from=builder /go/src/app/entitlements-api-go ./entitlements-api-go
COPY --from=builder /go/src/app/bundle-sync ./bundle-sync
COPY apispec ./apispec
COPY bundles ./bundles

USER 1001

CMD ["/entitlements-api-go"]