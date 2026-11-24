# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /src

ARG VERSION=dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/punkmcp \
    ./cmd/punkmcp

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/punkmcp /punkmcp

EXPOSE 8080

ENV MCP_TRANSPORT=http \
    MCP_ADDR=:8080

ENTRYPOINT ["/punkmcp"]
