# Build the manager binary
FROM registry.access.redhat.com/ubi9/go-toolset@sha256:c9c0129bff8e94b9e23e9e4a1ebb5b932cccc24e064940a792bdc002da512d2a AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /namespace-lister

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
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
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -trimpath -a -o server .

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM registry.access.redhat.com/ubi9/ubi-micro@sha256:03692dc74d7a1870540bfe92ebded314cfd37b72d6174935c9aaf65d6b875269
WORKDIR /
COPY --from=builder /namespace-lister/server .
USER 65532:65532

ENTRYPOINT ["/server"]
