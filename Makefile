APP := dist-lab
MAIN := ./cmd
DIST := dist

VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

VERSION_PKG := $(shell go list -f '{{.ImportPath}}' ./internal/version)

LDFLAGS := -s -w \
	-X $(VERSION_PKG).Version=$(VERSION) \
	-X $(VERSION_PKG).Commit=$(COMMIT) \
	-X $(VERSION_PKG).Date=$(DATE)

.PHONY: run tidy tidy-check fmt vet test check build smoke clean snapshot install-arch install-apt uninstall-arch uninstall-apt debug-version verify

run:
	go run cmd/main.go
tidy:
	go mod tidy

tidy-check:
	go mod tidy
	git diff --exit-code go.mod go.sum

fmt:
	test -z "$$(gofmt -l .)"

vet:
	go vet ./...

test:
	go test -race ./...

check: tidy-check fmt vet test

verify: check build smoke snapshot

build:
	mkdir -p $(DIST)
	go build -ldflags '$(LDFLAGS)' -o $(DIST)/$(APP) $(MAIN)

smoke: build
	$(DIST)/$(APP) --version

debug-version:
	@echo "APP=$(APP)"
	@echo "MAIN=$(MAIN)"
	@echo "DIST=$(DIST)"
	@echo "VERSION=$(VERSION)"
	@echo "COMMIT=$(COMMIT)"
	@echo "DATE=$(DATE)"
	@echo "VERSION_PKG=$(VERSION_PKG)"
	@echo "LDFLAGS=$(LDFLAGS)"

clean:
	rm -rf $(DIST)

snapshot:
	goreleaser release --snapshot --clean --skip=publish

UNAME_M := $(shell uname -m)

ifeq ($(UNAME_M),x86_64)
PKG_ARCH := amd64
else ifeq ($(UNAME_M),aarch64)
PKG_ARCH := arm64
else
$(error unsupported architecture: $(UNAME_M))
endif

install-arch: snapshot
	sudo pacman -U ./dist/*_$(PKG_ARCH).pkg.tar.zst

install-apt: snapshot
	sudo apt install ./dist/*_$(PKG_ARCH).deb

uninstall-arch:
	sudo pacman -R $(APP)

uninstall-apt:
	sudo apt remove $(APP)