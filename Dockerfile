# Build the manager binary
FROM registry.access.redhat.com/ubi9/go-toolset@sha256:799cc027d5ad58cdc156b65286eb6389993ec14c496cf748c09834b7251e78dc AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /namespace-lister

# Copy the Go Modules manifests
COPY go.mod go.sum ./
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY . .

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -trimpath -a -o /tmp/server .

FROM registry.access.redhat.com/ubi9/ubi-micro@sha256:093a704be0eaef9bb52d9bc0219c67ee9db13c2e797da400ddb5d5ae6849fa10
WORKDIR /
COPY --from=builder /tmp/server .
USER 65532:65532

ENTRYPOINT ["/server"]
