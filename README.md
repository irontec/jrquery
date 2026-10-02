# jrquery

Consulta tickets de Jira desde la terminal. Una sola orden, salida en tabla, enlaces
clicables y una sintaxis pensada para escribir poco: no hace falta saberse las claves
de proyecto, los `accountId` ni el nombre exacto de los estados.

```
$ j
╭────────────┬────────┬───────────┬─────────────┬─────────────────┬────────────┬──────────────────────────────╮
│ Key        │  Type  │ Priority  │ Status      │ Assignee        │ Updated    │ Summary                      │
├────────────┼────────┼───────────┼─────────────┼─────────────────┼────────────┼──────────────────────────────┤
│ PROJ-1042  │ Incid. │ 🟠 High   │ En curso    │ Ana García      │ 2026-09-18 │ Falla la copia de seguridad… │
│ PROJ-1039  │ Tarea  │ 🟡 Medium │ Nueva       │ Ana García      │ 2026-09-17 │ Actualizar documentación     │
╰────────────┴────────┴───────────┴─────────────┴─────────────────┴────────────┴──────────────────────────────╯

Showing 2 of 2 issue(s)
```

---

## Índice

- [Instalación](#instalación)
- [Configuración](#configuración)
- [Uso rápido](#uso-rápido)
- [Buscar por texto](#buscar-por-texto)
- [Filtrar por persona](#filtrar-por-persona)
- [Filtrar por proyecto](#filtrar-por-proyecto)
- [Filtrar por estado](#filtrar-por-estado)
- [Filtrar por tipo de incidencia](#filtrar-por-tipo-de-incidencia)
- [Rangos de fechas](#rangos-de-fechas)
- [El preset «activo»](#el-preset-activo)
- [Abrir incidencias en el navegador](#abrir-incidencias-en-el-navegador)
- [Ordenación](#ordenación)
- [Límite y recuento](#límite-y-recuento)
- [Horas imputadas](#horas-imputadas)
- [JQL propio y filtros guardados](#jql-propio-y-filtros-guardados)
- [Comandos de descubrimiento](#comandos-de-descubrimiento)
- [Recetario](#recetario)
- [Referencia completa de opciones](#referencia-completa-de-opciones)
- [Resolución de problemas](#resolución-de-problemas)
- [Contribuir](#contribuir)
- [Licencia](#licencia)

---

## Instalación

Requiere Python 3.8 o superior y dos dependencias:

```bash
pip install requests rich
```

Descarga la última versión desde la [página de Releases](https://github.com/irontec/jrquery/releases),
dale permisos de ejecución y muévela a tu directorio de binarios:

```bash
chmod +x jrquery.py
sudo mv jrquery.py /usr/local/bin/jrquery
```

### El alias `j`

Toda la documentación usa `j` en lugar de `jrquery`, porque esta herramienta se teclea
muchas veces al día. Para tenerlo:

```bash
sudo ln -s /usr/local/bin/jrquery /usr/local/bin/j
```

O, si prefieres no tocar `/usr/local/bin`, añade a tu `~/.zshrc` o `~/.bashrc`:

```bash
alias j=jrquery
```

Si no quieres ni una cosa ni otra, sustituye `j` por `jrquery` en cualquier ejemplo.

### La versión anterior en Go

Hasta la v0.0.4, jrquery era un binario escrito en Go. Esa versión sigue en la rama
[`legacy`](https://github.com/irontec/jrquery/tree/legacy) y sus binarios en las
releases `v0.0.x`. La versión actual es un único script de Python que la sustituye:
instálalo en la misma ruta (`/usr/local/bin/jrquery`) y reemplazará al binario anterior.
Las opciones no son las mismas; consulta `jrquery -h`.

## Configuración

La primera ejecución pregunta los datos y los guarda en `~/.jrquery.json` con permisos
`600`:

```
$ j
⚙  jrquery — First Time Setup
Your credentials will be stored in ~/.jrquery.json

Jira Base URL (e.g. https://yourcompany.atlassian.net): https://yourcompany.atlassian.net
Jira Login Email: tu.email@example.com
Jira API Token: ************************
```

El token se genera en <https://id.atlassian.com/manage-profile/security/api-tokens>.
Para cambiar credenciales: `j --reconfigure`.

### Personalizar el preset «activo»

`~/.jrquery.json` admite claves opcionales que definen qué considera «activo» tu
equipo. Si no están, el preset deja pasar cualquier tipo en un estado que no sea de
categoría *Done* (`statusCategory != Done`).

- `active_types` y `active_statuses` **sustituyen** a ese filtro por una lista cerrada.
- `extra_inactive_statuses` **resta** estados además: útil para estados como «Blocked»,
  que Jira no considera *Done* pero tú no quieres ver como activos.

Un ejemplo con lista cerrada:

```json
{
  "base_url": "https://yourcompany.atlassian.net",
  "email": "tu.email@example.com",
  "token": "...",

  "active_types": [
    "Tarea", "Incidencia", "Error"
  ],
  "active_statuses": [
    "Nueva", "En curso", "Reabierta"
  ]
}
```

O, sin cerrar la lista, excluyendo solo algunos estados:

```json
{
  "extra_inactive_statuses": ["Blocked"]
}
```

### Qué entiende `-O` por «sin resolver»

Por defecto `-O` filtra por `resolution = Unresolved`. Con este toggle filtra por categoría
de estado (`statusCategory != Done`), útil cuando el workflow cierra incidencias sin
rellenar la resolución:

```json
{
  "unresolved_by_category": true
}
```

Admite `true`/`false`, `1`/`0`, `"enable"`/`"disable"` y `"on"`/`"off"`.

`j --reconfigure` solo cambia las credenciales: el resto de claves se conserva.

## Uso rápido

```bash
j                              # mis incidencias activas
j -u ana                       # las activas de ana
j -p demo                      # todo el proyecto demo
j copia seguridad              # busca ese texto
j PROJ-1042                    # muestra esa incidencia
j -o PROJ-1042                 # la abre en el navegador
j -d                           # añade -d a cualquier orden para ver el JQL generado
```

`-d` es tu mejor amigo mientras aprendes: imprime el JQL exacto y las llamadas a la API.

```
$ j -u -d
JQL: assignee in (currentUser()) AND statusCategory != Done ORDER BY updated DESC
```

## Buscar por texto

Hay dos modos, y la diferencia importa:

| Orden | Qué busca |
|---|---|
| `j -s copia seguridad` | Incidencias que contengan **copia** o **seguridad** (palabras sueltas) |
| `j -ss "copia de seguridad"` | Incidencias que contengan **la frase exacta** «copia de seguridad» |

```bash
j -s error certificado            # cualquiera de las dos palabras
j -ss "error de certificado"      # la línea completa, en ese orden
```

Las palabras sueltas que escribas sin ningún flag se tratan como `-s`:

```bash
j copia seguridad                 # idéntico a: j -s "copia seguridad"
```

La búsqueda cubre resumen, descripción y comentarios (campo `text` de JQL).

### Claves de incidencia sueltas

Si un argumento tiene pinta de clave (`ABC-123`), se busca esa incidencia en vez de
tratarlo como texto. Puedes mezclar varias:

```bash
j PROJ-1042 PROJ-1039 PRO-77
```

## Filtrar por persona

No hace falta el nombre completo ni el `accountId`: escribe un trozo y jrquery lo resuelve.

```bash
j -u                          # tú
j -u ana                      # busca 'ana' entre los usuarios
j -u ana.garcia@example.com
j -R ana                      # por reportador en vez de por asignado
j -R                          # las que has reportado tú
```

Si hay varias coincidencias, te las lista y eliges:

```
$ j -u ana
Se encontraron 3 usuarios que coinciden con 'ana':

  1  Ana García                      ana.garcia@example.com
  2  Ana López                       ana.lopez@example.com
  3  Juliana Ruiz                    juliana.ruiz@example.com

Elige un número (1):
```

Combinado con `-p`, la búsqueda se restringe a quien puede recibir asignaciones en ese
proyecto, lo que deja fuera a las cuentas de clientes del portal:

```bash
j -p demo -u ana              # solo el equipo de demo
```

Alias válidos para «yo»: `me`, `@me`, `mi`, `yo`, `self`, `.`

## Filtrar por proyecto

Por clave o por un trozo del nombre, sin distinguir mayúsculas ni tildes:

```bash
j -p PROJ
j -p demo
j -p sopor                    # casa con 'Soporte'
```

Con varias coincidencias se muestra el mismo menú de selección que con usuarios.

## Filtrar por estado

`-e` no necesita el nombre exacto. Se descarga el catálogo real de estados de tu Jira y
se resuelve por coincidencia, **ignorando mayúsculas y tildes**, con esta prioridad:
exacto → por prefijo → por subcadena.

```bash
j -e curso                    # → status = "En curso"
j -e "in prog"                # → status = "In Progress"
j -e nueva,reabierta          # → status in ("Nueva", "Reabierta")
j -e nueva -e reabierta       # idéntico al anterior
```

### Atajos por categoría

Los tres atajos con `@` usan la *statusCategory* de Jira, así que funcionan sea cual sea
el idioma del workflow:

| Atajo | Equivale a |
|---|---|
| `@todo`, `@new`, `@nuevo`, `@pendiente`, `@abierto`, `@open` | todos los estados de categoría **To Do** |
| `@prog`, `@progress`, `@curso`, `@doing`, `@wip`, `@activo` | todos los de **In Progress** |
| `@done`, `@hecho`, `@cerrado`, `@closed`, `@resuelto`, `@fin` | todos los de **Done** |

```bash
j -e @done -p demo            # todo lo cerrado del proyecto, se llame como se llame
j -e @prog -u ana
```

Si lo que escribes no casa con nada, jrquery te lista los estados disponibles en lugar de
devolver cero resultados en silencio. Para consultarlos a mano: `j --list-statuses`.

Relacionado: `-O/--unresolved` añade `resolution = Unresolved`, que es cosa distinta del
estado (una incidencia puede estar «Cerrada» y sin resolución). Si tu workflow no rellena
la resolución, pon `"unresolved_by_category": true` en `~/.jrquery.json` y `-O` pasará a
filtrar por `statusCategory != Done`.

## Filtrar por tipo de incidencia

Mismo mecanismo difuso que los estados:

```bash
j -t incid                    # → issuetype = "Incidencia"
j -t tare                     # → issuetype = "Tarea"
j -t tarea,error              # varios de una vez
```

Para ver los que existen: `j --list-types`.

## Rangos de fechas

```bash
j --from 2026-01-01 --to 2026-03-31
j --from 7d                   # última semana
j --from 3m --to 1m           # entre hace 3 meses y hace 1 mes
j -r 7                        # atajo de --from 7d
```

**Formatos aceptados** en `--from` y `--to`:

| Formato | Ejemplo |
|---|---|
| ISO | `2026-01-31` |
| Europeo | `31/01/2026`, `31-01-2026` |
| Mes o año sueltos | `2026-01`, `2026` |
| Relativo | `7d` (días), `2w` (semanas), `3m` (meses), `1y` (años) |
| Solo número | `7` → se interpreta como `7d` |

**Sobre qué campo** se aplica el rango lo decide `--date-field`:

| Valor | Campo JQL |
|---|---|
| `updated` (por defecto) | `updated` |
| `created` | `created` |
| `resolved` | `resolutiondate` |
| `due` | `duedate` |

```bash
j -p demo --from 2026-01-01 --to 2026-03-31 --date-field created
```

> Una fecha absoluta en `--to` se amplía internamente a las `23:59` de ese día. Sin eso,
> Jira la interpretaría como las 00:00 y perderías el último día del rango.

## El preset «activo»

Para que `j` a secas sea útil sin escribir nada más, jrquery aplica un preset que filtra
por lo que tu equipo considera «en curso». Sin configurar es:

```sql
statusCategory != Done
```

Con `active_types` y `active_statuses` en `~/.jrquery.json` (ver
[Configuración](#personalizar-el-preset-activo)) pasa a ser:

```sql
issuetype in ("Tarea", "Incidencia", "Error") AND status in ("Nueva", "En curso", "Reabierta")
```

Con `extra_inactive_statuses` (y sin las otras dos claves) queda:

```sql
statusCategory != Done AND status not in ("Blocked")
```

**Cuándo se aplica:**

| Situación | ¿Preset? |
|---|---|
| `j` sin argumentos | Sí |
| `j -u`, `j -u ana` | Sí |
| `-a` / `--active` | Sí, explícito — combinable con cualquier otro filtro |
| `-A` / `--all` | **No**, lo desactiva por completo |
| Pasas tu propio `-e` | Sustituye la parte de estados (`extra_inactive_statuses` incluido) |
| Pasas tu propio `-t` | Sustituye la parte de tipos |
| `-O` / `--unresolved` | **No** se aplica implícitamente: «sin resolver» no es «activo». Usa `-a -O` si quieres los dos |
| `--from` / `--to` / `-r` | **No** se aplica implícitamente: una consulta por fechas es histórica. Usa `-a` si lo quieres |

```bash
j -u ana                      # solo lo activo de ana
j -u ana -A                   # absolutamente todo lo suyo
j -a -p demo                  # lo activo de demo
j -a -u ana --from 3m         # lo activo de ana en los últimos 3 meses
```

## Abrir incidencias en el navegador

j -o PROJ-1042                      # una
j -o PROJ-1042 PROJ-1039 PRO-77      # varias pestañas de golpe
j -p PROJ -o 1042 1039             # solo el número: completa la clave del proyecto
j -p PROJ -o 1042 1039              # solo el número: completa la clave del proyecto
```

Y el caso más cómodo: **`-o` sin claves abre la búsqueda entera** en el navegador de
incidencias de Jira, con el JQL ya montado. Construye la consulta en la terminal y
ábrela cuando te convenza:

```bash
j -u ana -e @prog                  # miras el resultado en la tabla
j -u ana -e @prog -o               # la misma consulta, ahora en Jira
```

Las claves en la tabla también son enlaces: si tu terminal lo soporta, Ctrl+clic abre el
ticket.

## Ordenación

```bash
j -T                          # por última actualización, más reciente primero (por defecto)
j -TT                         # más antigua primero
j -U                          # por asignado A→Z
j -UU                         # Z→A
j -U -T                       # por asignado, y dentro de cada uno por fecha
```

`-U` y `-T` se combinan: el resultado es `ORDER BY assignee ASC, updated DESC`.

## Límite y recuento

```bash
j -l 100                      # hasta 100 resultados (por defecto 50)
j -c                          # solo el número, sin traer las incidencias
j -u ana -e @prog -c          # ¿cuántas tiene en curso?
```

`-c` usa el endpoint de recuento de Jira, así que el número es el real y no está topado
por `-l`.

## Horas imputadas

`-w` suma las horas que has imputado (worklogs de Jira) y las desglosa por ticket y día:

```bash
j -w                          # esta semana, de lunes a hoy
j -w pasada                   # la semana pasada completa
j -w mes                      # este mes
j -w hoy                      # también: ayer, 2026-09-29, 29/09/2026, 3d
j -w --from 2026-09-01 --to 2026-09-30
j -w -u ana -p demo           # las de otra persona, solo en un proyecto
```

```
                     Horas imputadas · 29/09 – 01/10/2026
╭────────────┬──────────────────────────────┬───────┬───────┬───────┬───────╮
│ Key        │ Summary                      │ Lu 29 │ Ma 30 │ Mi 01 │ Total │
├────────────┼──────────────────────────────┼───────┼───────┼───────┼───────┤
│ PROJ-1042  │ Falla la copia de seguridad… │    5h │ 3h30  │       │ 8h30  │
│ PROJ-1039  │ Actualizar documentación     │ 2h30  │ 4h30  │    6h │   13h │
├────────────┼──────────────────────────────┼───────┼───────┼───────┼───────┤
│ Total      │                              │ 7h30  │    8h │    6h │ 21h30 │
╰────────────┴──────────────────────────────┴───────┴───────┴───────┴───────╯
```

Solo cuenta los worklogs del usuario elegido, aunque otros también hayan imputado en el
mismo ticket. Con rangos de más de 7 días se omiten las columnas por día. Funciona con
cualquier herramienta que guarde las horas como worklogs nativos de Jira (Clockwork,
el registro de trabajo de Jira…); un temporizador que siga en marcha aún no cuenta.

## JQL propio y filtros guardados

Cuando necesitas algo que los flags no cubren:

```bash
j -q 'project = PROJ AND labels = urgente AND created >= -14d ORDER BY priority DESC'
```

Y para reutilizar los filtros que ya tienes en Jira:

```bash
j --list-filters              # ver IDs
j -f 10234                    # ejecutar uno
j -f 10234 -o                 # abrirlo en el navegador
```

Con `-q` y `-f`, el resto de filtros se ignoran: mandan ellos.

## Comandos de descubrimiento

```bash
j --list-projects             # clave, nombre y tipo de cada proyecto visible
j --list-users                # nombre y email
j --list-statuses             # estados, su categoría y el atajo @ correspondiente
j --list-types                # tipos de incidencia
j --list-filters              # filtros guardados, con su ID
```

## Recetario

**Mi día a día**
```bash
j                                         # qué tengo encima ahora mismo
j -TT                                     # empezando por lo más olvidado
j -c                                      # cuántas son
```

**Seguimiento de una persona**
```bash
j -u ana                                  # lo activo
j -u ana -A -l 200                        # todo su histórico
j -u ana -e @done --from 1m               # qué cerró el último mes
```

**Auditoría de un proyecto en un trimestre**
```bash
j -p demo --from 2026-01-01 --to 2026-03-31 -l 200
j -p demo -t incid --from 2026-01-01 --to 2026-03-31 -e @done
j -p demo --from 2026-01-01 --to 2026-03-31 --date-field created -c
```

**Tipo + persona + fechas + estado, todo junto**
```bash
j -t incid -u ana -e curso --from 2026-01-01 --to 2026-03-31
j -t tarea -p demo -e @done --from 3m --date-field created
```

**Rastrear un tema concreto**
```bash
j -ss "copia de seguridad" -A -l 100      # la frase exacta, en cualquier estado
j -ss "copia de seguridad" -A -o          # y abrir la búsqueda en Jira
j -s certificado ssl -p demo              # cualquiera de las dos palabras
```

**Trabajo pendiente del sprint**
```bash
j -S -e @prog                             # sprint activo, en curso
j -S -p demo -U                           # sprint activo agrupado por persona
```

**Lo que se me escapa**
```bash
j -O -r 30                                # sin resolver y tocado el último mes
j -e @todo -u ana -c                      # cuánto tiene sin empezar
```

## Referencia completa de opciones

### Selección

| Opción | Descripción |
|---|---|
| `TERM…` | Texto libre a buscar y/o claves de incidencia (`PROJ-123`) |
| `-u, --user [USER]` | Asignado, por nombre parcial. Sin valor = tú |
| `-R, --reporter [USER]` | Reportador, por nombre parcial. Sin valor = tú |
| `-p, --project PROJECT` | Clave o nombre parcial del proyecto |
| `-s, --search TEXT` | Busca las palabras sueltas |
| `-ss, --search-exact TEXT` | Busca la frase exacta |
| `-t, --type TYPE` | Tipo de incidencia, por trozo de nombre. Repetible o con comas |
| `-e, --status STATUS` | Estado, por trozo de nombre o `@todo`/`@prog`/`@done`. Repetible o con comas |
| `-L, --label LABEL` | Etiqueta. Repetible o con comas |
| `-S, --sprint` | Solo el sprint activo |
| `-O, --unresolved` | Solo sin resolver (`resolution = Unresolved`, o `statusCategory != Done` con `unresolved_by_category`) |
| `-a, --active` | Aplica el preset de tipos y estados activos |
| `-A, --all` | Sin filtros de tipo ni estado. Anula `-a`, `-e` y `-O` |

### Fechas

| Opción | Descripción |
|---|---|
| `--from FECHA` | Desde. `2026-01-31`, `31/01/2026`, `2026-01`, `7d`, `2w`, `3m` |
| `--to FECHA` | Hasta, mismo formato |
| `--date-field CAMPO` | `updated` (por defecto), `created`, `resolved`, `due` |
| `-r, --recent DÍAS` | Atajo de `--from Ndías` |

### Salida

| Opción | Descripción |
|---|---|
| `-l, --limit N` | Máximo de resultados (por defecto 50) |
| `-c, --count` | Solo el número de incidencias |
| `-T, --order-by-time` | Ordena por última actualización. `-TT` invierte |
| `-U, --order-by-user` | Ordena por asignado. `-UU` invierte |
| `-o, --open [ISSUE…]` | Abre incidencias en el navegador. Sin claves, abre la búsqueda actual |
| `-d, --debug` | Muestra el JQL y las llamadas a la API |
| `-w, --worklog [CUÁNDO]` | Horas imputadas. Sin valor = esta semana. `pasada`, `mes`, `hoy`, `ayer`, una fecha o `--from`/`--to` |

### Consultas crudas

| Opción | Descripción |
|---|---|
| `-q, --query JQL` | Ejecuta un JQL propio |
| `-f, --filter FILTER_ID` | Ejecuta un filtro guardado de Jira |

### Listados y configuración

| Opción | Descripción |
|---|---|
| `--list-projects` | Proyectos visibles |
| `--list-users` | Usuarios |
| `--list-statuses` | Estados, categoría y atajo `@` |
| `--list-types` | Tipos de incidencia |
| `--list-filters` | Filtros guardados, con su ID |
| `--reconfigure` | Reinicia las credenciales |

## Resolución de problemas

**No encuentro un estado o un tipo que sé que existe**

Mira el catálogo real con `j --list-statuses` o `j --list-types`. jrquery lee los nombres
de tu instancia; si un estado solo existe en el workflow de un proyecto concreto, aparece
igualmente en la lista global.

**`-u nombre` no da resultados pero la persona existe**

Sin `-p`, la búsqueda recorre toda la instancia y puede filtrar cuentas que no son
internas. Prueba acotando: `j -p PROYECTO -u nombre`.

**Una búsqueda de texto devuelve de más**

Estás usando `-s`, que casa palabras sueltas. Usa `-ss "la frase entera"`.

**Una búsqueda de texto devuelve de menos**

Al revés: `-ss` exige la frase literal. Prueba con `-s`.

**El recuento no cuadra con lo que veo**

La tabla muestra hasta `-l` resultados (50 por defecto); el pie indica
`Showing N of TOTAL`. Sube el límite con `-l 200`.

**Errores de la API**

Añade `-d`: además del JQL verás el método y la URL de cada llamada. Los errores HTTP
incluyen el mensaje que devuelve Jira, no solo el código.

## Contribuir

Buena parte del código de esta herramienta la han escrito Claude y JLC, así que lo más
probable es que haya mucho que mejorar. No dudes en abrir un pull request con cualquier
sugerencia.

## Licencia

```
jrquery - Jira Issues query tool
Copyright (C) 2026 Irontec S.L.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

In addition, as a special exception, the copyright holders give
permission to link the code of portions of this program with the
OpenSSL library under certain conditions as described in each
individual source file, and distribute linked combinations
including the two.
You must obey the GNU General Public License in all respects
for all of the code used other than OpenSSL.  If you modify
file(s) with this exception, you may extend this exception to your
version of the file(s), but you are not obligated to do so.  If you
do not wish to do so, delete this exception statement from your
version.  If you delete this exception statement from all source
files in the program, then also delete it here.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
```
