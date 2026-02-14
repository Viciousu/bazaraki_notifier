# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o bazaraki_notifier .

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bazaraki_notifier .

RUN mkdir -p /app/data

ENV DATA_FOLDER=/app/data
ENV CHECKING_INTERVAL=300

VOLUME ["/app/data"]

ENTRYPOINT ["./bazaraki_notifier"]
