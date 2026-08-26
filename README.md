<p align="center">
  <img src="assets/icon.png" alt="relm" width="240">
</p>

<h1 align="center">relm</h1>

<p align="center">
  A TUI data browser for people who don't leave the terminal.<br>
  <strong>Relational · Document · Key-Value · Wide-Column · Graph</strong><br>
  SQLite · PostgreSQL · MySQL · MariaDB · SQL Server · MongoDB · Redis · Cassandra · Neo4j
</p>

---

<p align="center">
  <img src="assets/demo.gif" alt="relm demo" width="85%">
</p>

Browse tables, documents, key-values, and graph structures, run queries and inspect schemas — all from the keyboard, all in one window. No Electron, no browser tab, no mouse required.

> **First time?** `relm` connects to databases that already exist — it doesn't start servers.
> See **[USAGE.md](USAGE.md)** for a step-by-step guide: create a test database, connect and run your first query.
>
> **Try it right now:** `go run ./cmd/demo` creates `demo.db` with 20 tables and
> a few thousand rows each (no server needed). With `docker compose up -d` first,
> `go run ./cmd/demo --all` seeds the same dataset into PostgreSQL, MySQL,
> MariaDB, SQL Server, MongoDB, Redis, Cassandra, and Neo4j too.

## Supported Data Paradigms & Engines

| Paradigm | Engines | Catalog Unit | Data View | Structure Inspector | Query Language |
|---|---|---|---|---|---|
| **Relational** | SQLite, PostgreSQL, MySQL, MariaDB, SQL Server | Tables | Table rows (keyset pagination) | Columns (PK, NN, DEF), Indexes | SQL |
| **Document** | MongoDB | Collections | BSON/JSON Documents, IDs, Summaries | Collection stats, Indexes, Inferred Fields | MQL (`db.col.find()`, JSON) |
| **Key-Value** | Redis | Keys (with badge by type) | Strings, Hashes, Lists, Sets, ZSets | Key info, TTL, Memory, Server metrics | RESP Commands (`GET`, `HGETALL`) |
| **Wide-Column** | Apache Cassandra / ScyllaDB | Tables | Column rows (page state cursors) | Partition Keys (PK), Clustering Columns (CC) | CQL |
| **Graph** | Neo4j | Node Labels | Nodes, Labels, Properties, Incident Edges | Label schema, Properties, Indexes, Relationships | Cypher |

## Install

Requires Go 1.26.6+.

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
# Relational
relm ./app.db                                         # SQLite file
relm postgres://postgres:postgres@localhost:5432/test  # PostgreSQL
relm mysql://root:root@localhost:3306/test            # MySQL
relm 'sqlserver://sa:Str0ng!Passw0rd@localhost:1433?database=master'

# Non-Relational
relm mongodb://localhost:27017/test                   # MongoDB
relm redis://localhost:6379/0                         # Redis
relm cassandra://localhost:9042/relm_demo             # Cassandra
relm neo4j://neo4j:password@localhost:7687/neo4j     # Neo4j

# Read-only enforcement
relm --read-only postgres://user:pass@host:5432/mydb
relm --read-only redis://localhost:6379/0
```

## What you get

- **Single-window adaptive layout** — sidebar (catalog), main data viewer (tables, documents, key-values, graph nodes) and editor always visible, always in sync.
- **Multi-paradigm pagination** — relational keyset pagination, MongoDB ObjectIds, Redis SCAN/paging, Cassandra page states, and Cypher skip/limits.
- **Native Query Editor** — executes SQL, MQL, RESP, CQL, and Cypher with history of your last 100 queries (`~/.config/relm/history.json`).
- **Auto-refresh** — after any write query the catalog and active item reload automatically in the background.
- **Structure Inspector (`i`)** — view columns and indexes (relational), collection stats and inferred schema (MongoDB), key memory and server stats (Redis), Partition Keys & Clustering Columns (Cassandra), or node label schemas & relationship types (Neo4j).
- **Detail View (`v`)** — full values for table rows, pretty-printed JSON for documents, full key-value entries, and graph node properties + incident edges.
- **Universal Export (`Alt+E`)** — export tabular queries, documents, and key structures directly to CSV or formatted JSON.
- **Saved connections** — stored securely in `~/.config/relm/connections.json`.
- **Read-only enforcement** across all 9 engines.

## Shortcuts

| Key | Action |
|---|---|
| `Tab` | Cycle focus: sidebar → main view → editor |
| `Alt+1` / `Alt+2` / `Alt+3` | Jump directly to sidebar / main view / editor |
| `↑↓` / `k j` | Navigate rows, items, documents, keys, or nodes |
| `Enter` (sidebar) | Open item |
| `PgUp` / `PgDn` | Change page / cursor forward or backward |
| `i` | Structure inspection |
| `v` | Detail view (full row, JSON document, KV entries, graph node) |
| `r` | Refresh active item and catalog |
| `Alt+E` | Export result / current page to CSV or JSON |
| `Alt+B` | Toggle sidebar |
| `Ctrl+R` | Run query (SQL / MQL / RESP / CQL / Cypher) |
| `Esc` | Cancel running query or return from detail/structure |
| `Ctrl+L` | Clear editor |
| `↑↓` (editor) | Query history |
| `Ctrl+N` | New connection |
| `Ctrl+S` | Save connection |
| `Ctrl+P` | Settings (query timeout) |
| `?` | Help |
| `Ctrl+C` / `q` | Quit |
| right-click drag | Resize panes |
| scroll wheel | Scroll pane under cursor |

## Development

```bash
go run ./cmd/relm   # run
go test ./...       # tests
go vet ./...        # lint
```

### Integration environment (all 9 engines)

```bash
docker compose up -d --wait

make demo-all       # seeds all 9 engines with sample data
```

## Documentation

- **[USAGE.md](USAGE.md)** — step-by-step guide for relational and non-relational engines.
- **`docs/non-relational/`** — research, composable architecture, engine profiles, interaction design, and technical limitations.
- **`docs/design/`** — original foundational design specifications.

## License

MIT — see [LICENSE](LICENSE).
