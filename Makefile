OS      := $(shell uname -a | cut -f 1 -d ' ' | tr [:upper:] [:lower:])
ARCH    := $(shell uname -m)
TAG     := $(shell git describe master --tags)
TIMESTAMP := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

all: build

build:
	@go build -ldflags="-X main.buildTime=$(TIMESTAMP) -X main.version=$(TAG)"

release: build
	@echo "Packaging cluefs ${TAG} for ${OS}"
	@tar -czf cluefs-${TAG}-${OS}-${ARCH}.tar.gz cluefs

clean:
	@rm -f cluefs cluefs-*.tar.gz

buildall: clean build
