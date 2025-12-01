APP_NAME ?= web3-backend
BIN_DIR ?= bin
BIN_PATH := $(BIN_DIR)/server
IMAGE_NAME ?= $(APP_NAME):latest
ENV_FILE ?= .env
PORT ?= 8080

.PHONY: build run tidy clean docker-build docker-run

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_PATH) ./cmd

run:
	ENV_FILE=$(ENV_FILE) go run ./cmd

clean:
	rm -rf $(BIN_DIR)

tidy:
	go mod tidy

docker-build:
	docker build -t $(IMAGE_NAME) .

docker-run: docker-build
	docker run --rm -p $(PORT):$(PORT) \
		--env-file=$(ENV_FILE) \
		$(IMAGE_NAME)
