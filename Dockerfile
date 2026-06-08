# Manual Build: YYYY-MM-DD
# Use go-toolset as the builder image
# Once built, copys GO executable to a smaller image and runs it from there

FROM registry.access.redhat.com/ubi9/go-toolset:9.8-1780490420 as builder

WORKDIR /go/src/app

COPY go.mod go.sum ./

USER root

# TODO: Remove once base image includes Go 1.26.3 for Go toolset
ENV GO_VERSION=1.26.3
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o /tmp/go.tar.gz && \
    rm -rf /usr/local/go && \
    tar -C /usr/local -xzf /tmp/go.tar.gz && \
    rm /tmp/go.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"

RUN go mod download
COPY . .

RUN make

# Using ubi9-minimal due to its smaller footprint
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8-1780378819

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

WORKDIR /

# TODO: Remove temp CVE patches once base image is updated
# CVE-2026-40356, CVE-2026-40355 (krb5-libs), CVE-2025-14087, CVE-2025-14512 (glib2), CVE-2026-4878 (libcap)
RUN microdnf update -y krb5-libs glib2 libcap && microdnf clean all

# Copy GO executable file and need directories from the builder image
COPY --from=builder /go/src/app/entitlements-api-go ./entitlements-api-go
COPY --from=builder /go/src/app/bundle-sync ./bundle-sync
COPY apispec ./apispec
COPY bundles ./bundles
COPY licenses /licenses

USER 1001

CMD ["/entitlements-api-go"]