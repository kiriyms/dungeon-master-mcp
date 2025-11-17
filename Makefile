.PHONY: build run test push

push:
	@git push origin main

run: build
	@./bin/app

build:
	@go build -ldflags="-X main.commit=local" -o bin/app ./cmd