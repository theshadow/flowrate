version := $(shell git describe --tags --always --dirty --match='v*' 2> /dev/null || echo v0)

.PHONY: all build analysis
all: analysis build

analysis:
	go fmt .../..
	go vet .../..

build:
	go build \
		-ldflags "-X awsspy/cmd.Version=$(version)" \
		.../..