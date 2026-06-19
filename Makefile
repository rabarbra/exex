BINARY := exex
# Strip the symbol table (-s) and DWARF (-w) from release builds.
LDFLAGS := -s -w
DIST := dist
VERSION ?= dev
# Platforms built by `make release`.
RELEASE_PLATFORMS ?= darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 linux/386 linux/arm

.PHONY: build lite install test clean release

# Full build: includes Chroma syntax highlighting (source pane + asm colours).
build:
	go build -trimpath -ldflags="$(LDFLAGS)" -o $(BINARY) .

# Lite build: drops Chroma (and its embedded lexer/style data), ~3.5 MB smaller.
# Syntax highlighting falls back to the built-in minimal highlighter.
lite:
	go build -tags lite -trimpath -ldflags="$(LDFLAGS)" -o $(BINARY) .

install:
	go install -trimpath -ldflags="$(LDFLAGS)" .

test:
	go test ./...
	go vet -tags lite ./...

clean:
	rm -f $(BINARY)
	rm -rf $(DIST)

# release cross-compiles full + lite archives for every RELEASE_PLATFORMS entry
# into $(DIST), plus a checksums file. Used by the GitHub release workflow:
#   make release VERSION=v1.2.3
release:
	rm -rf $(DIST) && mkdir -p $(DIST)
	@for platform in $(RELEASE_PLATFORMS); do \
	  os=$${platform%/*}; arch=$${platform#*/}; \
	  for variant in full lite; do \
	    tags=""; suffix=""; \
	    if [ "$$variant" = lite ]; then tags="-tags lite"; suffix="-lite"; fi; \
	    bin=$(BINARY); [ "$$os" = windows ] && bin=$(BINARY).exe; \
	    stage=$$(mktemp -d); \
	    cp docs/config.example.yaml README.md "$$stage/" 2>/dev/null || true; \
	    echo "building $$os/$$arch ($$variant)"; \
	    CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
	      go build $$tags -trimpath -ldflags="$(LDFLAGS)" -o "$$stage/$$bin" . || exit 1; \
	    name=$(BINARY)-$(VERSION)-$$os-$$arch$$suffix; \
		tar -czf "$(DIST)/$$name.tar.gz" -C "$$stage" .; \
	    rm -rf "$$stage"; \
	  done; \
	done
	cd $(DIST) && { command -v sha256sum >/dev/null 2>&1 && sha256sum * || shasum -a 256 *; } > checksums.txt
	@ls -lh $(DIST)
