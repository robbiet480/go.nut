GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin

deps:
	@go mod tidy && go mod vendor && go mod verify
