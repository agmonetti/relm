# 05 — Technical decisions, edge cases and conventions

## Code conventions

### Naming

- Packages in lowercase, one word: `conn`, `store`, `browser`, `editor`, `tui`.
- Exported structs with descriptive names: `SQLiteStore`, `PGStore`, `MySQLStore`, `MSSQLStore`, `Browser`, `Editor`.
- Interfaces suffixed by behavior, not implementation: `Store` (not `IStore`, not `StoreInterface`).
- Errors prefixed with `Err`: `ErrUnsupportedDriver`, `ErrConnection`, `ErrTableNotFound`, `ErrInvalidQuery`.
- Screen constants as their own type: `type Screen int` with `const (ScreenConnect Screen = iota; ScreenWorkspace)`. Pane focus is `screens.WorkspaceFocus` (`FocusSidebar`, `FocusMain`, `FocusEditor`).
- Engine constructors by driver name: `sqlite.New`, `postgres.New`, `mysql.NewMySQL`, `mysql.NewMariaDB`, `mssql.New`.

### Function structure

- Prefer functions under 40 lines.
- If a function needs more, extract subfunctions with descriptive names.
- Early return for errors: don't nest `if err != nil` in a cascade.

```go
// Good
func (s *SQLiteStore) Tables() ([]string, error) {
    rows, err := s.db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
    if err != nil {
        return nil, fmt.Errorf("store.Tables: %w", err)
    }
    defer rows.Close()
    // ...
}

// Bad — don't do this
func (s *SQLiteStore) Tables() ([]string, error) {
    if rows, err := s.db.Query("..."); err == nil {
        // nested logic
    } else {
        return nil, err
    }
}
```

- Always wrap errors with `fmt.Errorf("context: %w", err)` so they are traceable.

### Comments

- Comment the "why", not the "what".
- Every exported function has a godoc comment.
- Complex SQL queries have a comment explaining what they return.

## Per-engine dialects

| Engine | QuoteIdent | Pagination | Schema source |
|---|---|---|---|
| SQLite | `"name"` | keyset `ORDER BY "pk"` / `LIMIT n` (OFFSET fallback) | `sqlite_master` + `PRAGMA` |
| PostgreSQL | `"name"` | keyset `ORDER BY "pk"` / `LIMIT n` (OFFSET fallback) | `information_schema` + `pg_indexes` |
| MySQL | `` `name` `` | keyset `ORDER BY "pk"` / `LIMIT n` (OFFSET fallback) | `information_schema` |
| MariaDB | `` `name` `` | keyset `ORDER BY "pk"` / `LIMIT n` (OFFSET fallback) | `information_schema` |
| SQL Server | `[name]` | keyset `ORDER BY "pk"` / `OFFSET..FETCH` (OFFSET fallback) | `INFORMATION_SCHEMA` + `sys.indexes` |

- Tables with a single-column primary key are browsed with **keyset pagination**
  (`WHERE "pk" > <last key> ORDER BY "pk" LIMIT n`), so refreshing never moves
  the visible rows and pages do not skip/duplicate rows between refreshes. The
  browser fetches one extra row to know whether another page exists.
- PostgreSQL casts the bound parameter (`$1::<data_type>` with the type read
  from `information_schema` and cached per table+column) so the primary-key
  index stays usable regardless of the column type.
- Tables without a primary key or with a composite primary key fall back to
  OFFSET pagination with `ORDER BY 1` (the first column).
- `LIMIT n` uses integers (page, pageSize), safe for direct interpolation.
- Identifiers (table/column names) are ALWAYS escaped with `QuoteIdent`, including the browser's active table: `SELECT * FROM "my table"`.
- `Result.Rows` is `[][]string` in all engines. The engine store converts each native type to string (including `[]byte`, `time.Time`, `decimal`) and `NULL` to `""` (the UI renders it as `∅`).

## Connection and DSN per engine

### SQLite (modernc.org/sqlite)

Pure-Go driver (no CGO/gcc). It registers the `"sqlite"` driver; pragmas go
through `_pragma=` query parameters:

```go
// Read/write mode (default)
dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", path)

// Read-only mode ("Read only" toggle in the form, cfg.ReadOnly)
dsn := fmt.Sprintf("file:%s?mode=ro&_pragma=foreign_keys(1)", path)
```

- `_pragma=journal_mode(WAL)`: better performance for concurrent reads. Not applicable in read-only.
- `_pragma=foreign_keys(1)`: SQLite doesn't enable them by default. Always enable.
- `_pragma=busy_timeout(5000)`: wait up to 5 seconds before returning `SQLITE_BUSY`.
- `ReadOnly` is also persisted in saved connections (`connections.json`, field `read_only`).
- The pure-Go driver removes the gcc requirement and enables cross-compilation with `CGO_ENABLED=0`.

### PostgreSQL (jackc/pgx/v5)

```go
// DSN via net/url in the code. sslmode comes from cfg.SSLMode (default "prefer").
u := url.URL{
    Scheme: "postgres",
    User:   url.UserPassword(cfg.User, cfg.Password),
    Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
    Path:   "/" + url.PathEscape(cfg.Database),
}
q := u.Query()
q.Set("sslmode", cfg.SSLMode) // or "prefer" if empty
u.RawQuery = q.Encode()
```

- Use `pgx/v5/stdlib` (wrapping `database/sql`). It's the most maintained integration.
- `sslmode` configurable from the form (`SSL` field) and persisted in saved connections. Default `prefer`; the TLS error is shown verbatim if it fails.
- Connection timeout of 5s: wrap `sql.Open` + `Ping` in `context.WithTimeout`.

### MySQL and MariaDB (go-sql-driver/mysql)

```go
dsn := fmt.Sprintf(
    "%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&timeout=5s",
    cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database,
)
```

- The same package works for both. MariaDB is registered separately because `Version()` and some `information_schema` responses differ.
- `parseTime=true` so `time.Time` doesn't arrive as `[]byte`.
- `timeout=5s` to avoid hanging on unreachable hosts.
- Don't use `multiStatements=true` (security).

### SQL Server (microsoft/go-mssqldb)

```go
dsn := fmt.Sprintf(
    "sqlserver://%s:%s@%s:%d?database=%s&connection+timeout=5",
    url.QueryEscape(cfg.User), url.QueryEscape(cfg.Password),
    cfg.Host, cfg.Port, url.QueryEscape(cfg.Database),
)
```

- `connection+timeout=5` seconds.
- Identifiers are escaped with `[...]`.

## Edge cases to handle

### Connection

| Case | Expected behavior |
|---|---|
| Unreachable host / connection refused | Error in footer: `cannot connect to localhost:5432: connection refused`. Return to `ScreenConnect` without crashing. |
| Wrong credentials | Engine's literal text: `password authentication failed for user "postgres"`. |
| Database doesn't exist | Engine's literal text (varies: `database "x" does not exist`, `Unknown database 'x'`). |
| Connection timeout (>5s) | `timeout connecting to localhost:5432`. |
| Unknown driver | `ErrUnsupportedDriver` (shouldn't happen: the UI selector only offers the 5). |
| SQLite: file doesn't exist | Error on the connection screen: `"archivo.db: no such file"`. |
| SQLite: file is not SQLite | `"not a valid SQLite database"`. |
| SQLite: file in exclusive use | SQLite returns `SQLITE_BUSY`. Show: `"the database is locked by another process"`. |
| SQLite: corrupted database | Show SQLite's literal error. |
| SQLite: empty file (0 bytes) | SQLite accepts it as a valid DB with no tables. Normal empty state. |

### Tables and data

| Case | Expected behavior |
|---|---|
| Table with no rows | Show column headers, empty area, text `empty table`. |
| Column with NULL value | Show `∅` in gray. |
| Very long text value (>200 chars) | Truncate with `…` to the column width. |
| Very long column name | Truncate too, with `…`. |
| Table name with spaces or special characters | Escape with the dialect's `QuoteIdent`: `"my table"`, `` `my table` ``, `[my table]`. |
| Database with 500+ tables | The sidebar shows a scrollable subset. Don't load everything into render memory. |
| Query returning 0 columns (e.g. `PRAGMA ...`) | Show the raw result as text, not as a table. |
| Binary type (`bytea`, `BLOB`, `varbinary`) | Convert to a string representation in the store: `0x...` or short base64, truncated in the UI. |
| `time.Time` / `decimal` types | The store serializes them to string. The UI never does type assertions. |

### SQL editor

| Case | Expected behavior |
|---|---|
| Empty query + `Ctrl+R` | Don't execute anything. Message: `write a query first`. |
| Query ending with semicolon | Works normally. |
| Multiple statements separated by `;` | Execute the statement under the cursor (by line). If several statements are on the same line, the first one is chosen. The split respects strings (`'...'`, `''`, `\`). |
| `DROP TABLE users` | Execute normally. `relm` doesn't ask for confirmation (the user knows what they're doing). |
| Query taking >5 seconds | No execution timeout by default. Show a spinner. The query runs with `context.WithTimeout` using the user's configured timeout (`Ctrl+P`, default 60s); `Esc` cancels it (both abort the driver call via `QueryContext`/`ExecContext`). |
| Unsupported dialect | The engine's literal SQL error is shown in red. The user writes SQL for the engine they're connected to. |

### Terminal

| Case | Expected behavior |
|---|---|
| Very small terminal (<60 cols) | Hide the sidebar automatically. If <40 cols, show a warning: `terminal too small`. |
| Very small terminal (<10 rows) | Show only header + footer with an error. |
| Resize during query | The resize is processed when the query finishes. No concurrency. |
| Mouse | Enabled with cell motion (drag-only reporting). A left click focuses a pane; a right-click drag resizes the nearest divider (sidebar width or editor height), persisted in `prefs.json`. Only active on the workspace screen. |
| Terminal without color support | lipgloss detects it automatically. Render without colors if `TERM=dumb` or similar. |
| `SIGTERM` / `Ctrl+C` | Exit cleanly: `store.Close()` before terminating. |

## Saved connections

- File: `~/.config/relm/connections.json`, permissions `0600`.
- Structure:
  ```json
  [
    { "name": "local", "driver": "postgres", "host": "localhost", "port": 5432,
      "user": "postgres", "password": "", "database": "mydb", "path": "",
      "ssl_mode": "prefer" },
    { "name": "prod", "driver": "sqlite", "path": "/data/app.db", "read_only": true }
  ]
  ```
- `password` is optional ("save password" field in the form). If saved, it goes as plain text in the JSON — tradeoff documented in `06-security.md`. The OS keychain remains a future improvement.
- `read_only` (SQLite) and `ssl_mode` (PostgreSQL) are also persisted and reused when connecting.
- The password is never shown in the form (masked field). If the saved connection has no password, the field is left empty for the user to enter it.
- The file is read when opening `ScreenConnect` and written with `Ctrl+S`. File read/write errors are silently ignored (they don't break the session).

## Preferences

- File: `~/.config/relm/prefs.json`, permissions `0600`.
- Structure:
  ```json
  { "query_timeout_seconds": 60, "sidebar_width": 0, "editor_height": 0 }
  ```
- `query_timeout_seconds` bounds every user query run from the editor
  (`context.WithTimeout`); values `<= 0` fall back to the default (60s).
- `sidebar_width` / `editor_height` persist the workspace pane sizes changed
  with a right-click drag; `0` means automatic. The values are clamped again
  against the current terminal size on every render.
- Edited from the settings screen (`Ctrl+P`, available from the connect and
  workspace screens) and saved with `Enter`. The change applies to the next
  query.
- Same directory resolution as `connections.json` (`RELM_CONFIG_DIR` override).

## User-facing messages

All visible messages follow these rules:

- In English (the project was Spanish-first by philosophy; it can be internationalized later).
- No unnecessary capitalization. Lowercase at the start except for proper nouns.
- No exclamation marks.
- No "error:" prefix — the red color already communicates the error.
- Concise: at most two lines in the footer.

Examples:

```
# Good
empty table
cannot connect to localhost:5432: connection refused
invalid query: near "FORM": syntax error

# Bad
Error! The query could not be executed because the syntax is incorrect.
ERROR: PERMISSION DENIED
```

## Testing

- Tests in `_test.go` in the same package (white box testing).
- **SQLite:** use `":memory:"` as path for store, browser and editor tests — creates no files and is the reference engine for all UI tests.
- **Other engines:** integration tests inside `internal/store/<engine>/`, triggered by separate env vars. If the var isn't set, `t.Skip`:
  ```go
  // SQLISH_TEST_POSTGRES_HOST=localhost
  // SQLISH_TEST_POSTGRES_USER=postgres
  // SQLISH_TEST_POSTGRES_PASSWORD=postgres
  // SQLISH_TEST_POSTGRES_DATABASE=test
  // (same pattern with MYSQL, MARIADB, MSSQL)
  ```
  - The setup creates a test table `relm_test`, runs the same assertions as the SQLite test.
  - Docker for local development: the repo's `compose.yaml` starts the 4 engines with fixed credentials and the auto-created `test` database (`docker compose up -d`). Full commands in `README.md`.
- Don't mock `Store` for browser/editor tests — use the real implementation with in-memory SQLite. It's more reliable.
- Test names: `TestBrowser_SelectTable_LoadsColumns`, `TestStore_Query_ReturnsError_OnInvalidSQL`.
- No external testing frameworks. Only stdlib `testing` and `errors.Is` for assertions.

## Performance considerations

- `CountTable` (the `COUNT(*)` for pagination) runs when a table is selected and
  on `Reload` (manual refresh or after a write query). It does NOT run on every
  page navigation (`PgUp`/`PgDn`): the row count is kept from the last load, so
  browsing a big table does not re-scan it on every page.
- Browser navigation (open table, change page, refresh) runs in the background,
  like editor queries: it is bounded by the configured query timeout, cancellable
  with `Esc`, and shows a spinner instead of freezing the UI.
- Editor queries cap the number of rows loaded into memory (`MaxResultRows`,
  10k); a longer result is marked `Truncated` and the UI shows a notice.
- Column widths are computed once when loading the table/result. Not on every frame.
- `View()` in bubbletea is called every frame. Don't do I/O there. Only string building.
- `lipgloss.Render()` is expensive in loops. Precompute styles as package-level variables, don't create them on every `View()` call.
- The `[]byte` → string conversion of each cell happens only once in the store, when reading the row. Don't repeat it in the UI.

## Expected initial go.mod

```
module github.com/agmonetti/relm

go 1.22

require (
    github.com/charmbracelet/bubbles v0.20.0
    github.com/charmbracelet/bubbletea v1.2.0
    github.com/charmbracelet/lipgloss v1.0.0
    github.com/go-sql-driver/mysql v1.8.1
    github.com/jackc/pgx/v5 v5.7.1
    github.com/microsoft/go-mssqldb v1.7.2
    modernc.org/sqlite v1.56.0
)
```

Exact versions to verify with `go get` at implementation time — use the latest stable ones.

## What comes after v0.1.0 (out of scope for now)

So the agent knows what NOT to implement yet:

- Export results to CSV / JSON.
- New engines (Oracle, DB2, Snowflake, NoSQL). The list is closed at five.
- Inline cell editing (too complex for v0.1).
- SQL syntax highlighting in the editor.
- Saving passwords to the OS keychain.
- Persistent query history across sessions.
- `relm <dsn>` as a CLI shortcut to skip the connection screen.
- Plugin system.

If any of these features seems necessary to get something basic working, check before implementing.
