SHELL=/bin/bash

# all: compile

# compile:
# 	./scripts/buildtestlets.sh
# 	go install ./cmd/...

update_deps:
	go get -u github.com/tools/godep
	godep save ./testlet/...