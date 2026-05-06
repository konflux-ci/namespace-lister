# Build the manager binary
FROM registry.access.redhat.com/ubi9/go-toolset:1.25.9-1778054913@sha256:180d433d97773ac90384662ee0f54c3b474f6eeb7219e414a4ca323d1196bb13 AS builder
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

FROM registry.access.redhat.com/ubi9/ubi-micro@sha256:e0b6e93fe3800bf75a3e95aaf63bdfd020ea6dc30a92ca4bfa0021fa28cd671a
WORKDIR /
COPY --from=builder /tmp/server .
USER 65532:65532

ENTRYPOINT ["/server"]
