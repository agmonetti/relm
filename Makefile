BINARY := relm
PKG    := ./cmd/relm

.PHONY: build test lint clean demo

build:
	go build -o bin/$(BINARY) $(PKG)

# Network engine tests are skipped without env vars. See README.
test:
	go test ./...

lint:
	go vet ./...

# Creates an example SQLite database to test without docker (no sqlite3 CLI
# needed; works on any platform). Usage: make demo && ./bin/relm
demo:
	go run ./cmd/demo

clean:
	rm -rf bin

# Multi-platform build.
#
# The SQLite driver is modernc.org/sqlite (pure Go), so cross-compiling works
# with CGO_ENABLED=0 on all platforms, no gcc required.
release:
	GOOS=linux    GOARCH=amd64 CGO_ENABLED=0 go build -o bin/$(BINARY)-linux-amd64   $(PKG)
	GOOS=darwin   GOARCH=amd64 CGO_ENABLED=0 go build -o bin/$(BINARY)-darwin-amd64  $(PKG)
	GOOS=darwin   GOARCH=arm64 CGO_ENABLED=0 go build -o bin/$(BINARY)-darwin-arm64  $(PKG)
	GOOS=windows  GOARCH=amd64 CGO_ENABLED=0 go build -o bin/$(BINARY)-windows-amd64.exe $(PKG)
