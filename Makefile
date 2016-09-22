SHELL=/bin/bash

all: compile

compile:
	go install ./cmd/...

install:

	# Set testlet capabilities
	./scripts/set-testlet-capabilities.sh

update_deps:
	go get -u github.com/tools/godep
	godep save ./...
