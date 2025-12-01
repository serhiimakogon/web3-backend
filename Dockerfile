# syntax=docker/dockerfile:1

FROM golang:1.24.3-bullseye AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /server ./cmd

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app

COPY --from=builder /server ./server

EXPOSE 8080
ENTRYPOINT ["./server"]
