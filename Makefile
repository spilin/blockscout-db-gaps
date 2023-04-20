GO = go
GOFLAGS = -ldflags="-s -w"
UPX = upx
UPXFLAGS = --quiet --best --lzma

# The default target:

all: blockscout-db-gaps

.PHONY: all

# Output executables:

blockscout-db-gaps: main.go
	GOARCH=amd64 CGO_ENABLED=0 $(GO) build $(GOFLAGS) -o $@ $^ && $(UPX) $(UPXFLAGS) $@

# Rules for development:

clean:
	@rm -Rf blockscout-db-gaps *~

distclean: clean

mostlyclean: clean

maintainer-clean: clean

.PHONY: clean distclean mostlyclean maintainer-clean

.SECONDARY:
.SUFFIXES:
