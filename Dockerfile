FROM golang:1.22-alpine AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -ldflags "-s -w" -o kryptlet .

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/kryptlet .
USER 65532:65532
ENTRYPOINT ["/kryptlet"]
