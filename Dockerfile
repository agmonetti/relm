FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /relm-mcp ./cmd/relm-mcp

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /relm-mcp /usr/local/bin/relm-mcp
ENTRYPOINT ["relm-mcp"]
