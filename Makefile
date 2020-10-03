# Makefile for squirssi, a proper IRC client.
# https://code.dopame.me/veonik/squirssi

SUBPACKAGES := colors widget

# To build squircy3 plugins for squirssi, it's necessary to compile them all
# together along with squirssi to avoid dependency mismatches during the plugin
# loading process.
# This Makefile copies the plugin sources to within this Go module so that the
# Go tool will happily compile them into standalone shared libraries.
SQUIRCY3_ROOT ?= ../squircy3
PLUGINS := $(patsubst $(SQUIRCY3_ROOT)/plugins/%,%,$(wildcard $(SQUIRCY3_ROOT)/plugins/*))

SOURCES := $(wildcard *.go) $(wildcard cmd/*/*.go) $(wildcard $(patsubst %,%/*.go,$(SUBPACKAGES))) $(shell find vendor/ -type f -name '*.go' 2> /dev/null)

OUTPUT_BASE := out

RACE      ?= -race
TEST_ARGS ?= -count 1

GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GOARM  ?= $(shell go env GOARM)
CC     ?= $(shell go env CC)

PLUGIN_TARGETS  := $(patsubst %,$(OUTPUT_BASE)/%.so,$(PLUGINS))
SQUIRSSI_TARGET := $(OUTPUT_BASE)/squirssi
SQUIRSSI_DIST   := $(OUTPUT_BASE)/squirssi_$(GOOS)_$(GOARCH)$(GOARM)
PLUGIN_DIST     := $(patsubst %,$(OUTPUT_BASE)/%_$(GOOS)_$(GOARCH)$(GOARM).so,$(PLUGINS))
DIST_TARGETS    := $(SQUIRSSI_DIST) $(PLUGIN_DIST)

SQUIRSSI_VERSION := $(if $(shell test -d .git && echo "1"),$(shell git describe --always --tags),SNAPSHOT)
SQUIRCY3_VERSION := $(if $(shell test -d $(SQUIRCY3_ROOT)/.git && echo "1"),$(shell cd $(SQUIRCY3_ROOT) && git describe --always --tags),SNAPSHOT)

.PHONY: all build generate run squirssi plugins clean

all: build plugins

clean:
	rm -rf $(OUTPUT_BASE)

build: squirssi

dist: $(DIST_TARGETS)

squirssi: $(SQUIRSSI_TARGET)

plugins: $(PLUGIN_TARGETS)

run: build
	$(SQUIRSSI_TARGET)

test:
	go test -tags netgo $(RACE) $(TEST_ARGS) ./...

$(OUTPUT_BASE)/plugins: $(OUTPUT_BASE)
	cp -r $(SQUIRCY3_ROOT)/plugins $(OUTPUT_BASE)/plugins

$(SQUIRSSI_TARGET): $(SOURCES)
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) CC=$(CC) CGO_ENABLED=1 \
		go build -tags netgo $(RACE) \
			-ldflags "-s -w -X code.dopame.me/veonik/squirssi.Version=$(SQUIRSSI_VERSION)-SNAPSHOT -X main.Squircy3Version=$(SQUIRCY3_VERSION)-SNAPSHOT" \
			-o $@ cmd/squirssi/*.go

$(SQUIRSSI_DIST): $(OUTPUT_BASE) $(SOURCES)
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) CC=$(CC) CGO_ENABLED=1 \
		go build -tags netgo \
			-ldflags "-s -w -X code.dopame.me/veonik/squirssi.Version=$(SQUIRSSI_VERSION) -X main.Squircy3Version=$(SQUIRCY3_VERSION)" \
			-o $@ cmd/squirssi/*.go

.SECONDEXPANSION:
$(PLUGIN_TARGETS): $(OUTPUT_BASE)/%.so: $$(wildcard plugins/%/*) $(OUTPUT_BASE)/plugins $(SOURCES)
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) CC=$(CC) CGO_ENABLED=1 \
 		go build -tags netgo $(RACE) -o $@ -buildmode=plugin $(OUTPUT_BASE)/plugins/$*/*.go

.SECONDEXPANSION:
$(PLUGIN_DIST): $(OUTPUT_BASE)/%_$(GOOS)_$(GOARCH)$(GOARM).so: $$(wildcard plugins/%/*) $(OUTPUT_BASE)/plugins $(SOURCES)
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) CC=$(CC) CGO_ENABLED=1 \
		go build -tags netgo -o $@ -buildmode=plugin $(OUTPUT_BASE)/plugins/$*/*.go

$(OUTPUT_BASE):
	mkdir -p $(OUTPUT_BASE)

$(SOURCES): $(OUTPUT_BASE)
