# Multi-stage build for TechTracker
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /techtracker ./cmd/techtracker

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /techtracker /app/techtracker

# Copy required runtime files
COPY config.json /app/config.json
COPY template.md /app/template.md
COPY projects/ /app/projects/
COPY utils/ /app/utils/

RUN mkdir -p /app/data /app/data/history

ENTRYPOINT ["/app/techtracker"]
