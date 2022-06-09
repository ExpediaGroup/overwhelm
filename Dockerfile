# Build the manager binary
FROM golang:1.17 as builder

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
COPY pkg/ pkg/
COPY controllers/ controllers/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -gcflags "all=-N -l" -a -o manager main.go

# ______________________________________________________________________________________________________________________
FROM golang:1.17 as delve
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# ______________________________________________________________________________________________________________________
# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot as prod
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]

# ______________________________________________________________________________________________________________________
FROM debian:buster as debug
WORKDIR /
COPY --from=delve /go/bin/dlv .
COPY --from=builder /workspace/manager .
USER 65532:65532
EXPOSE 40000
ENTRYPOINT ["/dlv", "--listen=:40000", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "./manager"]
