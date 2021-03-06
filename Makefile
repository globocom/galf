.SILENT:

SHELL = /bin/bash
.DEFAULT_GOAL := help
LAST_TAG := `git describe --tags`

SOURCE_FILES?=$$(go list ./... | grep -v /vendor/)

## Setup of the project
setup:
	@go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
	@go get -u github.com/golang/dep/...
	@go get -u golang.org/x/tools/cmd/cover
	@make vendor-install

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
	@go vet . 2>&1 | grep -v '^vendor\/' | grep -v '^exit\ status\ 1' || true

## Run all the tests and opens the coverage report
test-cover: test 
	go tool cover -html=coverage.txt

test-race:
	@go test -cpu=2 -race -v $(SOURCE_FILES)

## Run all the tests and code checks
test-ci: lint test 

lint: ## Run all the linters
	golangci-lint run

## Release of the project
release:
	@printf "\n"; \
	read -p "Tag ($(LAST_TAG)): "; \
	if [ ! "$$REPLY" ]; then \
		printf "\n${COLOR_RED}"; \
		echo "Invalid tag."; \
		exit 1; \
	fi; \
	TAG=$$REPLY; \
	git tag -s $$TAG -m "$$TAG"; \
	git push origin $$TAG


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
