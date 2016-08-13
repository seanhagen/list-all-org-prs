-include .env
export

BUILD_DIR ?= $(CURDIR)
CACHE_DIR ?= $(BUILD_DIR)/.tmp

GOVERSION = $(shell go version | sed -e 's/\ /-/g' | sed -e 's/\///')

ifeq ($(STACK),cedar-14)
export GOROOT := $(CACHE_DIR)/go/$(GOVERSION)
export PATH := $(GOROOT)/bin:$(PATH)
endif

SOURCEDIR=.
SOURCES := $(shell find $(SOURCEDIR) -type f -name '*.go')

BINARY=app

VERSION=1.0.0
BUILD_TIME=$(shell date +%FT%T%z)

REPO=github.com/seanhagen/list-all-org-prs
LDFLAGS=-ldflags "-X ${REPO}/core.Version=${VERSION} -X ${REPO}/core.BuildTime=${BUILD_TIME} -X ${REPO}/core.GoVersion=${GOVERSION}"

GODEPS=$(subst [,, $(subst ],, $(shell go list -f '{{.Deps}}')))
GODEPS+= \
	golang.org/x/tools/cmd/cover \
	github.com/mattn/goveralls \
	github.com/julienschmidt/httprouter \
	github.com/gorilla/context \
	github.com/justinas/alice \

define NL


endef

.DEFAULT_GOAL: $(BINARY)
.PHONY: clean generate test vet all install deps

$(BINARY): $(SOURCES)
	go build ${LDFLAGS} -o ${BINARY}

queries: db/queries.sql
	gotic -package app db/queries.sql > app/queries.go

clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi

generate:
	go generate

test:
	go test -coverprofile=out.cov -covermode=atomic -cover ./tests
	@if [ "$(COVERALLS_TOKEN)" != "" ]; then\
		goveralls -coverprofile=out.cov;\
	else \
		echo "not submitting to coveralls, COVERALLS_TOKEN not set"; \
	fi

vet:
	go vet

deps:
	$(foreach dep,$(GODEPS),go get $(dep)$(NL))

all: generate $(BINARY) test vet

database:
	go get bitbucket.org/liamstask/goose
	goose migrate up
