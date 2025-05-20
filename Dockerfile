FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY src/go.mod src/go.sum ./

RUN go mod download
RUN go mod verify

COPY src .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /go-action main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /go-action /app/go-action

ENTRYPOINT ["/app/go-action"]
