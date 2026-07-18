NAME    := claude-release-notes
VERSION := 0.1.0
ARCH    := $(shell dpkg --print-architecture)
BUILD   := build
PKGDIR  := $(BUILD)/pkg
DEB     := $(BUILD)/$(NAME)_$(VERSION)_$(ARCH).deb

GO_SOURCES := $(shell find cmd internal -name '*.go') go.mod go.sum

.PHONY: dpkg binary clean
dpkg: $(DEB)

binary: $(BUILD)/$(NAME)

$(BUILD)/$(NAME): $(GO_SOURCES)
	mkdir -p $(BUILD)
	CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o $@ ./cmd/$(NAME)

$(DEB): $(BUILD)/$(NAME)
	rm -rf $(PKGDIR)
	install -D -m 0755 $(BUILD)/$(NAME) $(PKGDIR)/usr/bin/$(NAME)
	mkdir -p $(PKGDIR)/DEBIAN
	printf 'Package: %s\nVersion: %s\nArchitecture: %s\nMaintainer: %s\nSection: utils\nPriority: optional\nDescription: %s\n' \
		'$(NAME)' \
		'$(VERSION)' \
		'$(ARCH)' \
		'sakura <tksmmn+claude@gmail.com>' \
		'Fetch Claude Code release notes, translate to Japanese, and email them' \
		> $(PKGDIR)/DEBIAN/control
	dpkg-deb --build --root-owner-group $(PKGDIR) $@

clean:
	rm -rf $(BUILD)
