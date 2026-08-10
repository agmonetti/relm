# relm

Browser de bases de datos para la terminal. Explora, consulta y edita bases de datos desde el teclado, sin salir de tu terminal.

Soporta exactamente **cinco motores**: SQLite, PostgreSQL, MySQL, MariaDB y SQL Server — y ninguno más.

> **¿Primera vez?** `relm` no levanta servidores: se conecta a bases que ya existen
> (un archivo `.db` para SQLite, un servidor para el resto). Seguí la guía
> paso a paso en **[USAGE.md](USAGE.md)** — cómo crear una base de prueba, conectarte
> por primera vez y correr queries.

## Instalación

Requiere Go 1.22+ y `gcc` (por el driver CGO de SQLite).

```bash
go install github.com/[usuario]/relm@latest
# o desde el repo:
go build -o relm ./cmd/relm
```

> **Sin gcc:** SQLite se puede reemplazar por el driver puro-Go `modernc.org/sqlite`
> en `internal/store/sqlite/` — el resto del sistema no cambia.

## Uso

```bash
relm
```

Se abre la **pantalla de conexión**. Elegí el motor con `←`/`→`, completá los campos y presioná `Enter` para conectar. Para SQLite solo necesitás el path del archivo.

### Atajos

| Tecla | Acción |
|---|---|
| `Ctrl+C` / `q` | Salir |
| `Ctrl+N` | Nueva conexión |
| `Ctrl+S` | Guardar conexión (en pantalla de conexión) |
| `Tab` | Alternar browser ↔ editor |
| `i` | Ver estructura de la tabla activa |
| `↑↓` / `k j` | Navegar filas |
| `PgUp` / `PgDn` | Cambiar de página |
| `r` | Refrescar tabla |
| `Alt+B` | Mostrar/ocultar sidebar |
| `Ctrl+R` | Ejecutar query (en el editor) |
| `Ctrl+L` | Limpiar input del editor |
| `↑↓` (en editor) | Navegar historial de queries |
| `?` | Ayuda |

## Características

- Browser de tablas con paginación (50 filas por página) y sidebar navegable.
- Editor SQL multilínea con historial de los últimos 100 queries.
- Estructura de tablas: columnas, constraints e índices.
- Conexiones guardadas en `~/.config/relm/connections.json`.
- NULL se muestra como `∅`; valores largos se truncan con `…`.
- Sin servidor, sin configuración previa, un solo binario.

## Desarrollo

```bash
go test ./...        # tests
go vet ./...         # lint estático
go run ./cmd/relm  # correr
```

### Tests de motores de red

Los tests de PostgreSQL/MySQL/MariaDB/SQL Server son de integración y se
saltan salvo que se setee la env var correspondiente. Levantá los servidores
con el compose incluido y apuntá los tests a ellos:

```bash
docker compose up -d

SQLISH_TEST_POSTGRES_HOST=localhost SQLISH_TEST_POSTGRES_USER=postgres SQLISH_TEST_POSTGRES_PASSWORD=postgres SQLISH_TEST_POSTGRES_DATABASE=test \
SQLISH_TEST_MYSQL_HOST=localhost    SQLISH_TEST_MYSQL_USER=root        SQLISH_TEST_MYSQL_PASSWORD=root        SQLISH_TEST_MYSQL_DATABASE=test \
SQLISH_TEST_MARIADB_HOST=localhost  SQLISH_TEST_MARIADB_USER=root      SQLISH_TEST_MARIADB_PASSWORD=root      SQLISH_TEST_MARIADB_DATABASE=test \
SQLISH_TEST_MSSQL_HOST=localhost    SQLISH_TEST_MSSQL_USER=sa          SQLISH_TEST_MSSQL_PASSWORD='Str0ng!Passw0rd' SQLISH_TEST_MSSQL_DATABASE=master \
go test ./...
```

## Stack

Go 1.22+, bubbletea + lipgloss + bubbles (Charmbracelet), drivers:
`mattn/go-sqlite3`, `jackc/pgx/v5`, `go-sql-driver/mysql`, `microsoft/go-mssqldb`.

## Documentación del diseño

La especificación vive como documentos numerados en este directorio:

| Archivo | Contenido |
|---|---|
| `00-LEEME.md` | Idea central (5 motores) y orden de lectura |
| `01-vision.md` | Visión, principios no negociables |
| `02-arquitectura.md` | Capas, interfaz `Store`, dialectos por motor |
| `03-ux-pantallas.md` | Pantallas, keymaps, estilos |
| `04-implementacion.md` | Fases de implementación |
| `05-decisiones-tecnicas.md` | DSNs, edge cases, dialectos |
| `LESSONS.md` | Decisiones del agente durante el desarrollo |

Para la guía de uso paso a paso, ver [USAGE.md](USAGE.md).
