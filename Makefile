# Makefile for squirssi, a proper IRC client.
# https://code.dopame.me/veonik/squirssi

SUBPACKAGES := colors

SQUIRCY3_ROOT := ../squircy3
PLUGINS := $(patsubst $(SQUIRCY3_ROOT)/plugins/%,%,$(wildcard $(SQUIRCY3_ROOT)/plugins/*))
SOURCES := $(wildcard cmd/*/*.go) $(wildcard $(patsubst %,%/*.go,$(SUBPACKAGES)))

OUTPUT_BASE := out

PLUGIN_TARGETS := $(patsubst %,$(OUTPUT_BASE)/%.so,$(PLUGINS))
SQUIRSSI_TARGET := $(OUTPUT_BASE)/squirssi

RACE ?= -race
TEST_ARGS ?= -count 1

TESTDATA_NODEMODS_TARGET := testdata/node_modules

.PHONY: all build generate run squirssi plugins clean

all: build

clean:
	rm -rf plugins/ && cp -r $(SQUIRCY3_ROOT)/plugins .
	rm -rf $(OUTPUT_BASE)

build: plugins squirssi

generate: $(OUTPUT_BASE)/.generated

squirssi: $(SQUIRSSI_TARGET)

plugins: $(PLUGIN_TARGETS)

run: build
	$(SQUIRSSI_TARGET) 2> squirssi_errors.log

test: $(TESTDATA_NODEMODS_TARGET)
	go test -tags netgo $(RACE) $(TEST_ARGS) ./...

$(TESTDATA_NODEMODS_TARGET):
	cd testdata && \
		yarn install

.SECONDEXPANSION:
$(PLUGIN_TARGETS): $(OUTPUT_BASE)/%.so: $$(wildcard plugins/%/*) $(SOURCES)
	go build -tags netgo $(RACE) -o $@ -buildmode=plugin plugins/$*/*.go

$(SQUIRSSI_TARGET): $(SOURCES)
	go build -tags netgo $(RACE) -o $@ cmd/squirssi/*.go

$(OUTPUT_BASE)/.generated: $(GENERATOR_SOURCES)
	go generate
	touch $@

$(OUTPUT_BASE):
	mkdir -p $(OUTPUT_BASE)

$(SOURCES): $(OUTPUT_BASE)

$(GENERATOR_SOURCES): $(OUTPUT_BASE)
