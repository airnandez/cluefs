OS   := $(shell uname -a | cut -f 1 -d ' ' | tr [:upper:] [:lower:])
ARCH := $(shell uname -m)
TAG  := $(shell git tag)

all: build release

build:
	@go build

release:
	@echo "Packaging cluefs ${TAG} for ${OS}"
	@tar -czf cluefs-${TAG}-${OS}-${ARCH}.tar.gz cluefs
