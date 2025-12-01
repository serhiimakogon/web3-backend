# web3-backend

Simple REST API server written in Go.

## Configuration

1. Copy `.env.example` to `.env` and adjust variables as needed.
2. `make run` automatically loads `.env`. To use a different file, set `ENV_FILE=path/to/file make run`.

## Run locally

```
make run
```

## Build binary

```
make build
```

## Docker

```
make docker-build
make docker-run
```
