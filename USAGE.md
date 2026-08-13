# USAGE — first steps, step by step

`relm` **does not start servers or create files on its own**. It connects to
databases that **already exist and are running** — you bring the database, `relm`
opens a window to look at it and query it.

- **SQLite** = a `.db` file on your disk (no server). You create it yourself with `sqlite3` (section 1a).
- **PostgreSQL / MySQL / MariaDB / SQL Server** = servers that must already be running (on your machine, in docker, or in the cloud). The repo ships a `compose.yaml` to start them for testing (section 2a).

There is no prior configuration or server installation done by `relm`.
You only need the binary and access to the database.

---

## 0. Build the binary

```bash
go build -o relm ./cmd/relm
./relm
```

(Requires Go 1.26+. The SQLite driver is pure-Go `modernc.org/sqlite`, so no gcc/CGO is needed; it builds and runs on Windows, Linux and macOS.)

---

## 1. First quick test with SQLite (no server needed)

### 1a. Create a test database

`relm` does not create files: if you give it a path that does not exist, it shows `no such file`.
Create the database yourself. Two options:

**With the built-in demo generator** (creates `demo.db` with example tables and
data, works on any platform — no `sqlite3` CLI needed):

```bash
go run ./cmd/demo
# or, with make: make demo
./relm
```

**By hand with the sqlite CLI:**

```bash
sqlite3 test.db "
CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT);
CREATE TABLE orders (id INTEGER PRIMARY KEY, total REAL, user_id INTEGER);
INSERT INTO users (name, email) VALUES ('Alice','alice@test.com'), ('Bob','bob@test.com');
"
```

### 1b. Connect

1. Run `./relm`. The **connection screen** opens:

```
┌─────────────────────────────────────────────┐
│ relm · no connection · — ·                 │
├─────────────────────────────────────────────┤
│                _____  ______  _      __  __ │
│               |  __ \|  ____|| |    |  \/  |│
│               | |__) | |__   | |    | \  / |│
│               |  _  /|  __|  | |    | |\/| |│
│               | | \ \| |____ | |____| |  | |│
│               |_|  \_\______|_\_____|_|  |_|│
│                                             │
│                    Connect                  │
│                                             │
│            Engine │ postgres    ←→ switch   │
│                   └─────────────────────────│
│            Host   │ localhost               │
│                   └─────────────────────────│
│            Port   │ 5432                    │
│            Database│ mydb                   │
│                                             │
│                Enter · Connect              │
│          ctrl+s  save   r  clear            │
├─────────────────────────────────────────────┤
│ ↑↓ saved · tab engine/fields · ←→ engine · enter connect │
└─────────────────────────────────────────────┘
```

- `Read-only` opens the file in `mode=ro` mode: any write fails. Useful for production databases. Toggle it with `Enter` or `Space` when the focus is on the toggle.

2. With `Tab` move to the `File` field, type `test.db` (or the full path) and press `Enter`.

3. You are now in the **workspace**: a single window with the sidebar (tables),
   the main pane (the open table) and the SQL editor at the bottom. The sidebar
   has the focus. The first table alphabetically (`orders`) is open — it is
   empty in the example database:

```
┌──────────────────────────────────────┐
│ > orders    │ empty table            │
│   users     │                        │
│             ├────────────────────────┤
│             │  ctrl+r to run         │
└──────────────────────────────────────┘
```

Press `↓` to move the sidebar cursor to `users` and `Enter` to open it in the
main pane:

```
┌──────────────────────────────────────┐
│   orders    │ id  name        email  │
│ > users     │ 1   Alice       alice  │
│             │ 2   Bob         bob    │
│             ├────────────────────────┤
│             │  ctrl+r to run         │
└──────────────────────────────────────┘
```

- In the sidebar: `↑↓` / `j k` move between tables, `Enter` opens the selected
  table. In the main pane: `↑↓` / `j k` navigate the rows, `PgUp/PgDn` change
  the page.
- `Tab` moves the focus to the next pane (sidebar → main → editor);
  `Alt+1/2/3` jump directly to a pane.
- `i` shows the active table structure (columns, constraints, indexes).
- `r` reloads the table.

### 1c. Run queries

1. Press `Tab` twice (or `Alt+3`) to focus the **SQL editor** at the bottom.
2. Type a query, e.g. `SELECT * FROM users WHERE id > 1;` and press `Ctrl+R`.
3. The result appears below the editor input, with the columns as header:

```
│ SELECT * FROM users WHERE id > 1          │
│ ─────────────────────────────────────     │
│ id  name  email                           │
│ 2   Bob   bob@test.com                    │
```

- `INSERT/UPDATE/DELETE` show `N rows affected`.
- If the query has an SQL error, the message is shown in red, without crashing.
- `↑`/`↓` with an empty input browse the history of the last 100 queries.
- `Ctrl+L` clears the input.
- After a write query (INSERT/UPDATE/DELETE/CREATE/...), the sidebar and the
  open table **auto-refresh**: no need to press `r`.
- `Esc` returns the focus to the main pane.

---

## 2. Connecting to PostgreSQL / MySQL / MariaDB / SQL Server

### 2a. If you have no server: start all 4 with a single command

The repo includes a `compose.yaml` that starts all four engines at once,
with fixed credentials and the `test` database **auto-created on first start**
(no need to create databases by hand):

```bash
docker compose up -d
```

Wait a few seconds until they are `healthy`:

```bash
docker compose ps        # all should say healthy
```

To stop them:

```bash
docker compose down      # stops, keeps the data
docker compose down -v   # stops and removes everything
```

**Alternative without compose** (one command per engine, same results):

```bash
docker run -d --rm --name pg    -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=test -p 5432:5432 postgres:16
docker run -d --rm --name mysql -e MYSQL_ROOT_PASSWORD=root   -e MYSQL_DATABASE=test -p 3306:3306 mysql:8
docker run -d --rm --name maria -e MARIADB_ROOT_PASSWORD=root -e MARIADB_DATABASE=test -p 3307:3306 mariadb:11
docker run -d --rm --name mssql -e ACCEPT_EULA=Y -e MSSQL_SA_PASSWORD='Str0ng!Passw0rd' -p 1433:1433 mcr.microsoft.com/mssql/server:2022-latest
```

To **take down a container** by name and check which containers are open:

```text
❯ docker stop pg
pg
❯ docker ps -a
CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS     NAMES
```

> In the example above the containers were started with `--rm`, so when you stop
> them they remove themselves and `docker ps -a` no longer lists them. If you want
> to keep one for reuse, start it without `--rm` (or use `docker compose`,
> which keeps the data with `docker compose down`).

> The `POSTGRES_DB` / `MYSQL_DATABASE` / `MARIADB_DATABASE` env vars make each
> server create the `test` database on its own. SQL Server does not need it: it already ships `master`.
> You can create tables directly from the `relm` editor (`Ctrl+R`) — no other client needed.

### 2b. Connect from the connection screen

1. With `←`/`→` on the **Engine** selector, pick the engine (SQLite, PostgreSQL, MySQL, MariaDB, SQL Server).
2. The form changes to `Host · Port · User · Password · Database` (PostgreSQL adds `SSL`):

```
│  Engine  [ PostgreSQL ▾ ]                  │
│  ─────────────────────                      │
│  Host     [ localhost ]                     │
│  Port     [ 5432 ]                          │
│  User     [ postgres ]                      │
│  Password [ •••••• ]                        │
│  Database [ test ]                          │
│  SSL      [ prefer ]                        │
```

> The `SSL` field controls TLS: `prefer` (default), `require`, `verify-full` (validates
> the certificate) or `disable`. For production use `require` or `verify-full`.

3. Fill in the server data. With the containers above:

| Engine | Host | Port | User | Password | Database |
|---|---|---|---|---|---|
| PostgreSQL | `localhost` | `5432` | `postgres` | `postgres` | `test` |
| MySQL | `localhost` | `3306` | `root` | `root` | `test` |
| MariaDB | `localhost` | `3307` | `root` | `root` | `test` |
| SQL Server | `localhost` | `1433` | `sa` | `Str0ng!Passw0rd` | `master` |

4. `Enter` connects. You go to the browser with the tables of that database.

> In MySQL/MariaDB an empty database shows `no tables — use the editor to create one`.
> Type `CREATE TABLE ...` in the editor and `Ctrl+R`, and the table appears in the sidebar after `r`.

### 2c. Save the connection for next time

With the connection screen ready, `Ctrl+S` saves it in
`~/.config/relm/connections.json`. Next time it appears in the
`Saved` section (below the form) and you connect with `Enter` on it.

To delete a saved connection: focus the `Saved` list with `Tab`, select it
with `↑↓` and press `d`.

> The config directory can be overridden with the `RELM_CONFIG_DIR` env var.

---

## 3. Common first-use problems

| Symptom | Cause | Solution |
|---|---|---|
| `no such file` | The `.db` path does not exist | `relm` does not create files. Create the database with `sqlite3 test.db "..."` (see section 1a) |
| `connection refused` | No server on that host/port | Start the docker container (section 2a) or check the host/port |
| `password authentication failed` / `Access denied` | Wrong credentials | Check user/password. With docker: `postgres/postgres`, `root/root`, `sa/Str0ng!Passw0rd` |
| `Unknown database` / `database "x" does not exist` | The database does not exist | Start the containers with the `POSTGRES_DB`/`MYSQL_DATABASE`/`MARIADB_DATABASE` env vars (section 2a) that create `test` on their own, or create it from the `relm` editor with `CREATE DATABASE` + `Ctrl+R` |
| The query errors | Engine SQL dialect | `relm` passes your SQL as-is to the engine. Write SQL for the engine you are connected to |
| The screen looks cut off | Terminal too small | Enlarge the terminal; below ~60 columns the sidebar is hidden |

---

## 4. Shortcut summary

| Key | Action |
|---|---|
| `Ctrl+C` / `q` | Quit |
| `Ctrl+N` | New connection |
| `Ctrl+S` | Save connection |
| `Ctrl+P` | Settings (query timeout) |
| `Tab` | Next pane (sidebar → main → editor) |
| `Alt+1` / `Alt+2` / `Alt+3` | Jump to sidebar / main / editor |
| `i` | Active table structure |
| `v` | Row detail (full values) |
| `Enter` (sidebar) | Open the selected table |
| `↑↓` / `j k` | Navigate rows (main) or tables (sidebar) |
| `PgUp` / `PgDn` | Change page |
| `r` | Refresh |
| right-click drag | Resize panes (sidebar / editor) |
| click | Focus / select a row |
| wheel | Scroll the pane under the pointer |
| `Ctrl+R` | Run query (editor) |
| `Esc` (while running) | Cancel the running query |
| `Ctrl+L` | Clear editor |
| `↑↓` (editor, empty input) | Query history |
| `?` | Help |
