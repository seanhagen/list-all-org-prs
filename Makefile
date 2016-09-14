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

BINARY=lprs

VERSION=1.0.0
BUILD_TIME=$(shell date +%FT%T%z)

REPO=github.com/KloudKtrl/asset-service
LDFLAGS=-ldflags "-X ${REPO}/core.Version=${VERSION} -X ${REPO}/core.BuildTime=${BUILD_TIME} -X ${REPO}/core.GoVersion=${GOVERSION}"

GODEPS=$(sort \
	$(strip $(subst [,, $(subst ],, \
	$(shell go list -f '{{.Deps}}' ./...)\
	)))\
	)
GODEPS+= \
		golang.org/x/tools/cmd/cover \
		github.com/mattn/goveralls \
		github.com/wadey/gocovmerge \

define NL


endef

.DEFAULT_GOAL: $(BINARY)
.PHONY: clean generate test vet all install build

DIRS := $(shell go list ./... | tail -n+2 | sed -e "s@$(REPO)@\.@" )
install:
	@$(foreach d,$(DIRS), go install $(d) &&)echo "Done install"

deps:
	$(foreach dep,$(GODEPS),go get $(dep)$(NL))
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install

build: queries vet $(BINARY)

$(BINARY): $(SOURCES)
	go build ${LDFLAGS} -o ${BINARY}

queries: db/queries.sql
	gotic -package app db/queries.sql > app/queries.go

clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi

generate:
	go generate

SUBDIRS := $(shell find . -type d -not -name '.*' | grep -v git | grep -v db )
COV := $(shell find . -name "*.cov")
test:
	$(foreach c,$(COV), rm $(c))
	@$(foreach PKG,$(SUBDIRS), \
		echo Testing $(PKG); \
		go test -coverprofile=$(PKG).cov -covermode=atomic -cover ./$(PKG)$(NL))
	find . -name "*.cov" | xargs -t gocovmerge > out.cov
	@if [ "$(COVERALLS_TOKEN)" != "" ] && [ -s out.cov ]; then\
		goveralls -coverprofile=out.cov;\
	else \
		echo "not submitting to coveralls, COVERALLS_TOKEN not set or file empty"; \
	fi
	find . -name "*.cov" | xargs rm

vet:
	go vet

lint: install
	@if [ "$(COVERALLS_TOKEN)" != "" ]; then \
		gometalinter --disable-all --enable=vet --enable=gotype --enable=gocyclo --enable=golint --enable=deadcode --enable=varcheck --enable=structcheck --enable=dupl --enable=goconst --enable=gosimple --enable=unconvert --deadline=60s ./...; \
	else \
		gometalinter --deadline=10s ./...; \
	fi

all: generate lint $(BINARY) test vet

database:
	go get bitbucket.org/liamstask/goose/cmd/goose
	goose up
