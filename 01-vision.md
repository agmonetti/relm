# 01 — Visión y filosofía

## Nombre del proyecto

`relm` — browser de bases de datos para la terminal.

## Frase que lo define

> La base de datos ya está ahí. No necesitás una app. Necesitás una ventana.

## Qué es

`relm` es una herramienta de terminal (TUI) escrita en Go para explorar, consultar y editar bases de datos. Corre completamente en la terminal, sin interfaz gráfica, sin servidor, sin configuración previa.

Soporta exactamente cinco motores — SQLite, PostgreSQL, MySQL, MariaDB y SQL Server. El usuario elige el motor en una pantalla de conexión, completa el formulario (o usa una conexión guardada), y puede navegar tablas, ejecutar SQL arbitrario y ver resultados — todo desde el teclado.

## Qué NO es

- No es un ORM ni una librería.
- No soporta Oracle, DB2, Snowflake, SAP HANA, MongoDB, Redis ni ningún otro motor. Solo los cinco listados, y no hay planes de agregar más.
- No es una GUI con ventana nativa.
- No intenta reemplazar herramientas complejas como DBeaver o TablePlus.
- No tiene modo mouse-first. El teclado es la interfaz principal.

## Inspiración

Inspirado en [DBee](https://github.com/murat-cileli/dbee) pero con identidad propia:

| DBee | relm |
|---|---|
| Múltiples motores | Cinco motores exactos: SQLite, PostgreSQL, MySQL, MariaDB, SQL Server |
| Tabs por motor en la conexión | Un formulario que cambia según el motor elegido |
| Browser principalmente | Browser + editor SQL de primera clase |
| Go + tview | Go + bubbletea + lipgloss |

## Principios de diseño (no negociables)

1. **Cinco motores, uno solo.** SQLite, PostgreSQL, MySQL, MariaDB y SQL Server. Ninguno más. La interfaz `Store` se escribe una vez y cada motor la implementa con su propio dialecto. La UI, el browser y el editor nunca saben a qué motor están hablando.

2. **Una conexión, una sesión.** El usuario se conecta con la pantalla de conexión y trabaja. Para cambiar de base, abre otra terminal o vuelve a conexión con `Ctrl+N`. No hay múltiples sesiones simultáneas en tabs.

3. **Teclado primero.** Toda acción tiene atajo de teclado. El mouse puede funcionar pero nunca es requerido.

4. **Sin abstracción innecesaria.** El usuario ejecuta SQL real, ve resultados reales. No hay "modo visual" que oculte el query.

5. **Liviano.** El binario compilado no debe superar ~40 MB. Sin dependencias externas en runtime. El peso de los cinco drivers es el costo aceptado de un solo binario multi-motor.

6. **Falla ruidosamente.** Si algo sale mal (conexión rechazada, query inválido, credenciales incorrectas, archivo corrupto), el error es visible, claro y accionable. No se silencian errores.

7. **Código legible sobre código clever.** El agente debe priorizar claridad. Funciones cortas, nombres descriptivos, comentarios donde el "por qué" no es obvio.

## Usuario objetivo

Desarrollador que ya vive en la terminal. Usa `vim`/`neovim`, `tmux`, `git` desde CLI. Conoce SQL. No quiere abrir una app gráfica para mirar una tabla de 50 filas, ni aprenderse un cliente distinto por motor.
