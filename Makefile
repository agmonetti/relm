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

# Creates an example SQLite database to test without docker.
# Usage: make demo && ./bin/relm
demo:
	rm -f demo.db
	sqlite3 demo.db "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT); CREATE TABLE orders (id INTEGER PRIMARY KEY, total REAL, user_id INTEGER); INSERT INTO users (name, email) VALUES ('Alice','alice@test.com'), ('Bob','bob@test.com'), ('Carol','carol@test.com'); INSERT INTO orders (total, user_id) VALUES (19.99, 1), (4.50, 2), (129.00, 1);"
	@echo "Example database created: demo.db - open it with ./bin/relm (engine SQLite, path demo.db)"

clean:
	rm -rf bin

# Multi-platform build.
#
# SQLite uses CGO (mattn/go-sqlite3), so cross-compiling to darwin requires
# osxcross (or CI on macOS). CGO-free alternative: swap the import of
# `mattn/go-sqlite3` for `modernc.org/sqlite` in internal/store/sqlite/
# (verified: builds to darwin/arm64 with CGO_ENABLED=0).
release:
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=1 go build -o bin/$(BINARY)-linux-amd64   $(PKG)
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=1 go build -o bin/$(BINARY)-darwin-amd64  $(PKG)
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=1 go build -o bin/$(BINARY)-darwin-arm64  $(PKG)
