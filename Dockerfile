# Build the manager binary
FROM golang:1.13 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM builder as test
COPY . .
RUN curl -L https://go.kubebuilder.io/dl/2.3.1/linux/amd64 | tar -xz -C /tmp/ && \
  mv /tmp/kubebuilder_2.3.1_linux_amd64 /usr/local/kubebuilder
RUN make test

FROM gcr.io/distroless/static:nonroot

ARG CREATED=unspecified
ARG GIT_COMMIT=unspecified
ARG CIRCLE_REPOSITORY_URL=unspecified
ARG PROJECT=unspecified
ARG VERSION=unspecified
ARG CIRCLE_BUILD_URL=unspecified

LABEL org.opencontainers.image.authors="Adarga" \
      org.opencontainers.image.created=${CREATED} \
      org.opencontainers.image.revision=${GIT_COMMIT} \
      org.opencontainers.image.source=${CIRCLE_REPOSITORY_URL} \
      org.opencontainers.image.title=${PROJECT} \
      org.opencontainers.image.version=${VERSION} \
      ai.adarga.circle-build-url=${CIRCLE_BUILD_URL}

WORKDIR /
COPY --from=builder /workspace/manager .
USER nonroot:nonroot

ENTRYPOINT ["/manager"]
