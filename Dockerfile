# Build the bridge binary
FROM golang:1.25 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o bridge ./bridge/

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/bridge .
USER 65532:65532

ENTRYPOINT ["/bridge"]
