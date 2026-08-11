# Security — quick guide for `relm` users

`relm` runs in your terminal and connects to databases **you** choose. Everything that goes through `relm` stays on your machine and in the database you connect to. Here's what you need to know.

## What it saves and where

When you save a connection with `Ctrl+S`, `relm` writes `~/.config/relm/connections.json`:

- `0600` permissions (only your user can read it).
- If you check **save password**, it's stored **in plaintext** in that file.
- That's why: **don't save the password on a shared machine**, and don't leave the file in unencrypted backups. If in doubt, leave the password field empty when saving: `relm` will ask for it when connecting.

## Connecting securely

- **Use the `SSL` field in PostgreSQL.** By default it's `prefer` (uses TLS if the server offers it). For production databases, set `require` or `verify-full` (the latter also validates the certificate).
- **`Read-only` mode for SQLite.** If you're going to open a data file you don't want to touch, enable the `Read-only` toggle: any write fails instead of modifying the file.
- **Use a database user with minimal permissions.** `relm` uses the credentials as-is: connect with a user that only has what you need, not the default `root`/`sa`/`admin`.

## Destructive queries

`relm` **doesn't ask for confirmation** for `DROP`, `TRUNCATE`, `UPDATE`/`DELETE` without `WHERE` — it assumes you know what you're doing. Before running something destructive against a database that matters:

- Check in the header which database you're connected to (engine, host, database).
- If it's SQLite, open it in `Read-only` if you're only going to look.
- Write the query with a `WHERE` first, run a `SELECT` of the affected rows, and only then the `UPDATE`/`DELETE`.

## Sensitive data

- There's no logging to file: errors are shown on screen and gone when you exit.
- Query history is in-memory only (last 100, lost on close).
- The engine protects network traffic: for connections to external servers, prefer TLS (`SSL` field in PostgreSQL).

---

Technical details and threat model for maintainers: `06-security.md`.
