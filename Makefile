.SILENT:

SHELL = /bin/bash
.DEFAULT_GOAL := help

SOURCE_FILES?=$$(go list ./... | grep -v /vendor/)

## Setup of the project
setup:
	@go get -u github.com/alecthomas/gometalinter
	@go get -u github.com/golang/dep/...
	@go get -u golang.org/x/tools/cmd/cover
	@make vendor-install
	gometalinter --install --update

## Install dependencies of the project
vendor-install:
	@dep ensure -v

## Visualizing dependencies status of the project
vendor-status:
	@dep status

## Visualizing dependencies 
vendor-view:
	@brew install graphviz
	@dep status -dot | dot -T png | open -f -a /Applications/Preview.app

## Update dependencies of the project
vendor-update:
	@echo "READ https://github.com/golang/dep"
	@echo ">> $$ dep ensure -update"
	@echo ">> $$ dep ensure -add github.com/foo/bar"

## Runs the project unit tests
test:
	@go test -v -covermode atomic -cover -coverprofile coverage.txt $(SOURCE_FILES)
	@go tool vet . 2>&1 | grep -v '^vendor\/' | grep -v '^exit\ status\ 1' || true

## Run all the tests and opens the coverage report
test-cover: test 
	go tool cover -html=coverage.txt

test-race:
	@go test -cpu=2 -race -v $(SOURCE_FILES)

## Run all the tests and code checks
test-ci: lint test 

lint: ## Run all the linters
	gometalinter --vendor --disable-all \
		--enable=deadcode \
		--enable=ineffassign \
		--enable=gosimple \
		--enable=staticcheck \
		--enable=gofmt \
		--enable=goimports \
		--enable=dupl \
		--enable=misspell \
		--enable=errcheck \
		--enable=vet \
		--enable=vetshadow \
		--deadline=10m \
		./...


COLOR_RESET = \033[0m
COLOR_COMMAND = \033[36m
COLOR_YELLOW = \033[33m

## Prints this help
help:
	printf "${COLOR_YELLOW}Galf\n------\n${COLOR_RESET}"
	awk '/^[a-zA-Z\-\_0-9\.%]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			printf "${COLOR_COMMAND}$$ make %s${COLOR_RESET} %s\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST) | sort
	printf "\n"
