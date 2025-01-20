SHELL := /bin/bash # Use bash syntax
SERVICE?=restic_wrapper


.PHONY: help
## help: prints this help message
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## run: run the program locally
run:
	go run cmd/main.go

build:
	go build -o ${SERVICE} -ldflags "-s -w" cmd/main.go

build-intel:
	GOOS=darwin GOARCH=amd64 go build -o ${SERVICE}_amd64 -ldflags "-s -w" cmd/main.go

build-arm:
	GOOS=darwin GOARCH=arm64 go build -o ${SERVICE}_arm64 -ldflags "-s -w" cmd/main.go