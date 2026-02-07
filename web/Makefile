# ADS Marketplace Mini App — frontend build & run
# Usage: make [target]

.PHONY: install dev build start lint clean deploy

NPM ?= npm
WEB_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

install:
	cd $(WEB_DIR) && $(NPM) install

dev:
	cd $(WEB_DIR) && $(NPM) run dev

build:
	cd $(WEB_DIR) && $(NPM) run build

start:
	cd $(WEB_DIR) && $(NPM) run start

lint:
	cd $(WEB_DIR) && $(NPM) run lint

clean:
	rm -rf $(WEB_DIR).next $(WEB_DIR)node_modules

# Deploy: build and optionally copy standalone to a server.
# Override DEST e.g. make deploy DEST=/var/www/ads-mrkt
deploy: build
	@echo "Build complete. Standalone output is in .next/standalone."
	@if [ -n "$(DEST)" ]; then \
		echo "Copying to $(DEST)..."; \
		mkdir -p $(DEST); \
		cp -r $(WEB_DIR).next/standalone/* $(DEST)/; \
		cp -r $(WEB_DIR).next/static $(DEST)/.next/; \
		cp -r $(WEB_DIR)public $(DEST)/ 2>/dev/null || true; \
		echo "Done."; \
	fi
