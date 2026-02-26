APP := jex
JSON ?= test.json

.PHONY: build run run-demo clean

build:
	go build -o $(APP) .

run:
	go run . $(JSON)

run-demo:
	go run . test.json

clean:
	rm -f $(APP)
