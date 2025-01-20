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

