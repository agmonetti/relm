# LESSONS — agent lessons

This file documents crossroads and decisions the agent makes during development, so they can be reused in the future. It is updated continuously.

---

## L-01 — Go module without a real repo: don't use the literal `[usuario]`

**Crossroads:** the SPEC says `go mod init github.com/[usuario]/relm`, but `[usuario]` with brackets is not a valid module path for Go and there is no real repo.

**Decision:** use `module relm` (simple name). When the project was published to `github.com/agmonetti/relm`, it was changed to `module github.com/agmonetti/relm` with a `sed` (`s|"relm/|"github.com/agmonetti/relm/|g` over the `*.go` files plus the `module` line in `go.mod`), which enables `go install github.com/agmonetti/relm@latest`.

**Lesson:** when the SPEC has invalid placeholders, resolve with the simplest option and note it here instead of asking. The module path is migrated on publish with a mechanical sed.

---

## L-02 — Minimal stack first, features later

**Crossroads:** the SPEC asks for `cobra` as optional, spinners, etc. Starting everything at once complicates debugging.

**Decision:** implement in SPEC phases. Phase 1 = only `store` + `conn` + `cmd` with output to stdout. No TUI yet.

**Lesson:** follow `04-implementation.md` to the letter. Each phase has a done criterion verifiable with `go test`/`go build`. Don't skip phases.

---

## L-03 — The `Store` interface is the sacred contract

**Crossroads:** the TUI needs to know if a query returns rows or "rows affected" to render.

**Decision:** `Query()` always returns `*Result` with `Columns`/`Rows`. The UI decides how to display based on `len(Result.Columns)`. `Exec()` is for `rowsAffected`. Never import `database/sql` outside `internal/store/**`.

**Lesson:** keeping `Store` neutral is what makes it possible to add engines without touching the UI. If a requirement seems to demand engine logic outside, that's a sign the interface is wrong — fix the interface.

---

## L-04 — Dependencies with CGO complicate the build, have a plan B

**Crossroads:** `mattn/go-sqlite3` requires gcc. There is gcc on this machine, but cross-compiling will fail.

**Decision:** use `mattn` for now (the SPEC asks for it). Leave the `modernc.org/sqlite` alternative documented for CGO-less builds.

**Lesson:** the cost of CGO is paid at release time. Note the escape hatch in the README and in `05-technical-decisions.md`.

---

## L-05 — Engine per package + registration in `init()` avoids the import cycle

**Crossroads:** `store` (factory) needs to import `sqlite`, but `sqlite` imports `store` for the shared types (`store.Store`, `store.Column`...) → import cycle. It's inevitable with the SPEC's pattern.

**Decision:** registry pattern:
- `store` defines the interface, types, errors and a `Register(driver, constructor)` + `New(cfg)` that looks up the registry.
- Each engine calls `store.Register(...)` in its `init()`.
- `cmd` imports the engines blank (`_ "relm/internal/store/sqlite"`) so they register.
- `store` does NOT import any engine → no cycle.

**Lesson:** the "plugin registry with init()" is the Go pattern to avoid cycles between a contract and its implementations. The registry is build-time configuration, not runtime state — it doesn't violate "no global state".

---

## L-06 — SQLite creates nonexistent files: validate with os.Stat

**Crossroads:** the SPEC requires "file doesn't exist → error", but the SQLite driver creates a blank file when opening a nonexistent path.

**Decision:** `os.Stat(cfg.Path)` before opening; if it doesn't exist, `ErrConnection: no such file`. Exception: `:memory:` and `file::memory:` (in-memory databases for tests) skip the check.

**Lesson:** SQLite drivers have auto-creation behaviors that contradict the expected contract. Validate at the store layer, don't assume.

---

## L-07 — Factory tests go in an external package (`package store_test`)

**Crossroads:** the `store` factory test needs to import `sqlite` (to register it), and `sqlite` imports `store` → import cycle in the test.

**Decision:** test in `package store_test` (external test). The external package can import packages that import the package under test, without creating a cycle.

**Lesson:** when the registry pattern creates cycles, contract tests go in `package <name>_test`. If something "is unused" in the test after the change, it's a sign the calls were left without a prefix (`New` → `store.New`).

---

## L-08 — `Result` needs to distinguish a table from "rows affected"

**Crossroads:** the UI decides how to show a result: table if there are columns, "N rows affected" for INSERT/UPDATE/DELETE. The `Store` interface had no way to express the latter.

**Decision:** add `Affected int64` to `store.Result`. Read queries set it to `-1`; `Exec` sets it to `rowsAffected`. The UI decides with `len(Columns) > 0` (table) or `Affected >= 0` (rows affected).

**Lesson:** when a UI requirement has nowhere to live in the model, the model is incomplete — don't patch the UI. A new field on `Result` is simpler than inspecting strings in the TUI.

---

## L-09 — The SELECT heuristic is fragile; let the store dictate the truth

**Crossroads:** `Query()` or `Exec()` for the editor buffer? `strings.HasPrefix(SELECT)` fails with `WITH ... SELECT`, `PRAGMA`, `EXPLAIN`.

**Decision:** the editor uses a short list of keywords that return rows (`SELECT, WITH, PRAGMA, SHOW, EXPLAIN, DESCRIBE, VALUES, TABLE`) and documents it as a hint. The final result is dictated by the store (a `Result` with columns vs `Affected`), not the editor.

**Lesson:** every presentation decision is based on the store's `Result`, never on re-parsing SQL at the UI layer.

---

## L-10 — `Ctrl+I` and `Tab` are indistinguishable in the terminal

**Crossroads:** the SPEC mapped `Ctrl+I` to "view structure". When testing the TUI, `Ctrl+I` opened the editor: the `Tab` handler captured it.

**Decision:** in the terminal `Ctrl+I` and `Tab` are the same byte (0x09, HT). bubbletea normalizes them to `KeyTab` (its `String()` is "tab"), so a `ctrl+i` binding never matches. I changed structure inspection to `i` (info) and updated the docs.

**Lesson:** never design keymaps with `Ctrl+I` if `Tab` is taken. Before writing a binding, verify it doesn't collide with another control key with the same ANSI code (`Ctrl+J`=`Enter`, `Ctrl+M`=`Enter`/CR, `Ctrl+H`=`Backspace`, `Ctrl+I`=`Tab`). This is a class of bug that only appears when testing the real TUI, not in model unit tests — which is why it's worth testing the full flow with keys.

---

## L-11 — Test the full TUI flow by executing the `tea.Cmd`s

**Crossroads:** the Model tests passed messages with `m.Update(msg)` but ignored the returned `tea.Cmd`. The connection flow never connected in the tests, even though it worked at runtime.

**Decision:** a `step()` helper in the tests executes the returned `cmd` and feeds back the message it produces (like the bubbletea program does). Without this, deferred messages (`ConnectMsg`) never reach the model in the test.

**Lesson:** to test a message architecture you have to simulate the runtime: execute cmds and re-feed their messages. Also, the TUI tests need the engine's blank import (`_ "relm/internal/store/sqlite"`) or the registry never runs.

---

## L-12 — `tea.BatchMsg` is `[]Cmd` in bubbletea v1.x, not messages

**Crossroads:** the test helper that executed cmds treated `tea.BatchMsg` as a list of messages and passed them to `Update`; the query result was never applied and `loading` stayed `true`.

**Decision:** in bubbletea v1.3.x, `BatchMsg` is `[]Cmd` (commands). The program executes them one by one and delivers each resulting message to `Update`. The test helper now executes each sub-cmd and feeds back its message.

**Lesson:** always verify the actual signature of `tea.Batch`/`BatchMsg` in the installed version. The API changed between v0.x and v1.x.

---

## L-13 — History is lost if each execution creates a new editor

**Crossroads:** when the editor execution was made asynchronous, each query ran in a new `editor.Editor`. The history ended up with a single element because each new editor started empty.

**Decision:** share the `History` pointer between the model's editor and the goroutine's editor (`ed.History = m.editor.History`). The goroutine only mutates the ring buffer; navigation in the UI is guarded with `!m.loading` to avoid races. Verified with `go test -race`.

**Lesson:** when moving an operation to a goroutine, state that must survive between calls (history, counters) has to live in a shared object. Copying the struct doesn't share slices/pointers unless the pointer is copied explicitly.

---

## L-14 — `Ctrl+I`/`Tab` already documented; the pattern repeats (ambiguous control keys)

**Crossroads:** `Ctrl+I` was bound to "structure" but collided with `Tab`.

**Decision:** use `i`. Already documented in L-10. The lesson extends: review the full keymap before implementing to catch ANSI collisions.

**Lesson:** when a SPEC keymap is technically unfeasible, the real TUI is the only way to detect it — test the full flow with simulated keys, not just rendering.

---

## L-15 — SQL Server doesn't support boolean expressions in SELECT

**Crossroads:** mssql's column introspection query used `(c.IS_NULLABLE = 'NO')` directly in the SELECT. Error `Incorrect syntax near '='`.

**Decision:** use `CASE WHEN ... THEN 1 ELSE 0 END` to derive booleans in SQL Server. Also `ISNULL()` instead of `COALESCE()` (both work, ISNULL is the T-SQL idiosyncratic one).

**Lesson:** SQL Server's `information_schema` looks the same as Postgres/MySQL's but the T-SQL syntax differs. Integration tests against real docker containers are the only reliable way to validate introspection queries — and they're fast to run once docker is up.

---

## L-16 — Validate each engine against a real server, not just "it compiles"

**Crossroads:** the engines compiled and the unit-tested dialects passed, but real syntax and introspection bugs only appeared against real servers (docker) — the mssql case above.

**Decision:** per-engine integration tests triggered by env vars (`SQLISH_TEST_<MOTOR>_HOST`, etc.) that exercise the whole `Store` interface: tables, columns, constraints, indexes, count, pagination, version. They skip with `t.Skip` if there's no env var, so `go test ./...` always passes.

**Lesson:** the phase-7 done criterion ("connect to each engine in docker") is what really validates the work. Docker is available in this environment: use ephemeral containers (`--rm`) with standard ports.

---

## L-17 — CGO limits cross-compilation; `modernc` enables it

**Crossroads:** `CGO_ENABLED=1` cross-compiling to darwin from linux fails (`clang: unsupported option '-arch'`). The SPEC asked for a multi-platform build.

**Decision:** I verified that `modernc.org/sqlite` (pure-Go driver, registers "sqlite") compiles to `darwin/arm64` with `CGO_ENABLED=0`. Documented in the Makefile as an escape hatch. The linux binary with the 5 drivers weighs 21MB (limit: 40MB).

**Lesson:** when a dependency brings CGO, cross-compiling isn't free. Validate the pure-Go alternative concretely (actually compile it) before promising it in the docs.

---

## L-18 — The password `EchoMode` was applied to the wrong field

**Crossroads:** in the connection form, `EchoMode = EchoPassword` was applied by index `c.fields[2]` (Port) instead of `c.fields[4]` (Password). Silent bug: the password wasn't masked and the port was.

**Decision:** fix the index and add screen tests (`connect_test.go`) that verify per-engine visible fields and masking.

**Lesson:** magic indexes in field slices are fragile. A screen test that validates visible behavior (how many fields, whether it's masked) catches this class of bug that compiles but works wrong.

---

## L-19 — Documentation commands have to be tested, not copied

**Crossroads:** `USAGE.md` had `docker exec maria mysql -uroot -proot -e "CREATE DATABASE test"`. When running it: the `mariadb:11` image doesn't ship the `mysql` binary, only the `mariadb-*` ones (`mariadb`, `mariadb-admin`, ...).

**Decision:** verify every USAGE command by actually running it (sqlite one-liner, 4 `docker run`s, credentials). Fix using `mariadb` and, better, remove the manual step: the `POSTGRES_DB` / `MYSQL_DATABASE` / `MARIADB_DATABASE` env vars make the official images auto-create the database on first boot.

**Lesson:** a command that isn't run in the docs can be broken and nobody notices. The MariaDB container binaries are called `mariadb-*` (not `mysql-*`), and the official images create databases with env vars — use that instead of manual `exec` steps.

---

## L-20 — `docker compose` is the best way to provide test DBs

**Crossroads:** the user asked whether a docker image or a `docker run` is better for "bringing up the database". Four loose `docker run`s are noise and error-prone.

**Decision:** official `compose.yaml`: `docker compose up -d` brings up the 4 engines with fixed credentials, auto-created `test` database, and healthchecks. It was validated: all 4 end up `healthy` and the integration tests pass with those credentials. The per-container `docker run` alternative was kept (with env vars, no `exec`).

**Lesson:** for "bring up the test environment", `docker compose` with healthchecks is the standard, most robust option. A docker image of the TUI itself adds nothing (it needs a TTY and wouldn't include the database); the compose of the databases is what the user needs.

---

## L-22 — A checkbox in a textinput form breaks the focus model

**Crossroads:** adding the `Read-only` toggle (SQLite) and the `SSL` field (PostgreSQL) to the connection form. The toggle isn't a `textinput`, but `ConnScreen` treats all fields the same: `applyFocus()` called `.Focus()` on all of them, and `Update()` passed the key to the active input.

**Decision:** extend `field` with `isToggle bool` + `checked bool`. `applyFocus()` skips toggles (they have no `.Focus()`); `Update()` toggles with `Space`/`Enter` and doesn't forward the key to the input; `renderForm()` draws `[ ]`/`[x]`; `reset()` turns them off. The number of fields per engine changed (sqlite: 2, postgres: 6), so `connect_test.go` was updated and toggle and `sslmode` tests were added.

**Lesson:** when a visual component doesn't fit the base type of the fields (`textinput`), it isn't forced — it's modeled as a variant of the field with its own handler. The real cost is in focus (which element "owns" the key), not in rendering.

---

## L-21 — Makefile: multiline recipe lines break if they don't carry a tab

**Crossroads:** `make demo` failed with "missing separator": the multiline SQL inside the recipe had continuation lines indented with spaces, not tabs.

**Decision:** every recipe line in a Makefile MUST start with a tab. A multiline SQL string with its own indentation breaks parsing. Solution: SQL on a single line per recipe.

**Lesson:** in Makefiles, recipe indentation is a tab (not spaces). To avoid the problem, keep each command on a single line.

---

## L-23 — SQLite switched to `modernc.org/sqlite` (pure Go, no CGO)

**Crossroads:** `mattn/go-sqlite3` requires gcc/CGO. On a machine without a C toolchain the binary **builds** (mattn ships a stub) but SQLite fails at runtime with "go-sqlite3 requires cgo", and every SQLite-dependent test fails. L-04/L-17 documented `modernc.org/sqlite` as an escape hatch, but it was never wired in.

**Decision:** migrate SQLite to `modernc.org/sqlite` (registers driver `"sqlite"`). The DSN changes from mattn params to pragma syntax: `?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)` (read-only: `?mode=ro`). Direct `sql.Open("sqlite3", ...)` callers in tests switch to `"sqlite"`. Remove the gcc requirement from the docs; `make release` now uses `CGO_ENABLED=0`.

**Lesson:** a CGO driver that "builds without CGO" via a stub is a silent failure — the binary compiles but the engine is broken at runtime. Prefer a pure-Go driver when the tool must be a single portable binary. Bonus: on Windows, tests that open a SQLite file must close the store, or `TempDir` cleanup fails because the file stays locked. It was the same with mattn; the stub hid it before.




