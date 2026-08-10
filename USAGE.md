# USAGE — primer uso paso a paso

`relm` **no levanta servidores ni crea archivos por su cuenta**. Se conecta a
bases que **ya existen y están corriendo** — vos traés la base, `relm` te abre
una ventana para mirarla y consultarla.

- **SQLite** = un archivo `.db` en tu disco (no hay servidor). Lo creás vos con `sqlite3` (sección 1a).
- **PostgreSQL / MySQL / MariaDB / SQL Server** = servidores que ya deben estar corriendo (en tu máquina, en docker, o en la nube). El repo trae un `compose.yaml` para levantarlos de prueba (sección 2a).

No hay configuración previa ni instalación de servidores por parte de `relm`.
Solo necesitás el binario y acceso a la base.

---

## 0. Compilar el binario

```bash
go build -o relm ./cmd/relm
./relm
```

(Requiere Go 1.22+ y `gcc`. Sin gcc, ver la alternativa `modernc.org/sqlite` en `05-decisiones-tecnicas.md`.)

---

## 1. Primera prueba rápida con SQLite (no necesitás ningún servidor)

### 1a. Crear una base de prueba

`relm` no crea archivos: si le pasás un path que no existe, te muestra `no such file`.
Creá una base vos. Dos opciones:

**Con `make demo`** (crea `demo.db` con tablas de ejemplo y datos):

```bash
make demo && ./bin/relm
```

**A mano con el CLI de sqlite:**

```bash
sqlite3 test.db "
CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT);
CREATE TABLE orders (id INTEGER PRIMARY KEY, total REAL, user_id INTEGER);
INSERT INTO users (name, email) VALUES ('Alice','alice@test.com'), ('Bob','bob@test.com');
"
```

### 1b. Conectar

1. Ejecutá `./relm`. Se abre la **pantalla de conexión**:

```
┌─────────────────────────────────────────────┐
│ relm · sin conexión · — ·                 │
├──────────────┬──────────────────────────────┤
│ Conectar     │ Motor  sqlite  ←→ cambiar    │
│              │ Archivo  /data/app.db        │
│ Guardadas    │                              │
│              │ [ Conectar (enter) ]         │
│              │ ctrl+s guardar · r limpiar   │
├──────────────┴──────────────────────────────┤
│ ↑↓ guardadas · tab motor/campos · ←→ motor · enter conectar │
└─────────────────────────────────────────────┘
```

2. Con `Tab` movete al campo `Archivo`, escribí `test.db` (o el path completo) y presioná `Enter`.

3. Ya estás en el **browser**. El sidebar lista las tablas y la primera alfabéticamente (`orders`) queda seleccionada — en la base de ejemplo está vacía:

```
│ > orders    │ tabla vacía
│   users     │
```

Presioná `↓` para bajar a `users` y `Enter` para seleccionarla:

```
│   orders    │ id  name        email          │
│ > users     │ 1   Alice       alice@test.com │
│             │ 2   Bob         bob@test.com   │
```

- `↑↓` / `j k` navegan las filas, `PgUp/PgDn` cambian de página.
- `i` muestra la estructura de la tabla activa (columnas, constraints, índices).
- `r` recarga la tabla.

### 1c. Correr queries

1. Presioná `Tab` para ir al **editor SQL**.
2. Escribí un query, por ejemplo `SELECT * FROM users WHERE id > 1;` y presioná `Ctrl+R`.
3. El resultado aparece abajo, con las columnas como header:

```
│ SELECT * FROM users WHERE id > 1          │
│ ─────────────────────────────────────     │
│ id  name  email                           │
│ 2   Bob   bob@test.com                    │
```

- `INSERT/UPDATE/DELETE` muestran `N filas afectadas`.
- Si el query tiene un error de SQL, se muestra el mensaje en rojo, sin crashear.
- `↑`/`↓` con el input vacío recorren el historial de los últimos 100 queries.
- `Ctrl+L` limpia el input.
- `Tab` vuelve al browser.

---

## 2. Conectarse a PostgreSQL / MySQL / MariaDB / SQL Server

### 2a. Si no tenés un servidor: levantá los 4 con un solo comando

El repo incluye un `compose.yaml` que levanta los cuatro motores a la vez,
con credenciales fijas y la base `test` **auto-creada en el primer arranque**
(no hace falta crear bases a mano):

```bash
docker compose up -d
```

Esperá unos segundos a que estén `healthy`:

```bash
docker compose ps        # todos deben decir healthy
```

Para detenerlos:

```bash
docker compose down      # detiene, conserva los datos
docker compose down -v   # detiene y borra todo
```

**Alternativa sin compose** (un comando por motor, mismos resultados):

```bash
docker run -d --rm --name pg    -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=test -p 5432:5432 postgres:16
docker run -d --rm --name mysql -e MYSQL_ROOT_PASSWORD=root   -e MYSQL_DATABASE=test -p 3306:3306 mysql:8
docker run -d --rm --name maria -e MARIADB_ROOT_PASSWORD=root -e MARIADB_DATABASE=test -p 3307:3306 mariadb:11
docker run -d --rm --name mssql -e ACCEPT_EULA=Y -e MSSQL_SA_PASSWORD='Str0ng!Passw0rd' -p 1433:1433 mcr.microsoft.com/mssql/server:2022-latest
```

> Los env vars `POSTGRES_DB` / `MYSQL_DATABASE` / `MARIADB_DATABASE` hacen que
> cada servidor cree la base `test` sola. SQL Server no la necesita: ya trae `master`.
> Podés crear tablas directamente desde el editor de `relm` (`Ctrl+R`) — no necesitás otro cliente.

### 2b. Conectar desde la pantalla de conexión

1. Con `←`/`→` sobre el selector de **Motor**, elegí el motor (SQLite, PostgreSQL, MySQL, MariaDB, SQL Server).
2. El formulario cambia a `Host · Puerto · Usuario · Password · Base`:

```
│  Motor   [ PostgreSQL ▾ ]                  │
│  ─────────────────────                      │
│  Host     [ localhost ]                     │
│  Puerto   [ 5432 ]                          │
│  Usuario  [ postgres ]                      │
│  Password [ •••••• ]                        │
│  Base     [ test ]                          │
```

3. Completá con los datos del servidor. Con los contenedores de arriba:

| Motor | Host | Puerto | Usuario | Password | Base |
|---|---|---|---|---|---|
| PostgreSQL | `localhost` | `5432` | `postgres` | `postgres` | `test` |
| MySQL | `localhost` | `3306` | `root` | `root` | `test` |
| MariaDB | `localhost` | `3307` | `root` | `root` | `test` |
| SQL Server | `localhost` | `1433` | `sa` | `Str0ng!Passw0rd` | `master` |

4. `Enter` conecta. Vas al browser con las tablas de esa base.

> En MySQL/MariaDB una base vacía muestra `sin tablas — usá el editor para crear una`.
> Escribí en el editor `CREATE TABLE ...` y `Ctrl+R`, y la tabla aparece en el sidebar tras `r`.

### 2c. Guardar la conexión para la próxima

Con la pantalla de conexión armada, `Ctrl+S` la guarda en
`~/.config/relm/connections.json`. La próxima vez aparece en el panel
`Guardadas` y se conecta con `Enter` sobre ella.

---

## 3. Problemas comunes al primer uso

| Síntoma | Causa | Solución |
|---|---|---|
| `no such file` | El path del `.db` no existe | `relm` no crea archivos. Creá la base con `sqlite3 test.db "..."` (ver sección 1a) |
| `connection refused` | No hay servidor en ese host/puerto | Levantá el contenedor docker (sección 2a) o revisá host/puerto |
| `password authentication failed` / `Access denied` | Credenciales incorrectas | Revisá usuario/password. Con docker: `postgres/postgres`, `root/root`, `sa/Str0ng!Passw0rd` |
| `Unknown database` / `database "x" does not exist` | La base no existe | Levantá los contenedores con los env vars `POSTGRES_DB`/`MYSQL_DATABASE`/`MARIADB_DATABASE` (sección 2a) que crean `test` solas, o creala desde el editor de `relm` con `CREATE DATABASE` + `Ctrl+R` |
| El query da error | Dialecto SQL del motor | `relm` pasa tu SQL tal cual al motor. Escribí SQL del motor al que estás conectado |
| La pantalla se ve cortada | Terminal muy chica | Agrandá la terminal; debajo de ~60 columnas se oculta el sidebar |

---

## 4. Resumen de atajos

| Tecla | Acción |
|---|---|
| `Ctrl+C` / `q` | Salir |
| `Ctrl+N` | Nueva conexión |
| `Ctrl+S` | Guardar conexión |
| `Tab` | Browser ↔ Editor |
| `i` | Estructura de la tabla activa |
| `↑↓` / `j k` | Navegar filas |
| `PgUp` / `PgDn` | Cambiar página |
| `r` | Refrescar |
| `Ctrl+R` | Ejecutar query (editor) |
| `Ctrl+L` | Limpiar editor |
| `↑↓` (editor, input vacío) | Historial de queries |
| `?` | Ayuda |
