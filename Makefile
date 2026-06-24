GO ?= go
HB_BIN ?= $(HOME)/.local/bin/hb
MARKDOWNLINT ?= markdownlint-cli2

.PHONY: all fmt test vet build check lint smoke clean

all: check build

fmt:
	gofmt -w cmd internal

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

build:
	$(GO) build -o $(HB_BIN) ./cmd/hb

check: fmt test vet

lint:
	$(MARKDOWNLINT) README.md AGENTS.md .agents/skills/homebase-platform-ui/SKILL.md

smoke: build
	$(HB_BIN) help
	$(HB_BIN) install --group __none__ --yes --no-setup

clean:
	rm -f $(HB_BIN)
