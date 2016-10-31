.SILENT:

SHELL = /bin/bash

COLOR_RESET = \033[0m
COLOR_GREEN = \033[32m
COLOR_YELLOW = \033[33m
COLOR_RED = \033[31m

## Prints this help
help:
	printf "${COLOR_YELLOW}Galf\n------\n\n${COLOR_RESET}"
	awk '/^[a-zA-Z\-\_0-9\.%]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			printf "${COLOR_GREEN}$$ make %s${COLOR_RESET} %s\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST) | sort
	printf "\n"

## Install dependencies of the project
setup:
	@go get golang.org/x/tools/cmd/cover
	@go get gopkg.in/check.v1
	@go get -v ./...

## Runs the project unit tests
test:
	@go test -v -cover `go list | grep -v vendor`
	@go vet `go list | grep -v vendor`
