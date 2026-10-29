# Manual Build: YYYY-MM-DD
# Use go-toolset as the builder image
# Once built, copys GO executable to a smaller image and runs it from there

FROM registry.access.redhat.com/hi/go:1.27.2-fips-builder@sha256:08949f405667fd34ee37cd65d3e628c3f18decc5150af68243cbbb9e5a41b33d as builder

WORKDIR /go/src/app

COPY go.mod go.sum ./

USER root

ENV PATH="/usr/local/go/bin:${PATH}"

RUN go mod download
COPY . .

RUN make

FROM registry.access.redhat.com/hi/core-runtime:2.43-openssl-fips-builder@sha256:595c02312c44c3e4aa820a28d09ddc1c1b07ca1a35fab4448e339674d05e414f

WORKDIR /

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

# Copy GO executable file and need directories from the builder image
COPY --from=builder /go/src/app/entitlements-api-go ./entitlements-api-go
COPY --from=builder /go/src/app/bundle-sync ./bundle-sync
COPY apispec ./apispec
COPY bundles ./bundles
COPY licenses /licenses

ENV GODEBUG=fips140=on

USER 1001

CMD ["/entitlements-api-go"]