# Makefile for squirssi, a proper IRC client.
# https://code.dopame.me/veonik/squirssi

SUBPACKAGES := colors

SQUIRCY3_ROOT ?= ../squircy3
PLUGINS := $(patsubst $(SQUIRCY3_ROOT)/plugins/%,%,$(wildcard $(SQUIRCY3_ROOT)/plugins/*))
SOURCES := $(wildcard *.go) $(wildcard cmd/*/*.go) $(wildcard $(patsubst %,%/*.go,$(SUBPACKAGES))) $(shell find vendor/ -type f -name '*.go' 2> /dev/null)

OUTPUT_BASE := out

PLUGIN_TARGETS := $(patsubst %,$(OUTPUT_BASE)/%.so,$(PLUGINS))
SQUIRSSI_TARGET := $(OUTPUT_BASE)/squirssi

RACE ?= -race
TEST_ARGS ?= -count 1

.PHONY: all build generate run squirssi plugins clean

all: build plugins

clean:
	rm -rf $(OUTPUT_BASE)

build: squirssi

squirssi: $(SQUIRSSI_TARGET)

plugins: $(PLUGIN_TARGETS)

run: build
	$(SQUIRSSI_TARGET) 2>> $(OUTPUT_BASE)/squirssi_errors.log

test:
	go test -tags netgo $(RACE) $(TEST_ARGS) ./...

$(OUTPUT_BASE)/plugins: $(OUTPUT_BASE)
	cp -r $(SQUIRCY3_ROOT)/plugins $(OUTPUT_BASE)/plugins

.SECONDEXPANSION:
$(PLUGIN_TARGETS): $(OUTPUT_BASE)/%.so: $$(wildcard plugins/%/*) $(OUTPUT_BASE)/plugins $(SOURCES)
	go build -tags netgo $(RACE) -o $@ -buildmode=plugin $(OUTPUT_BASE)/plugins/$*/*.go

$(SQUIRSSI_TARGET): $(SOURCES)
	go build -tags netgo $(RACE) -o $@ cmd/squirssi/*.go

$(OUTPUT_BASE):
	mkdir -p $(OUTPUT_BASE)

$(SOURCES): $(OUTPUT_BASE)
