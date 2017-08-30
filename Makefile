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

## Setup of the project
setup: 
	@brew install graphviz
	@brew install dep
	@go get golang.org/x/tools/cmd/cover
	@make vendor-install

## Install dependencies of the project
vendor-install:
	@dep ensure -v

## Visualizing dependencies status of the project
vendor-status:
	@dep status

## Visualizing dependencies 
vendor-view:
	@dep status -dot | dot -T png | open -f -a /Applications/Preview.app

## Update dependencies of the project
vendor-update:
	@echo "READ https://github.com/golang/dep"
	@echo ">> $$ dep ensure -update"
	@echo ">> $$ dep ensure -add github.com/foo/bar"

## Runs the project unit tests
test:
	@go test -v -cover `go list | grep -v vendor`
	@go vet `go list | grep -v vendor`

test_race:
	@go test -cpu=2 -race -v `go list | grep -v vendor`
