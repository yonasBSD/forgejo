ifeq ($(USE_REPO_TEST_DIR),1)

# This rule replaces the whole Makefile when we're trying to use /tmp repository temporary files
location = $(CURDIR)/$(word $(words $(MAKEFILE_LIST)),$(MAKEFILE_LIST))
self := $(location)

%:
	@tmpdir=`mktemp --tmpdir -d` ; \
	echo Using temporary directory $$tmpdir for test repositories ; \
	USE_REPO_TEST_DIR= $(MAKE) -f $(self) --no-print-directory REPO_TEST_DIR=$$tmpdir/ $@ ; \
	STATUS=$$? ; rm -r "$$tmpdir" ; exit $$STATUS

else

# This is the "normal" part of the Makefile

DIST := ../../dist
DIST_DIRS := $(DIST)/contrib $(DIST)/contrib-release

GO ?= go
HAS_GO = $(shell hash $(GO) > /dev/null 2>&1 && echo "GO" || echo "NOGO" )
COMMA := ,

XGO_VERSION := go-1.19.x
XGO_PACKAGE ?= src.techknowlogick.com/xgo@latest

ifeq ($(HAS_GO), GO)
	GOPATH ?= $(shell $(GO) env GOPATH)
	export PATH := $(GOPATH)/bin:$(PATH)

	CGO_CFLAGS ?= $(shell $(GO) env CGO_CFLAGS) $(CGO_EXTRA_CFLAGS)
endif

EXECUTABLEPREFIX := environment-to-ini

ifeq ($(shell sed --version 2>/dev/null | grep -q GNU && echo gnu),gnu)
	SED_INPLACE := sed -i
else
	SED_INPLACE := sed -i ''
endif

EXTRA_GOFLAGS ?=

MAKE_VERSION := $(shell "$(MAKE)" -v | head -n 1)

ifeq ($(RACE_ENABLED),true)
	GOFLAGS += -race
	GOTESTFLAGS += -race
endif

STORED_VERSION_FILE := VERSION

ifneq ($(DRONE_TAG),)
	VERSION ?= $(subst v,,$(DRONE_TAG))
	GITEA_VERSION ?= $(VERSION)
else
	ifneq ($(DRONE_BRANCH),)
		VERSION ?= $(shell echo $(DRONE_BRANCH) | sed -e 's|v\([0-9.][0-9.]*\)/.*|\1|')
	else
		VERSION ?= main
	endif

	STORED_VERSION=$(shell cat $(STORED_VERSION_FILE) 2>/dev/null)
	ifneq ($(STORED_VERSION),)
		GITEA_VERSION ?= $(STORED_VERSION)
	else
		GITEA_VERSION ?= $(shell git describe --tags --always | sed 's/-/+/' | sed 's/^v//')
	endif
endif

LDFLAGS := $(LDFLAGS) -s -w -X "main.MakeVersion=$(MAKE_VERSION)" -X "main.Version=$(GITEA_VERSION)" -X "main.Tags=$(TAGS)"

LINUX_ARCHS ?= linux/amd64,linux/386,linux/arm-5,linux/arm-6,linux/arm64

GO_PACKAGES ?= $(shell $(GO) list ./... | grep -v /vendor/)

TAGS ?=
TAGS_SPLIT := $(subst $(COMMA), ,$(TAGS))

GO_DIRS := .

GO_SOURCES := $(wildcard *.go)
GO_SOURCES += $(shell find $(GO_DIRS) -type f -name "*.go")

.PHONY: go-check
go-check:
	$(eval MIN_GO_VERSION_STR := $(shell grep -Eo '^go\s+[0-9]+\.[0-9]+' go.mod | cut -d' ' -f2))
	$(eval MIN_GO_VERSION := $(shell printf "%03d%03d" $(shell echo '$(MIN_GO_VERSION_STR)' | tr '.' ' ')))
	$(eval GO_VERSION := $(shell printf "%03d%03d" $(shell $(GO) version | grep -Eo '[0-9]+\.[0-9]+' | tr '.' ' ');))
	@if [ "$(GO_VERSION)" -lt "$(MIN_GO_VERSION)" ]; then \
		echo "Gitea requires Go $(MIN_GO_VERSION_STR) or greater to build. You can get it at https://go.dev/dl/"; \
		exit 1; \
	fi

.PHONY: clean
clean:
	$(GO) clean -i ./...
	rm -rf $(DIST)

.PHONY: generate
generate: $(TAGS_PREREQ) generate-go

.PHONY: generate-go
generate-go: $(TAGS_PREREQ)
	@echo "Running go generate..."
	@CC= GOOS= GOARCH= $(GO) generate -tags '$(TAGS)' $(GO_PACKAGES)

.PHONY: release
release: generate release-linux release-copy

$(DIST_DIRS):
	mkdir -p $(DIST_DIRS)

.PHONY: release-windows
release-windows: | $(DIST_DIRS)
	CGO_CFLAGS="$(CGO_CFLAGS)" $(GO) run $(XGO_PACKAGE) -go $(XGO_VERSION) -buildmode exe -dest $(DIST)/contrib -tags 'netgo osusergo $(TAGS)' -ldflags '-linkmode external -extldflags "-static" $(LDFLAGS)' -targets 'windows/*' -out $(EXECUTABLEPREFIX)-$(VERSION) .
ifeq ($(CI),true)
	cp /build/$(EXECUTABLEPREFIX)-* $(DIST)/contrib
endif

.PHONY: release-linux
release-linux: | $(DIST_DIRS)
	CGO_CFLAGS="$(CGO_CFLAGS)" $(GO) run $(XGO_PACKAGE) -go $(XGO_VERSION) -dest $(DIST)/contrib -tags 'netgo osusergo $(TAGS)' -ldflags '-linkmode external -extldflags "-static" $(LDFLAGS)' -targets '$(LINUX_ARCHS)' -out $(EXECUTABLEPREFIX)-$(VERSION) .
ifeq ($(CI),true)
	cp /build/$(EXECUTABLEPREFIX)-* $(DIST)/contrib
endif

.PHONY: release-darwin
release-darwin: | $(DIST_DIRS)
	CGO_CFLAGS="$(CGO_CFLAGS)" $(GO) run $(XGO_PACKAGE) -go $(XGO_VERSION) -dest $(DIST)/contrib -tags 'netgo osusergo $(TAGS)' -ldflags '$(LDFLAGS)' -targets 'darwin-10.12/amd64,darwin-10.12/arm64' -out $(EXECUTABLEPREFIX)-$(VERSION) .
ifeq ($(CI),true)
	cp /build/$(EXECUTABLEPREFIX)-* $(DIST)/contrib
endif

.PHONY: release-copy
release-copy: | $(DIST_DIRS)
	cd $(DIST); for file in `find ./contrib -type f -name "$(EXECUTABLEPREFIX)-*"`; do cp $${file} ./contrib-release/; done;

# This endif closes the if at the top of the file
endif
