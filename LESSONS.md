# LESSONS — lecciones del agente

Este archivo documenta encrucijadas y decisiones que el agente toma durante el desarrollo, para reutilizarlas en el futuro. Se actualiza continuamente.

---

## L-01 — Módulo Go sin repo real: no usar `[usuario]` literal

**Encrucijada:** el SPEC dice `go mod init github.com/[usuario]/relm`, pero `[usuario]` con corchetes no es un path de módulo válido para Go y no hay repo real.

**Decisión:** usar `module relm` (nombre simple). Si el proyecto se publica, es un `sed` de una línea.

**Lección:** cuando el SPEC tenga placeholders no válidos, resolver con la opción más simple y anotarlo acá en vez de preguntar.

---

## L-02 — Stack mínimo primero, features después

**Encrucijada:** el SPEC pide `cobra` como opcional, spinners, etc. Empezar todo de golpe complica el debug.

**Decisión:** implementar por fases del SPEC. Fase 1 = solo `store` + `conn` + `cmd` con salida a stdout. No TUI todavía.

**Lección:** seguir `04-implementacion.md` al pie de la letra. Cada fase tiene criterio de done verificable con `go test`/`go build`. No saltar fases.

---

## L-03 — La interfaz `Store` es el contrato sagrado

**Encrucijada:** el TUI necesita saber si un query devuelve filas o "filas afectadas" para renderizar.

**Decisión:** `Query()` devuelve siempre `*Result` con `Columns`/`Rows`. La UI decide cómo mostrar según `len(Result.Columns)`. `Exec()` es para `rowsAffected`. Nunca importar `database/sql` fuera de `internal/store/**`.

**Lección:** mantener la neutralidad de `Store` es lo que hace posible agregar motores sin tocar la UI. Si un requisito parece exigir lógica de motor afuera, es señal de que la interfaz está mal — arreglar la interfaz.

---

## L-04 — Dependencias con CGO complican el build, tener plan B

**Encrucijada:** `mattn/go-sqlite3` requiere gcc. En esta máquina hay gcc, pero cross-compile va a fallar.

**Decisión:** usar `mattn` ahora (SPEC lo pide). Dejar documentada la alternativa `modernc.org/sqlite` para builds sin CGO.

**Lección:** el costo de CGO se paga en release. Anotar el escape hatch en el README y en `05-decisiones-tecnicas.md`.

---

## L-05 — Motor por paquete + registro en `init()` evita el import cycle

**Encrucijada:** `store` (factory) necesita importar `sqlite`, pero `sqlite` importa `store` por los tipos compartidos (`store.Store`, `store.Column`...) → ciclo de imports. Es inevitable con el patrón del SPEC.

**Decisión:** patrón de registro:
- `store` define la interfaz, tipos, errores y un `Register(driver, constructor)` + `New(cfg)` que busca en el registro.
- Cada motor llama `store.Register(...)` en su `init()`.
- `cmd` importa los motores en blanco (`_ "relm/internal/store/sqlite"`) para que se registren.
- `store` NO importa a ningún motor → sin ciclo.

**Lección:** el "plugin registry con init()" es el patrón Go para evitar ciclos entre un contrato y sus implementaciones. El registro es configuración de compilación, no estado de runtime — no viola "no global state".

---

## L-06 — SQLite crea archivos inexistentes: validar con os.Stat

**Encrucijada:** el SPEC exige "archivo no existe → error", pero el driver SQLite crea el archivo en blanco al abrir un path inexistente.

**Decisión:** `os.Stat(cfg.Path)` antes de abrir; si no existe, `ErrConnection: no such file`. Excepción: `:memory:` y `file::memory:` (bases en memoria para tests) se saltan el check.

**Lección:** los drivers de SQLite tienen comportamientos de auto-creación que contradicen el contrato esperado. Validar en la capa del store, no asumir.

---

## L-07 — Tests del factory van en paquete externo (`package store_test`)

**Encrucijada:** el test del factory de `store` necesita importar `sqlite` (para registrarlo), y `sqlite` importa `store` → ciclo de imports en el test.

**Decisión:** test en `package store_test` (test externo). El paquete externo puede importar paquetes que importan al paquete bajo test, sin crear ciclo.

**Lección:** cuando el patrón de registro genera ciclos, los tests del contrato van en `package <name>_test`. Si algo "no se usa" en el test tras el cambio, es señal de que las llamadas quedaron sin prefijo (`New` → `store.New`).

---

## L-08 — `Result` necesita distinguir tabla de "filas afectadas"

**Encrucijada:** la UI decide cómo mostrar un resultado: tabla si hay columnas, "N filas afectadas" para INSERT/UPDATE/DELETE. La interfaz `Store` no tenía cómo expresar lo segundo.

**Decisión:** agregar `Affected int64` a `store.Result`. Queries de lectura lo setean a `-1`; `Exec` lo setea a `rowsAffected`. La UI decide con `len(Columns) > 0` (tabla) o `Affected >= 0` (filas afectadas).

**Lección:** cuando un requisito de la UI no tiene dónde vivir en el modelo, el modelo está incompleto — no parchear la UI. Un campo nuevo en `Result` es más simple que inspeccionar strings en el TUI.

---

## L-09 — La heurística SELECT es frágil; que el store dicte la verdad

**Encrucijada:** ¿`Query()` o `Exec()` para el buffer del editor? `strings.HasPrefix(SELECT)` falla con `WITH ... SELECT`, `PRAGMA`, `EXPLAIN`.

**Decisión:** el editor usa una lista corta de keywords que devuelven filas (`SELECT, WITH, PRAGMA, SHOW, EXPLAIN, DESCRIBE, VALUES, TABLE`) y documenta que es un hint. El resultado final lo dicta el store (un `Result` con columnas vs `Affected`), no el editor.

**Lección:** toda decisión de presentación se basa en el `Result` del store, nunca en re-parsear SQL en la capa UI.

---

## L-10 — `Ctrl+I` y `Tab` son indistinguibles en la terminal

**Encrucijada:** el SPEC mapeaba `Ctrl+I` a "ver estructura". Al probar la TUI, `Ctrl+I` abría el editor: el handler de `Tab` lo capturaba.

**Decisión:** en la terminal `Ctrl+I` y `Tab` son el mismo byte (0x09, HT). bubbletea los normaliza a `KeyTab` (su `String()` es "tab"), así que un binding `ctrl+i` nunca matchea. Cambié inspección de estructura a `i` (info) y actualicé los docs.

**Lección:** nunca diseñar keymaps con `Ctrl+I` si `Tab` está ocupado. Antes de escribir un binding, verificar que no colisione con otra tecla de control con el mismo código ANSI (`Ctrl+J`=`Enter`, `Ctrl+M`=`Enter`/CR, `Ctrl+H`=`Backspace`, `Ctrl+I`=`Tab`). Esta es una clase de bug que solo aparece probando la TUI de verdad, no en tests unitarios de modelo — por eso vale la pena testear el flujo completo con keys.

---

## L-11 — Testear el flujo completo de la TUI ejecutando los `tea.Cmd`

**Encrucijada:** los tests del Model pasaban mensajes con `m.Update(msg)` pero ignoraban el `tea.Cmd` devuelto. El flujo de conexión nunca conectaba en los tests, aunque funcionaba en runtime.

**Decisión:** un helper `step()` en los tests ejecuta el `cmd` devuelto y realimenta el mensaje que produzca (como hace el programa de bubbletea). Sin esto, los mensajes diferidos (`ConnectMsg`) nunca llegan al modelo en el test.

**Lección:** para testear una arquitectura de mensajes hay que simular el runtime: ejecutar cmds y re-alimentar sus mensajes. Además, los tests del TUI necesitan el import en blanco del motor (`_ "relm/internal/store/sqlite"`) o el registro nunca corre.

---

## L-12 — `tea.BatchMsg` es `[]Cmd` en bubbletea v1.x, no mensajes

**Encrucijada:** el helper de tests que ejecutaba cmds trataba `tea.BatchMsg` como una lista de mensajes y los pasaba a `Update`; el resultado de la query nunca se aplicaba y `loading` quedaba `true`.

**Decisión:** en bubbletea v1.3.x, `BatchMsg` es `[]Cmd` (comandos). El programa los ejecuta uno por uno y entrega cada mensaje resultante a `Update`. El helper de tests ahora ejecuta cada sub-cmd y realimenta su mensaje.

**Lección:** siempre verificar la firma real de `tea.Batch`/`BatchMsg` en la versión instalada. La API cambió entre v0.x y v1.x.

---

## L-13 — El historial se pierde si cada ejecución crea un editor nuevo

**Encrucijada:** al hacer la ejecución del editor asíncrona, cada query corría en un `editor.Editor` nuevo. El historial quedaba con un solo elemento porque cada editor nuevo empezaba vacío.

**Decisión:** compartir el puntero al `History` entre el editor del modelo y el editor de la goroutine (`ed.History = m.editor.History`). La goroutine solo muta el ring buffer; la navegación en la UI se guarda con `!m.loading` para evitar races. Verificado con `go test -race`.

**Lección:** al mover una operación a una goroutine, el estado que debe sobrevivir entre llamadas (historial, contadores) tiene que vivir en un objeto compartido. La copia del struct no comparte slices/pointers salvo que se copie el puntero explícitamente.

---

## L-14 — `Ctrl+I`/`Tab` ya documentado; el patrón se repite (teclas de control ambiguas)

**Encrucijada:** `Ctrl+I` se ligó a "estructura" pero colisionaba con `Tab`.

**Decisión:** usar `i`. Ya documentado en L-10. La lección se extiende: revisar el keymap completo antes de implementar para detectar colisiones ANSI.

**Lección:** cuando un keymap del SPEC es técnicamente inviable, la TUI real es la única forma de detectarlo — probar el flujo completo con keys simuladas, no solo renderizar.

---

## L-15 — SQL Server no soporta expresiones booleanas en SELECT

**Encrucijada:** el query de introspección de columnas de mssql usaba `(c.IS_NULLABLE = 'NO')` directamente en el SELECT. Error `Incorrect syntax near '='`.

**Decisión:** usar `CASE WHEN ... THEN 1 ELSE 0 END` para derivar booleanos en SQL Server. También `ISNULL()` en vez de `COALESCE()` (ambos funcionan, ISNULL es el idiosincrático de T-SQL).

**Lección:** el `information_schema` de SQL Server tiene el mismo aspecto que el de Postgres/MySQL pero la sintaxis T-SQL difiere. Los tests de integración contra contenedores docker reales son la única forma confiable de validar queries de introspección — y son rápidos de correr una vez que docker está arriba.

---

## L-16 — Validar cada motor contra un servidor real, no solo "compila"

**Encrucijada:** los motores compilaban y los dialectos unit-test pasaban, pero solo contra servidores reales (docker) aparecieron bugs reales de sintaxis e introspección (el caso de mssql arriba).

**Decisión:** tests de integración por motor gatillados por env vars (`SQLISH_TEST_<MOTOR>_HOST`, etc.) que ejercitan toda la interfaz `Store`: tablas, columnas, constraints, índices, count, paginación, versión. Se saltan con `t.Skip` si no hay env var, así `go test ./...` siempre pasa.

**Lección:** el criterio de done de la fase 7 ("conectar a cada motor en docker") es el que realmente valida el trabajo. Docker está disponible en este entorno: usar contenedores efímeros (`--rm`) con puertos estándar.

---

## L-17 — CGO limita el cross-compile; `modernc` lo habilita

**Encrucijada:** `CGO_ENABLED=1` cross-compilando a darwin desde linux falla (`clang: unsupported option '-arch'`). El SPEC pedía build multi-plataforma.

**Decisión:** verifiqué que `modernc.org/sqlite` (driver pure-Go, registra "sqlite") compila a `darwin/arm64` con `CGO_ENABLED=0`. Documentado en el Makefile como escape hatch. El binario linux con los 5 drivers pesa 21MB (tope: 40MB).

**Lección:** cuando una dependencia trae CGO, cross-compile no es gratis. Validar la alternativa pure-Go de forma concreta (compilar de verdad) antes de prometerla en los docs.

---

## L-18 — El `EchoMode` de password se aplicó al campo equivocado

**Encrucijada:** en el formulario de conexión, `EchoMode = EchoPassword` se aplicaba por índice `c.fields[2]` (Puerto) en vez de `c.fields[4]` (Password). Bug silencioso: la password no se enmascaraba y el puerto sí.

**Decisión:** corregir el índice y agregar tests de pantalla (`connect_test.go`) que verifican campos visibles por motor y enmascaramiento.

**Lección:** los índices mágicos en slices de campos son frágiles. Un test de pantalla que valida comportamiento visible (cuántos campos, si se enmascara) atrapa esta clase de bug que compila pero funciona mal.

---

## L-19 — Los comandos de la documentación hay que probarlos, no copiarlos

**Encrucijada:** `USAGE.md` tenía `docker exec maria mysql -uroot -proot -e "CREATE DATABASE test"`. Al ejecutarlo: la imagen `mariadb:11` no trae el binario `mysql`, solo los `mariadb-*` (`mariadb`, `mariadb-admin`, ...).

**Decisión:** verificar cada comando del USAGE ejecutándolo de verdad (one-liner de sqlite, 4 `docker run`, credenciales). Corregir usando `mariadb` y, mejor, eliminar el paso manual: los env vars `POSTGRES_DB` / `MYSQL_DATABASE` / `MARIADB_DATABASE` hacen que las imágenes oficiales auto-creen la base en el primer arranque.

**Lección:** un comando que no se ejecuta en la doc puede estar roto y nadie lo nota. Los binarios de los contenedores de MariaDB se llaman `mariadb-*` (no `mysql-*`), y las imágenes oficiales crean bases con env vars — usar eso en vez de pasos `exec` manuales.

---

## L-20 — `docker compose` es la mejor forma de ofrecer DBs de prueba

**Encrucijada:** el usuario preguntó si conviene una imagen docker o un `docker run` para "levantar la base". Cuatro `docker run` sueltos son ruido y propensos a errores.

**Decisión:** `compose.yaml` oficial: `docker compose up -d` levanta los 4 motores con credenciales fijas, base `test` auto-creada, y healthchecks. Se validó: los 4 quedan `healthy` y los tests de integración pasan con esas credenciales. Se dejó la alternativa `docker run` por contenedor (con env vars, sin `exec`).

**Lección:** para "levantá el entorno de prueba", `docker compose` con healthchecks es la opción estándar y la más robusta. Una imagen docker de la TUI misma no aporta (necesita un TTY y no incluiría la base); el compose de las bases es lo que el usuario necesita.

---

## L-21 — Makefile: las líneas de receta multilínea rompen si no llevan tab

**Encrucijada:** `make demo` fallaba con "falta un separador": el SQL multilínea dentro de la receta tenía líneas de continuación indentadas con espacios, no con tab.

**Decisión:** cada línea de receta en un Makefile DEBE empezar con tab. Un string SQL multilínea con indentación propia rompe el parseo. Solución: SQL en una sola línea por receta.

**Lección:** en Makefiles, la indentación de las recetas es tab (no espacios). Para evitar el problema, mantener cada comando en una sola línea.







