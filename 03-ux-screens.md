# 03 — UX, screens and keymaps

## Screen 0: Connection

It is the first screen when opening the tool. lazyvim style: logo centered on top, and below it the form and the saved connections, all centered in the available area. Each field is a row `label (left) + bordered box`, and there is a bottom button bar with distinguishable actions.

```
┌─────────────────────────────────────────────┐
│ relm · no connection                       │
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
│            Engine │ sqlite      ←→ switch  │
│                   └─────────────────────────│
│            File   │ /data/app.db            │
│                   └─────────────────────────│
│            Read-only │ [ ] open in read mode│
│                      │                      │
│                                             │
│                Enter · Connect              │
│          ctrl+s save   r clear              │
├─────────────────────────────────────────────┤
│ ↑↓ list  Tab engine  Enter connect  ? help │
└─────────────────────────────────────────────┘
```

When choosing another engine, the form changes:

```
│  Engine  [ PostgreSQL ▾ ]                   │
│  ─────────────────────                       │
│  Host     [ localhost ]                      │
│  Port     [ 5432 ]                           │
│  User     [ postgres ]                       │
│  Password [ •••••••• ]                       │
│  Database [ mydb ]                           │
│  SSL      [ prefer ]                         │
│                                              │
│  [ Connect ]                                 │
```

**Behavior:**
- The engine selector offers exactly: **SQLite, PostgreSQL, MySQL, MariaDB, SQL Server**. No others.
- SQLite shows the `File` field and the `Read-only` toggle (toggled with `Enter`/`Space`; opens the file in `mode=ro`).
- The other engines show `Host`, `Port`, `User`, `Password`, `Database`.
- PostgreSQL adds the `SSL` field (`prefer`, `require`, `verify-full`, `disable`; default `prefer`). Invalid values block the connection with an error.
- `Enter` connects. Connection errors are shown in the footer in red, without crashing.
- `Ctrl+S` saves the current connection to the local history (see `05-technical-decisions.md`).
- Saved connections are listed below the form (only if there are any); `Enter` on one loads and connects it.
- Connecting resets the previous session: it closes the previous store, creates the new one, and loads the browser.
- `Ctrl+N` on any screen returns to connection (new session).

## General layout (work screens)

The terminal is divided into three fixed zones:

```
┌─────────────────────────────────────────────┐
│ HEADER (1 line)                             │
│ relm · postgres@localhost:5432/mydb · users · browser │
├───────────────┬─────────────────────────────┤
│               │                             │
│  SIDEBAR      │   MAIN CONTENT              │
│  (tables)     │   (browser or editor)       │
│  ~22 cols     │                             │
│               │                             │
├───────────────┴─────────────────────────────┤
│ FOOTER (1 line)                             │
│ [contextual shortcuts]          [row X/N]   │
└─────────────────────────────────────────────┘
```

- The header is always visible. Format: `relm · <engine> <identification> · <active table> · <mode>`.
  - Network: `relm · postgres@localhost:5432/mydb · users · browser`
  - SQLite: `relm · sqlite /data/app.db · users · browser`
- The sidebar lists the database tables. It can be hidden with `Alt+B`.
- The footer shows the most relevant shortcuts for the current screen.
- The main content changes according to the active mode.

## Screen 1: Browser

Default mode after connecting.

```
┌─────────────────────────────────────────────┐
│ relm · postgres@localhost:5432/mydb · users · browser │
├──────────────┬──────────────────────────────┤
│ > users      │ id  name        email        │
│   orders     │ ─── ─────────── ──────────── │
│   products   │  1  Alice       a@test.com   │
│   sessions   │  2  Bob         b@test.com   │
│              │  3  Carol       c@test.com   │
│              │ ...                          │
├──────────────┴──────────────────────────────┤
│ ↑↓ navigate  Tab editor  ? help  50/1240 ▼  │
└─────────────────────────────────────────────┘
```

**Behavior:**
- On connecting, it loads the first table alphabetically.
- Shows 50 rows per page (configurable in the future).
- The ID column (if it exists) is always the first one.
- Column widths are calculated automatically to the maximum of the visible content, without exceeding the available space.
- `NULL` values are shown as `∅` in dim gray.
- Very long values are truncated with `…` to the column width.

**Sidebar:**
- `↑↓` navigates the table list.
- `Enter` selects the table and loads the browser.
- `1..9` quick selection by index.
- The active table name is shown with a `>` cursor.

## Screen 2: SQL Editor

Activated with `Tab` from the browser, or `Ctrl+E`.

```
┌─────────────────────────────────────────────┐
│ relm · postgres@localhost:5432/mydb · — · editor │
├──────────────┬──────────────────────────────┤
│ > users      │ SELECT * FROM users          │
│   orders     │ WHERE created_at > '2024-01' │
│   products   │ LIMIT 10;                    │
│   sessions   │ ─────────────────────────── │
│              │ id  name    email            │
│              │  4  Dave    d@test.com       │
│              │  7  Eve     e@test.com       │
│              │                             │
├──────────────┴──────────────────────────────┤
│ Ctrl+R run  ↑↓ history  Tab browser         │
└─────────────────────────────────────────────┘
```

**Behavior:**
- The input area occupies the upper half of the content.
- The results appear below, separated by a line.
- `Ctrl+R` runs the current query.
- If the query is a `SELECT`, it shows the results in a table.
- If it is `INSERT/UPDATE/DELETE`, it shows "N rows affected".
- If there is an SQL error, it shows the engine's literal message in red, without crashing.
- `↑↓` in the input navigates the history (last 100 queries, ring buffer).
- The input supports multiple lines with `Enter`. `Ctrl+R` always runs the entire buffer.

## Screen 3: Table structure

Activated with `i` from the browser (info/inspect).

> **Note:** the original SPEC used `Ctrl+I`, but in the terminal `Ctrl+I` and `Tab` are the same byte (0x09) and indistinguishable. `i` is used to inspect the structure.

```
┌─────────────────────────────────────────────┐
│ relm · postgres@localhost:5432/mydb · users · structure │
├──────────────┬──────────────────────────────┤
│ > users      │ Columns                      │
│   orders     │ ─────────────────────────── │
│              │ id       INTEGER  PK  NN     │
│              │ name     TEXT         NN     │
│              │ email    TEXT         NN     │
│              │ created  TIMESTAMP           │
│              │                             │
│              │ Indexes                     │
│              │ ─────────────────────────── │
│              │ idx_email  (email)  UNIQUE   │
│              │                             │
├──────────────┴──────────────────────────────┤
│ Esc back  Tab editor                        │
└─────────────────────────────────────────────┘
```

## Full keymap

### Global (available on any screen)

| Key | Action |
|---|---|
| `Ctrl+C` / `q` | Quit |
| `Ctrl+N` | New connection (closes the current session) |
| `Ctrl+S` | Save current connection (on the connection screen) |
| `Alt+B` | Toggle sidebar |
| `?` | Toggle help panel |
| `Tab` | Toggle between browser and editor |
| `i` | View active table structure |

### Connection screen

| Key | Action |
|---|---|
| `↑` / `↓` | Navigate list of saved connections (left panel) |
| `Tab` | Switch between form fields |
| `Enter` | Connect |
| `Ctrl+S` | Save connection |
| `r` | Reset form to blank |

### Browser

| Key | Action |
|---|---|
| `↑` / `k` | Previous row |
| `↓` / `j` | Next row |
| `PgUp` / `Ctrl+U` | Previous page |
| `PgDn` / `Ctrl+D` | Next page |
| `g` / `Home` | First row |
| `G` / `End` | Last row |
| `r` | Refresh (reloads table) |
| `1..9` | Select table by index in sidebar |

### Editor

| Key | Action |
|---|---|
| `Ctrl+R` | Run query |
| `↑` (on empty input) | Previous query in history |
| `↓` (on empty input) | Next query in history |
| `Ctrl+L` | Clear input |
| `Ctrl+A` | Go to start of line |
| `Ctrl+E` | Go to end of line |
| `Esc` | Cancel / return to browser |

## Styles and colors

Use a palette adaptive to the terminal theme (don't hardcode RGB colors). Lipgloss with `lipgloss.AdaptiveColor`:

```go
var (
    ColorPrimary   = lipgloss.AdaptiveColor{Light: "#1D9E75", Dark: "#5DCAA5"} // teal
    ColorAccent    = lipgloss.AdaptiveColor{Light: "#534AB7", Dark: "#AFA9EC"} // purple
    ColorMuted     = lipgloss.AdaptiveColor{Light: "#888780", Dark: "#B4B2A9"} // gray
    ColorError     = lipgloss.AdaptiveColor{Light: "#A32D2D", Dark: "#F09595"} // red
    ColorNull      = lipgloss.AdaptiveColor{Light: "#B4B2A9", Dark: "#5F5E5A"} // dim gray
    ColorBorder    = lipgloss.AdaptiveColor{Light: "#D3D1C7", Dark: "#444441"} // border gray
)
```

- Header: `ColorPrimary` bold.
- Active table in sidebar: `ColorAccent`.
- Errors: `ColorError`.
- NULL: `ColorNull` with `∅` text.
- Separator borders: `ColorBorder` with `lipgloss.Border(lipgloss.NormalBorder())`.
- Footer: `ColorMuted`.

## Loading states

When a query takes longer than 100ms, show in the footer:
```
running query…
```
Using `tea.Tick` with a bubbletea spinner (`bubbles/spinner`). Don't block the event loop.

## Error messages

Never crash with `panic`. Always show the error in the footer or in the results area:

- Connection error: in footer, red. Example: `cannot connect to localhost:5432: connection refused`
- Wrong credentials: engine's literal text: `password authentication failed for user "postgres"`
- SQL query error: in results area, red. Engine's literal text.
- Empty table: centered message in results area: `empty table`
- Database without tables: `no tables — use the editor to create one`
- Database without permission to list: engine's literal text.
