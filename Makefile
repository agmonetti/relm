BINARY := relm
PKG    := ./cmd/relm

.PHONY: build test lint clean demo

build:
	go build -o bin/$(BINARY) $(PKG)

# Los tests de motores de red se saltan sin env vars. Ver README.
test:
	go test ./...

lint:
	go vet ./...

# Crea una base SQLite de ejemplo para probar sin docker.
# Uso: make demo && ./bin/relm
demo:
	rm -f demo.db
	sqlite3 demo.db "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT); CREATE TABLE orders (id INTEGER PRIMARY KEY, total REAL, user_id INTEGER); INSERT INTO users (name, email) VALUES ('Alice','alice@test.com'), ('Bob','bob@test.com'), ('Carol','carol@test.com'); INSERT INTO orders (total, user_id) VALUES (19.99, 1), (4.50, 2), (129.00, 1);"
	@echo "Base de ejemplo creada: demo.db - abrila con ./bin/relm (motor SQLite, path demo.db)"

clean:
	rm -rf bin

# Build multi-plataforma.
#
# SQLite usa CGO (mattn/go-sqlite3), así que cross-compile a darwin requiere
# osxcross (o CI con macOS). Alternativa sin CGO: cambiar el import de
# `mattn/go-sqlite3` por `modernc.org/sqlite` en internal/store/sqlite/
# (verificado: compila a darwin/arm64 con CGO_ENABLED=0).
release:
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=1 go build -o bin/$(BINARY)-linux-amd64   $(PKG)
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=1 go build -o bin/$(BINARY)-darwin-amd64  $(PKG)
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=1 go build -o bin/$(BINARY)-darwin-arm64  $(PKG)
