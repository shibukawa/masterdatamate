SHELL := /bin/sh

APP_NAME := masterdatamate
DESKTOP_NAME := MasterDataMate

HOST ?= 127.0.0.1
PORT ?= 8787
WORKSPACE ?= .

GO ?= go
NPM ?= npm

GOCACHE_DIR := $(CURDIR)/.cache/go-build
GOFLAGS ?= -mod=mod -ldflags="-s -w" -trimpath
GOCACHE_ENV := GOCACHE=$(GOCACHE_DIR)

FRONTEND_DIST := dist
SERVER_EMBED_DIST := cmd/masterdatamate/dist
WEB_BINARY := dist-native/$(APP_NAME)
DESKTOP_BINARY := build/bin/$(DESKTOP_NAME)
MAC_APP := build/$(DESKTOP_NAME).app

.PHONY: all install frontend backend desktop package package-mac test check run clean help

all: backend

install:
	$(NPM) ci

frontend:
	$(NPM) run build

$(SERVER_EMBED_DIST): frontend
	rm -rf $(SERVER_EMBED_DIST)
	cp -R $(FRONTEND_DIST) $(SERVER_EMBED_DIST)

backend: $(SERVER_EMBED_DIST)
	mkdir -p $(GOCACHE_DIR) dist-native
	$(GOCACHE_ENV) $(GO) build $(GOFLAGS) -o $(WEB_BINARY) ./cmd/masterdatamate

desktop: frontend
	mkdir -p $(GOCACHE_DIR) build/bin
	$(GOCACHE_ENV) $(GO) build $(GOFLAGS) -o $(DESKTOP_BINARY) .

package: package-mac

package-mac: desktop
	rm -rf $(MAC_APP)
	mkdir -p $(MAC_APP)/Contents/MacOS $(MAC_APP)/Contents/Resources
	cp $(DESKTOP_BINARY) $(MAC_APP)/Contents/MacOS/$(DESKTOP_NAME)
	cp build/darwin/Info.plist $(MAC_APP)/Contents/Info.plist

test:
	mkdir -p $(GOCACHE_DIR)
	$(GOCACHE_ENV) $(GO) test $(GOFLAGS) ./...

check: frontend test

run: backend
	$(WEB_BINARY) --workspace $(WORKSPACE) --host $(HOST) --port $(PORT)

clean:
	rm -rf $(FRONTEND_DIST) $(SERVER_EMBED_DIST) dist-native build/bin $(MAC_APP)

help:
	@echo "Targets:"
	@echo "  make              Build frontend and Go web backend"
	@echo "  make frontend     Build React/Vite frontend into dist/"
	@echo "  make backend      Build single-binary Go web server"
	@echo "  make desktop      Build Wails desktop binary"
	@echo "  make package-mac  Build macOS .app bundle"
	@echo "  make test         Run Go tests"
	@echo "  make check        Build frontend and run Go tests"
	@echo "  make run          Build and run web server"
	@echo "  make clean        Remove generated build outputs"
