APP := jex
JSON ?= test.json
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
DIST_DIR ?= dist

.PHONY: build run run-demo portable clean

build:
	go build -o $(APP) .

run:
	go run . $(JSON)

run-demo:
	go run . test.json

portable:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(DIST_DIR)/$(APP)-$(GOOS)-$(GOARCH) .

clean:
	rm -f $(APP)
