<p align="center">
  <img src="assets/icon.png" alt="relm" width="240">
</p>

<h1 align="center">relm</h1>

<p align="center">
  A TUI database browser for people who don't leave the terminal.<br>
  SQLite · PostgreSQL · MySQL · MariaDB · SQL Server
</p>

---

<p align="center">
  <img src="assets/demo.gif" alt="relm demo" width="85%">
</p>

Browse tables, run queries and inspect schemas — all from the keyboard, all in one window. No Electron, no browser tab, no mouse required.

> **First time?** `relm` connects to databases that already exist — it doesn't start servers.
> See **[USAGE.md](USAGE.md)** for a step-by-step guide: create a test database, connect and run your first query.
>
> **Try it right now:** `go run ./cmd/demo` creates `demo.db` with 20 tables and
> a few thousand rows each (no server needed). With `docker compose up -d` first,
> `go run ./cmd/demo --all` seeds the same dataset into PostgreSQL, MySQL,
> MariaDB and SQL Server too.

## Install

Requires Go 1.26.6+. No CGO, no gcc, no libsqlite — the SQLite driver is pure Go.

```bash
go install github.com/agmonetti/relm@latest
```

Or build from source:

```bash
git clone https://github.com/agmonetti/relm
cd relm
go build -o relm ./cmd/relm
```

## Usage

```bash
relm
```

The connection screen opens. Pick the engine with `←`/`→`, fill in the fields and press `Enter`. For SQLite you only need the file path.

To skip the connection screen, pass a DSN directly:

```bash
relm ./app.db                                # SQLite file
relm postgres://user:pass@host:5432/mydb     # or mysql://, mariadb://, sqlserver://…?tls=…
relm --read-only postgres://user:pass@host:5432/mydb   # block every write
```

## What you get

- **Single-window layout** — sidebar, table browser and SQL editor always visible, always in sync.
- **Keyset pagination** — pages are anchored to the primary key, so refreshing or re-sorting a large table never moves the visible rows.
- **Live editor** — multiline SQL with history of your last 100 queries, runs in the background, cancellable with `Esc`. The history survives restarts (`~/.config/relm/history.json`).
- **Auto-refresh** — after any write query the sidebar and open table reload automatically.
- **Table structure** — columns, constraints and indexes with `i`.
- **Row detail** — full values of any row with `v`, no truncation.
- **Export** — query results or the browsed table to CSV/JSON with `Alt+E` (format from the file extension; SQL NULL becomes JSON `null`).
- **Saved connections** — stored in `~/.config/relm/connections.json`.
- **Read-only mode** for all five engines (SQLite via `mode=ro`, the rest at
  session level): toggle in the form or `--read-only` on the command line.
- **SSL/TLS** for every network engine: PostgreSQL (`prefer`/`require`/
  `verify-full`/`disable`), MySQL/MariaDB and SQL Server (`prefer`/`require`/
  `disable`).
- `∅` for NULL · `…` for long values · no setup required.

## Shortcuts

| Key | Action |
|---|---|
| `Tab` | Cycle focus: sidebar → browser → editor |
| `Alt+1` / `Alt+2` / `Alt+3` | Jump directly to sidebar / browser / editor |
| `↑↓` / `k j` | Navigate rows or tables |
| `Enter` (sidebar) | Open table |
| `PgUp` / `PgDn` | Change page |
| `i` | Table structure |
| `v` | Row detail |
| `r` | Refresh |
| `Alt+E` | Export result / current table page to CSV or JSON |
| `Alt+B` | Toggle sidebar |
| `Ctrl+R` | Run query |
| `Esc` | Cancel running query |
| `Ctrl+L` | Clear editor |
| `↑↓` (editor) | Query history |
| `Ctrl+N` | New connection |
| `Ctrl+S` | Save connection |
| `Ctrl+P` | Settings (query timeout) |
| `?` | Help |
| `Ctrl+C` / `q` | Quit |
| right-click drag | Resize panes |
| scroll wheel | Scroll pane under cursor |

## Stack

Go 1.26.6+ · [bubbletea](https://github.com/charmbracelet/bubbletea) · [lipgloss](https://github.com/charmbracelet/lipgloss) · [bubbles](https://github.com/charmbracelet/bubbles)

Drivers: `modernc.org/sqlite` · `jackc/pgx/v5` · `go-sql-driver/mysql` · `microsoft/go-mssqldb`

## Development

```bash
go run ./cmd/relm   # run
go test ./...       # tests
go vet ./...        # lint
```

`relm --print-layout` prints the connect screen and workspace as plain text —
useful to report layout bugs from any terminal. Pass `--width`/`--height` to
force the terminal size (e.g. `relm --print-layout --width 80 --height 40`),
which makes layout bugs reproducible in CI without a real terminal.

### Integration tests (network engines)

```bash
docker compose up -d --wait

RELM_TEST_POSTGRES_HOST=localhost RELM_TEST_POSTGRES_USER=postgres \
RELM_TEST_POSTGRES_PASSWORD=postgres RELM_TEST_POSTGRES_DATABASE=test \
RELM_TEST_MYSQL_HOST=localhost RELM_TEST_MYSQL_USER=root \
RELM_TEST_MYSQL_PASSWORD=root RELM_TEST_MYSQL_DATABASE=test \
RELM_TEST_MARIADB_HOST=localhost RELM_TEST_MARIADB_USER=root \
RELM_TEST_MARIADB_PASSWORD=root RELM_TEST_MARIADB_DATABASE=test \
RELM_TEST_MSSQL_HOST=localhost RELM_TEST_MSSQL_USER=sa \
RELM_TEST_MSSQL_PASSWORD='Str0ng!Passw0rd' RELM_TEST_MSSQL_DATABASE=master \
go test -timeout 300s ./internal/store/...
```

The CI `integration` job runs exactly these against `docker compose`, so the
network drivers are exercised on every push and pull request (not only locally).

A copy of these variables (with the same defaults) is in
[`.env.example`](.env.example); Go does not load `.env` files, so export them
by hand as shown above.

## Documentation

- **[USAGE.md](USAGE.md)** — first steps, step by step: create a test database,
  connect, run your first query.
- **`docs/design/`** — the design documents behind the project: vision,
  architecture, UX, implementation phases, technical decisions and the security
  threat model (maintainers).
- **`docs/LESSONS.md`** — engineering decisions and pitfalls recorded during
  development.

## Roadmap

Plans documented in the design docs (`docs/design/06-security.md` and
`docs/design/05-technical-decisions.md`), not yet implemented:

- OS keychain integration for saved passwords (removes the plaintext tradeoff).
- Inline cell editing and SQL syntax highlighting.

Implemented:
- Export results to CSV/JSON (`Alt+E`).
- `relm <dsn>` to skip the connection screen.
- Global `--read-only` for all five engines (SQL Server: connect with a
  read-only user — see USAGE).
- TLS options for MySQL/MariaDB and SQL Server.
- Persistent query history across sessions.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the setup, the checks to run
(`gofmt`, `go vet`, `go test`) and the pull request workflow.

## License

MIT — see [LICENSE](LICENSE).

---

