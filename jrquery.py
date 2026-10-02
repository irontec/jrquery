#!/usr/bin/env python3
"""
jrquery - A simple CLI tool to display Jira tickets on the terminal
"""

import argparse
import json
import os
import re
import sys
import unicodedata
import webbrowser
from datetime import date, datetime, timedelta
from pathlib import Path
from urllib.parse import quote

try:
    import requests
    from requests.auth import HTTPBasicAuth
except ImportError:
    print("Missing dependency: pip install requests")
    sys.exit(1)

try:
    from rich.console import Console
    from rich.table import Table
    from rich import box
    from rich.text import Text
    from rich.markup import escape
    from rich.prompt import Prompt
    from rich.panel import Panel
except ImportError:
    print("Missing dependency: pip install rich")
    sys.exit(1)

# ─── Config ────────────────────────────────────────────────────────────────────

CONFIG_PATH = Path.home() / ".jrquery.json"
console = Console()

ISSUE_KEY_RE = re.compile(r"^[A-Za-z][A-Za-z0-9_]*-\d+$")
ME_ALIASES   = {"@me", "me", "mi", "yo", "self", "."}

# Preset "activo" (-a, y por defecto con -u). Sin configurar: cualquier tipo en un
# estado que no sea de categoría Done, sea cual sea el idioma del workflow.
# Se afina desde ~/.jrquery.json con las claves "active_types" y "active_statuses"
# (sustituyen a ese filtro) y "extra_inactive_statuses" (estados que se excluyen además).
ACTIVE_FALLBACK_JQL = "statusCategory != Done"

# -O filtra por "resolution = Unresolved". Con "unresolved_by_category" activo en
# ~/.jrquery.json usa la categoría de estado, para workflows que no rellenan la resolución.
UNRESOLVED_JQL          = "resolution = Unresolved"
UNRESOLVED_CATEGORY_JQL = "statusCategory != Done"

# Categorías de estado de Jira, independientes del idioma del workflow
CATEGORY_KEYS = {
    "todo": "new", "new": "new", "nuevo": "new", "nueva": "new",
    "pendiente": "new", "abierto": "new", "open": "new",
    "prog": "indeterminate", "progress": "indeterminate", "curso": "indeterminate",
    "doing": "indeterminate", "wip": "indeterminate", "activo": "indeterminate",
    "done": "done", "hecho": "done", "cerrado": "done", "closed": "done",
    "resuelto": "done", "fin": "done",
}

DATE_FIELDS = {
    "updated":  "updated",
    "created":  "created",
    "resolved": "resolutiondate",
    "due":      "duedate",
}

# Color base por categoría de estado. Jira devuelve la categoría en cada issue,
# así que esto funciona con workflows en cualquier idioma sin mantener listas.
STATUS_CATEGORY_COLORS = {
    "new":           "white",
    "indeterminate": "yellow",
    "done":          "green",
}

# Afinados por nombre concreto, para distinguir dentro de una misma categoría
# (una incidencia bloqueada y una en curso comparten categoría, no color).
# Se comparan sin tildes ni mayúsculas y ganan a la categoría.
STATUS_COLORS = {
    "in review":    "cyan",
    "en revision":  "cyan",
    "revision":     "cyan",
    "reabierta":    "magenta",
    "reopened":     "magenta",
    "blocked":      "red",
    "bloqueada":    "red",
    "bloqueado":    "red",
    "en espera":    "red",
    "on hold":      "red",
    "cancelled":    "dim",
    "canceled":     "dim",
    "cancelada":    "dim",
    "cancelado":    "dim",
    "descartada":   "dim",
    "rechazada":    "dim",
}

# El emoji ya lleva color; esto tiñe también el texto de la prioridad
PRIORITY_STYLES = {
    "🔴": "bold red",
    "🟠": "dark_orange",
    "🟡": "yellow",
    "🔵": "blue",
    "⚪": "dim",
    "•":  "dim",
}

PRIORITY_ICONS = {
    # Castellano (main) e inglés, porque el nombre depende del idioma de la instancia
    "muy alta": "🔴",
    "highest":  "🔴",
    "blocker":  "🔴",
    "alta":     "🟠",
    "high":     "🟠",
    "critical": "🟠",
    "media":    "🟡",
    "medium":   "🟡",
    "major":    "🟡",
    "baja":     "🔵",
    "low":      "🔵",
    "minor":    "🔵",
    "muy baja": "⚪",
    "lowest":   "⚪",
    "trivial":  "⚪",
}


def load_config() -> dict:
    if CONFIG_PATH.exists():
        with open(CONFIG_PATH) as f:
            return json.load(f)
    return {}


def save_config(config: dict):
    # Se crea ya con 600: con open() + chmod habría un instante legible por otros
    fd = os.open(CONFIG_PATH, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    with os.fdopen(fd, "w") as f:
        json.dump(config, f, indent=2)
    os.chmod(CONFIG_PATH, 0o600)   # por si ya existía con otros permisos


def setup_config() -> dict:
    console.print(Panel.fit(
        "[bold cyan]⚙  jrquery — First Time Setup[/bold cyan]\n"
        "Your credentials will be stored in [dim]~/.jrquery.json[/dim]",
        border_style="cyan"
    ))
    base_url = Prompt.ask("\n[bold]Jira Base URL[/bold] [dim](e.g. https://yourcompany.atlassian.net)[/dim]").rstrip("/")
    email    = Prompt.ask("[bold]Jira Login Email[/bold]")
    token    = Prompt.ask("[bold]Jira API Token[/bold]", password=True)

    # Conserva el resto de claves (preset activo, comentarios...)
    config = {**load_config(), "base_url": base_url, "email": email, "token": token}
    save_config(config)
    console.print("\n[green]✔ Config saved![/green]\n")
    return config


def get_config() -> dict:
    config = load_config()
    if not config.get("base_url") or not config.get("email") or not config.get("token"):
        config = setup_config()
    return config


# ─── Texto ─────────────────────────────────────────────────────────────────────

def normalize(text) -> str:
    """Minúsculas y sin tildes, para comparar 'peticion' con 'Petición'."""
    decomposed = unicodedata.normalize("NFD", str(text).lower())
    return "".join(c for c in decomposed if not unicodedata.combining(c))


# ─── JQL helpers ───────────────────────────────────────────────────────────────

def jql_str(value) -> str:
    """Escapa un valor y lo devuelve como literal entrecomillado de JQL."""
    escaped = str(value).replace("\\", "\\\\").replace('"', '\\"')
    return f'"{escaped}"'


def jql_phrase(value) -> str:
    """
    Literal JQL que fuerza búsqueda de frase exacta.
    Lucene interpreta las comillas internas como 'la línea completa',
    así 'copia de seguridad' no devuelve issues que solo tengan 'copia'.
    """
    inner = str(value).replace('"', '\\"')
    return jql_str(f'"{inner}"')


def jql_in(field: str, values: list) -> str:
    if len(values) == 1:
        return f"{field} = {jql_str(values[0])}"
    return f"{field} in ({', '.join(jql_str(v) for v in values)})"


def config_flag(value) -> bool:
    """Interpreta un toggle de la config: true/false, 1/0, enable/disable, on/off, yes/no."""
    if isinstance(value, str):
        return value.strip().lower() in {"true", "1", "enable", "enabled", "on", "yes", "si", "sí"}
    return bool(value)


def split_csv(values) -> list:
    """['a,b', 'c'] → ['a', 'b', 'c']"""
    out = []
    for value in values or []:
        out.extend(part.strip() for part in str(value).split(",") if part.strip())
    return out


def parse_date(value: str, flag: str) -> str:
    """
    Acepta fechas absolutas (2026-01-31, 31/01/2026) y relativas (7, 7d, 2w, 3m).
    Devuelve un literal listo para JQL.
    """
    raw = str(value).strip().lower()

    relative = re.fullmatch(r"-?(\d+)\s*([dwmy]?)", raw)
    if relative:
        return f"-{relative.group(1)}{relative.group(2) or 'd'}"

    for fmt in ("%Y-%m-%d", "%Y/%m/%d", "%d/%m/%Y", "%d-%m-%Y", "%Y-%m", "%Y"):
        try:
            parsed = datetime.strptime(raw, fmt)
        except ValueError:
            continue
        return parsed.strftime("%Y-%m-%d")

    console.print(f"[red]✖ Fecha no reconocida en {flag}: '{escape(str(value))}'[/red]")
    console.print("[dim]Formatos válidos: 2026-01-31, 31/01/2026, 2026-01, 2026, 7d, 2w, 3m[/dim]")
    sys.exit(1)


def http_detail(response) -> str:
    """Extrae el mensaje de error que devuelve Jira en el cuerpo de la respuesta."""
    if response is None:
        return ""
    try:
        data = response.json()
    except ValueError:
        return (response.text or "").strip()[:300]
    if isinstance(data, dict):
        for key in ("errorMessages", "warningMessages"):
            messages = data.get(key) or []
            if messages:
                return " ".join(str(m) for m in messages)
        errors = data.get("errors") or {}
        if errors:
            return "; ".join(f"{k}: {v}" for k, v in errors.items())
    return ""


# ─── Jira API ──────────────────────────────────────────────────────────────────

class JiraClient:
    def __init__(self, config: dict, debug: bool = False):
        self.base_url = config["base_url"]
        # Basic auth lleva el token en cada petición: sin TLS viajaría en claro
        if not self.base_url.lower().startswith("https://"):
            console.print(f"[red]✖ La URL de Jira debe empezar por https:// "
                          f"({escape(self.base_url)}).[/red]")
            console.print("[dim]Corrígela con --reconfigure.[/dim]")
            sys.exit(1)
        self.auth     = HTTPBasicAuth(config["email"], config["token"])
        self.headers  = {"Accept": "application/json"}
        self.debug    = debug
        self.active_types    = config.get("active_types")    or []
        self.active_statuses = config.get("active_statuses") or []
        self.extra_inactive_statuses = config.get("extra_inactive_statuses") or []
        self.unresolved_by_category  = config_flag(config.get("unresolved_by_category", False))
        self._statuses = None
        self._types    = None

    def _request(self, method: str, path: str, params: dict = None,
                 payload: dict = None, soft: bool = False):
        """soft=True devuelve None en lugar de abortar, para permitir fallbacks."""
        url = f"{self.base_url}/rest/api/3/{path}"
        if self.debug:
            console.print(f"[dim]→ {method} {escape(url)} {escape(str(params or payload or ''))}[/dim]")
        try:
            r = requests.request(
                method, url,
                auth=self.auth,
                headers={**self.headers, "Content-Type": "application/json"},
                params=params,
                json=payload,
                timeout=15,
            )
            r.raise_for_status()
            if not r.content:
                return {}
            return r.json()
        except requests.exceptions.ConnectionError:
            if soft:
                return None
            console.print(f"[red]✖ Cannot connect to {escape(self.base_url)}[/red]")
            sys.exit(1)
        except requests.exceptions.HTTPError as e:
            if soft:
                return None
            console.print(f"[red]✖ HTTP Error: {escape(str(e))}[/red]")
            detail = http_detail(e.response)
            if detail:
                console.print(f"[red]  {escape(detail)}[/red]")
            sys.exit(1)
        except ValueError:
            if soft:
                return None
            console.print(f"[red]✖ Respuesta no válida de {escape(url)}[/red]")
            sys.exit(1)

    def _get(self, path: str, params: dict = None, soft: bool = False):
        return self._request("GET", path, params=params, soft=soft)

    def _post(self, path: str, payload: dict = None, soft: bool = False):
        return self._request("POST", path, payload=payload or {}, soft=soft)

    def count(self, jql: str) -> int:
        """
        El endpoint search/jql ya no devuelve 'total'; approximate-count sí.
        Devuelve -1 si la instancia no lo soporta.
        """
        data = self._post("search/approximate-count", {"jql": jql}, soft=True)
        if isinstance(data, dict) and isinstance(data.get("count"), int):
            return data["count"]
        return -1

    def search(self, jql: str, limit: int = 50) -> tuple:
        all_issues = []
        next_page_token = None

        while True:
            remaining = limit - len(all_issues)
            if remaining <= 0:
                break

            payload = {
                "jql":        jql,
                "maxResults": min(remaining, 100),
                "fields":     ["summary", "status", "assignee", "priority",
                               "issuetype", "updated", "resolution"],
            }
            if next_page_token:
                payload["nextPageToken"] = next_page_token

            data = self._post("search/jql", payload)
            issues = data.get("issues", [])
            all_issues.extend(issues)

            next_page_token = data.get("nextPageToken")
            if not next_page_token or not issues:
                break

        total = self.count(jql)
        if total < 0:
            total = len(all_issues)
        return all_issues[:limit], total

    def list_projects(self) -> list:
        return self._get("project")

    def status_catalog(self) -> list:
        """Todos los estados de la instancia, deduplicados por nombre."""
        if self._statuses is None:
            data = self._get("status", soft=True) or []
            seen, out = set(), []
            for s in data:
                name = s.get("name")
                if not name or normalize(name) in seen:
                    continue
                seen.add(normalize(name))
                category = s.get("statusCategory") or {}
                out.append({
                    "name":          name,
                    "category":      category.get("key", ""),
                    "category_name": category.get("name", ""),
                })
            self._statuses = sorted(out, key=lambda x: normalize(x["name"]))
        return self._statuses

    def type_catalog(self) -> list:
        """Todos los tipos de incidencia, deduplicados por nombre."""
        if self._types is None:
            data = self._get("issuetype", soft=True) or []
            seen, out = set(), []
            for t in data:
                name = t.get("name")
                if not name or normalize(name) in seen:
                    continue
                seen.add(normalize(name))
                out.append({
                    "name":     name,
                    "subtask":  bool(t.get("subtask")),
                    "category": "",
                })
            self._types = sorted(out, key=lambda x: normalize(x["name"]))
        return self._types

    def find_project(self, query: str) -> list:
        """Busca proyectos por clave o nombre (fuzzy, case-insensitive)."""
        all_projects = self._get("project")
        q = normalize(query.strip())
        exact, partial = [], []
        for p in all_projects:
            key  = normalize(p.get("key",  ""))
            name = normalize(p.get("name", ""))
            if key == q:
                exact.append(p)
            elif key.startswith(q) or q in key or q in name:
                partial.append(p)
        return exact + partial

    def find_assignable_user(self, query: str, project_key: str = None) -> list:
        """
        Busca usuarios por nombre/email.
        Con proyecto usa assignable/search (solo quien puede recibir asignaciones).
        Sin proyecto cae a user/search filtrando por cuentas internas, porque
        assignable/multiProjectSearch exige projectKeys y devuelve 400 sin ellas.
        """
        if project_key:
            results = self._get("user/assignable/search",
                                {"query": query, "project": project_key, "maxResults": 50},
                                soft=True)
        else:
            results = self._get("user/search",
                                {"query": query, "maxResults": 50},
                                soft=True)

        if not isinstance(results, list):
            return []

        # Solo usuarios activos e internos (fuera clientes del portal y apps)
        candidates = [
            u for u in results
            if u.get("active") and u.get("accountType", "atlassian") == "atlassian"
        ]

        q = normalize(query.strip())
        exact, partial = [], []
        for u in candidates:
            name  = normalize(u.get("displayName") or "")
            email = normalize(u.get("emailAddress") or "")
            if name == q or email == q:
                exact.append(u)
            else:
                partial.append(u)
        return exact + partial

    def myself(self) -> dict:
        return self._get("myself")

    def worklogs(self, key: str, started_after: int, started_before: int) -> list:
        """Worklogs de una incidencia entre dos instantes (ms epoch), paginando."""
        out, start_at = [], 0
        while True:
            data = self._get(f"issue/{key}/worklog", {
                "startedAfter":  started_after,
                "startedBefore": started_before,
                "startAt":       start_at,
                "maxResults":    1000,
            })
            page = data.get("worklogs", [])
            out.extend(page)
            start_at += len(page)
            if not page or start_at >= data.get("total", 0):
                break
        return out

    def list_users(self) -> list:
        return self._get("users/search", {"query": ".", "maxResults": 200})

    def list_filters(self) -> list:
        return self._get("filter/favourite")

    def get_filter(self, filter_id: str) -> dict:
        return self._get(f"filter/{filter_id}")


# ─── Resolvers ─────────────────────────────────────────────────────────────────

def match_catalog(term: str, catalog: list) -> list:
    """Exacto → por prefijo → por subcadena. Sin tildes ni mayúsculas."""
    q = normalize(term.strip())
    if not q:
        return []
    exact = [c["name"] for c in catalog if normalize(c["name"]) == q]
    if exact:
        return exact
    prefix = [c["name"] for c in catalog if normalize(c["name"]).startswith(q)]
    if prefix:
        return prefix
    return [c["name"] for c in catalog if q in normalize(c["name"])]


def resolve_catalog_terms(terms: list, catalog: list, label: str,
                          allow_categories: bool = False, debug: bool = False) -> list:
    """
    Convierte lo que escribe el usuario ('curso', 'incid', '@done')
    en los nombres reales de Jira. Si el catálogo no está disponible,
    devuelve los términos tal cual.
    """
    if not catalog:
        return terms

    resolved, missing = [], []
    for term in terms:
        names = []
        if allow_categories and term.startswith("@"):
            key = CATEGORY_KEYS.get(normalize(term[1:]))
            if key:
                names = [c["name"] for c in catalog if c.get("category") == key]
        else:
            names = match_catalog(term, catalog)

        if names:
            resolved.extend(names)
        else:
            missing.append(term)

    if missing:
        console.print(f"\n[red]✖ No hay ningún {label} que coincida con: "
                      f"[bold]{escape(', '.join(missing))}[/bold][/red]")
        available = escape(", ".join(c["name"] for c in catalog[:40]))
        console.print(f"[dim]Disponibles: {available}"
                      f"{' …' if len(catalog) > 40 else ''}[/dim]\n")
        sys.exit(1)

    # dedup conservando el orden
    resolved = list(dict.fromkeys(resolved))
    if debug or resolved != terms:
        console.print(f"[dim]{label.capitalize()}: {escape(', '.join(resolved))}[/dim]")
    return resolved


def resolve_project(client, query: str) -> str:
    """Resuelve nombre/clave parcial de proyecto → clave exacta."""
    matches = client.find_project(query)

    if not matches:
        console.print(f"\n[red]✖ No se encontró ningún proyecto que coincida con '[bold]{escape(query)}[/bold]'[/red]")
        console.print("[dim]Usa --list-projects para ver los proyectos disponibles.[/dim]\n")
        sys.exit(1)

    if len(matches) == 1:
        p = matches[0]
        console.print(f"[dim]Proyecto: [bold]{escape(p['key'])}[/bold] — {escape(p['name'])}[/dim]")
        return p["key"]

    console.print(f"\n[yellow]Se encontraron {len(matches)} proyectos que coinciden con '[bold]{escape(query)}[/bold]':[/yellow]\n")
    for i, p in enumerate(matches, 1):
        console.print(f"  [bold cyan]{i}[/bold cyan]  [bold]{escape(p['key']):<12}[/bold] {escape(p['name'])}")
    console.print()
    try:
        choice = Prompt.ask("Elige un número", default="1")
        idx = int(choice) - 1
        if 0 <= idx < len(matches):
            return matches[idx]["key"]
    except (ValueError, KeyboardInterrupt):
        pass

    console.print("[red]Selección inválida.[/red]")
    sys.exit(1)


def resolve_user(client, query: str, project_key: str = None) -> str:
    """
    Resuelve nombre parcial de asignado → accountId.
    Busca primero entre asignables del proyecto; si no hay nada, en toda la instancia.
    """
    matches = client.find_assignable_user(query, project_key=project_key)

    if not matches and project_key:
        console.print("[dim]No encontrado en el proyecto, buscando en toda la instancia...[/dim]")
        matches = client.find_assignable_user(query)

    if not matches:
        console.print(f"\n[red]✖ No se encontró ningún usuario que coincida con '[bold]{escape(query)}[/bold]'[/red]")
        console.print("[dim]Usa --list-users para ver los usuarios disponibles.[/dim]\n")
        sys.exit(1)

    if len(matches) == 1:
        u = matches[0]
        console.print(f"[dim]Usuario: [bold]{escape(u.get('displayName') or '')}[/bold] ({escape(u.get('emailAddress') or '')})[/dim]")
        return u["accountId"]

    console.print(f"\n[yellow]Se encontraron {len(matches)} usuarios que coinciden con '[bold]{escape(query)}[/bold]':[/yellow]\n")
    for i, u in enumerate(matches[:10], 1):
        name  = escape(u.get("displayName") or "?")
        email = escape(u.get("emailAddress") or "")
        console.print(f"  [bold cyan]{i}[/bold cyan]  [bold]{name:<30}[/bold] [dim]{email}[/dim]")
    console.print()
    try:
        choice = Prompt.ask("Elige un número", default="1")
        idx = int(choice) - 1
        if 0 <= idx < len(matches[:10]):
            return matches[idx]["accountId"]
    except (ValueError, KeyboardInterrupt):
        pass

    console.print("[red]Selección inválida.[/red]")
    sys.exit(1)


def resolve_assignee(client, value, project_key, field: str) -> str:
    if str(value).strip().lower() in ME_ALIASES:
        return f"{field} in (currentUser())"
    if client is None:
        return f"{field} = {jql_str(value)}"
    account_id = resolve_user(client, value, project_key=project_key)
    return f"{field} = {jql_str(account_id)}"


# ─── JQL Builder ───────────────────────────────────────────────────────────────

def build_order(args) -> str:
    """-U y -T se combinan; antes uno anulaba al otro en silencio."""
    parts = []
    if args.order_by_user:
        parts.append("assignee ASC" if args.order_by_user == 1 else "assignee DESC")
    if args.order_by_time:
        parts.append("updated DESC" if args.order_by_time == 1 else "updated ASC")
    if not parts:
        parts.append("updated DESC")
    return " ORDER BY " + ", ".join(parts)


def build_date_conditions(args) -> list:
    """--from / --to sobre el campo elegido con --date-field."""
    field = DATE_FIELDS[args.date_field]
    conditions = []

    since = args.date_from or (f"{args.recent}d" if args.recent else None)
    if since:
        conditions.append(f"{field} >= {jql_str(parse_date(since, '--from'))}")

    if args.date_to:
        value = parse_date(args.date_to, "--to")
        # Una fecha absoluta se interpreta a las 00:00, así que ampliamos
        # hasta el final del día para que el rango incluya ese día completo.
        if not value.startswith("-"):
            value = f"{value} 23:59"
        conditions.append(f"{field} <= {jql_str(value)}")

    return conditions


def build_jql(args, client=None) -> str:
    conditions = []
    debug = getattr(args, "debug", False)

    # Claves sueltas en la línea de comandos: j PROJ-123 PROJ-456
    if args.keys:
        conditions.append(jql_in("key", args.keys))

    # Proyecto primero — necesitamos la key para buscar usuarios asignables
    project_key = None
    if args.project:
        project_key = resolve_project(client, args.project) if client else args.project
        conditions.append(f"project = {jql_str(project_key)}")

    if args.user is not None:
        conditions.append(resolve_assignee(client, args.user, project_key, "assignee"))
    if args.reporter is not None:
        conditions.append(resolve_assignee(client, args.reporter, project_key, "reporter"))

    if args.search:
        conditions.append(f"text ~ {jql_str(args.search)}")
    if args.search_exact:
        conditions.append(f"text ~ {jql_phrase(args.search_exact)}")

    labels = split_csv(args.label)
    if labels:
        conditions.append(jql_in("labels", labels))

    conditions.extend(build_date_conditions(args))

    if args.sprint:
        conditions.append("sprint in openSprints()")

    # Sin ningún filtro: mis incidencias
    mine_by_default = not conditions and not split_csv(args.type) and not split_csv(args.status)
    if mine_by_default:
        conditions.append("assignee in (currentUser())")

    # Preset "activo": explícito con -a, implícito con -u o sin argumentos.
    # -A lo desactiva; -t y -e sustituyen la mitad que toquen.
    # Una consulta con rango de fechas es histórica: ahí solo cuenta el -a explícito.
    # -O ya pide "sin resolver", que no es lo mismo que "activo": tampoco lo activa solo.
    historic   = bool(args.date_from or args.date_to or args.recent)
    want_active = args.active or (
        not args.all and not historic and not args.unresolved
        and (args.user is not None or mine_by_default)
    )
    if args.all:
        want_active = False

    types = split_csv(args.type)
    if types and client:
        types = resolve_catalog_terms(types, client.type_catalog(), "tipo", debug=debug)
    elif not types and want_active:
        types = client.active_types if client else []
    if types:
        conditions.append(jql_in("issuetype", types))

    if not args.all:
        statuses = split_csv(args.status)
        if statuses and client:
            statuses = resolve_catalog_terms(statuses, client.status_catalog(), "estado",
                                             allow_categories=True, debug=debug)
        elif not statuses and want_active:
            statuses = client.active_statuses if client else []
        if statuses:
            conditions.append(jql_in("status", statuses))
        elif want_active:
            conditions.append(ACTIVE_FALLBACK_JQL)
        # Solo cuando los estados salen del preset: un -e explícito manda
        if want_active and not args.status and client and client.extra_inactive_statuses:
            excluded = client.extra_inactive_statuses
            conditions.append(f"status not in ({', '.join(jql_str(v) for v in excluded)})")

        if args.unresolved:
            by_category = client.unresolved_by_category if client else False
            unresolved  = UNRESOLVED_CATEGORY_JQL if by_category else UNRESOLVED_JQL
            # El preset sin configurar ya puede haber puesto la misma condición
            if unresolved not in conditions:
                conditions.append(unresolved)

    if not conditions:
        conditions.append("assignee in (currentUser())")

    return " AND ".join(conditions) + build_order(args)


def resolve_jql(args, client) -> str:
    if args.query:
        return args.query
    if args.filter:
        f = client.get_filter(args.filter)
        jql = f.get("jql", "")
        console.print(f"[dim]Filtro: {escape(f.get('name') or '')} → {escape(jql)}[/dim]")
        return jql
    return build_jql(args, client)


def split_terms(terms: list) -> tuple:
    """Separa los argumentos sueltos en claves de incidencia y texto libre."""
    keys, words = [], []
    for term in terms or []:
        if ISSUE_KEY_RE.match(term):
            keys.append(term.upper())
        else:
            words.append(term)
    return keys, words


# ─── Display ───────────────────────────────────────────────────────────────────

def status_style(status: str, category: str = "") -> str:
    """El nombre concreto manda; si no está mapeado, decide la categoría."""
    override = STATUS_COLORS.get(normalize(status))
    if override:
        return override
    return STATUS_CATEGORY_COLORS.get(category, "white")


def issue_link(base_url: str, key: str) -> Text:
    """Clave clicable. Text no interpreta marcado: un dato de Jira no puede inyectar enlaces."""
    return Text(key, style=f"link {base_url}/browse/{quote(key)}")


def render_issues(issues: list, total: int, base_url: str, count_only: bool = False):
    if count_only:
        console.print(f"\n[bold cyan]{total}[/bold cyan] issue(s) found.\n")
        return

    if not issues:
        console.print("\n[yellow]No issues found.[/yellow]\n")
        return

    table = Table(
        box=box.ROUNDED,
        show_header=True,
        header_style="bold cyan",
        border_style="dim",
        expand=False,
        show_lines=False,
    )

    table.add_column("Key",        style="bold blue",  no_wrap=True, min_width=10)
    table.add_column("Type",       no_wrap=True,       min_width=6)
    table.add_column("Priority",   no_wrap=True,       min_width=8)
    table.add_column("Status",     no_wrap=True,       min_width=12)
    table.add_column("Assignee",   no_wrap=True,       min_width=15)
    table.add_column("Updated",    no_wrap=True,       min_width=12)
    table.add_column("Summary",    min_width=40)

    for issue in issues:
        f          = issue["fields"]
        key        = issue["key"]
        summary    = f.get("summary", "")
        status_f   = f.get("status") or {}
        status     = status_f.get("name", "?")
        status_cat = (status_f.get("statusCategory") or {}).get("key", "")
        assignee_n = (f.get("assignee") or {}).get("displayName")
        assignee   = Text(assignee_n) if assignee_n else Text("Unassigned", style="dim")
        priority   = (f.get("priority") or {}).get("name", "?")
        issuetype  = (f.get("issuetype") or {}).get("name", "?")
        updated    = (f.get("updated") or "")[:10]

        priority_icon = PRIORITY_ICONS.get(priority.lower(), "•")
        status_color  = status_style(status, status_cat)

        table.add_row(
            issue_link(base_url, key),
            Text(issuetype),
            Text(f"{priority_icon} {priority}",
                 style=PRIORITY_STYLES.get(priority_icon, "")),
            Text(status, style=status_color),
            assignee,
            updated,
            Text(summary),
        )

    console.print()
    console.print(table)
    shown = len(issues)
    console.print(f"\n[dim]Showing {shown} of {total} issue(s)[/dim]\n")


def render_projects(projects: list):
    table = Table(box=box.SIMPLE, header_style="bold cyan", border_style="dim")
    table.add_column("Key",  style="bold blue", no_wrap=True)
    table.add_column("Name")
    table.add_column("Type", style="dim")
    for p in sorted(projects, key=lambda x: x.get("key", "")):
        table.add_row(Text(p.get("key") or ""), Text(p.get("name") or ""), Text(p.get("projectTypeKey") or ""))
    console.print()
    console.print(table)
    console.print(f"\n[dim]{len(projects)} project(s)[/dim]\n")


def render_users(users: list):
    table = Table(box=box.SIMPLE, header_style="bold cyan", border_style="dim")
    table.add_column("Name",  style="bold")
    table.add_column("Email", style="dim")
    for u in users:
        table.add_row(Text(u.get("displayName") or ""), Text(u.get("emailAddress") or ""))
    console.print()
    console.print(table)
    console.print(f"\n[dim]{len(users)} user(s)[/dim]\n")


def render_filters(filters: list):
    table = Table(box=box.SIMPLE, header_style="bold cyan", border_style="dim")
    table.add_column("ID",   style="bold blue", no_wrap=True)
    table.add_column("Name")
    table.add_column("Owner", style="dim")
    for f in filters:
        table.add_row(
            Text(str(f.get("id", ""))),
            Text(f.get("name") or ""),
            Text((f.get("owner") or {}).get("displayName") or ""),
        )
    console.print()
    console.print(table)
    console.print(f"\n[dim]{len(filters)} filter(s)[/dim]\n")


def render_statuses(statuses: list):
    table = Table(box=box.SIMPLE, header_style="bold cyan", border_style="dim")
    table.add_column("Estado", style="bold")
    table.add_column("Categoría", style="dim")
    table.add_column("Atajo", style="dim")
    for s in statuses:
        shortcut = {"new": "@todo", "indeterminate": "@prog", "done": "@done"}.get(s["category"], "")
        table.add_row(Text(s["name"], style=status_style(s["name"], s["category"])), Text(s["category_name"]), shortcut)
    console.print()
    console.print(table)
    console.print(f"\n[dim]{len(statuses)} estado(s) · basta con escribir un trozo: "
                  f"-e curso, -e revision, -e @done[/dim]\n")


def render_types(types: list):
    table = Table(box=box.SIMPLE, header_style="bold cyan", border_style="dim")
    table.add_column("Tipo", style="bold")
    table.add_column("Subtarea", style="dim")
    for t in types:
        table.add_row(Text(t["name"]), "sí" if t["subtask"] else "")
    console.print()
    console.print(table)
    console.print(f"\n[dim]{len(types)} tipo(s) · basta con escribir un trozo: -t incid[/dim]\n")


# ─── Horas imputadas ───────────────────────────────────────────────────────────

WEEKDAYS = ["Lu", "Ma", "Mi", "Ju", "Vi", "Sá", "Do"]


def parse_day(value: str, flag: str) -> date:
    """Como parse_date, pero devuelve un día concreto: hoy, ayer, 3d, 1w, 2026-09-29…"""
    raw   = normalize(value).strip()
    today = date.today()
    if raw in ("hoy", "today"):
        return today
    if raw in ("ayer", "yesterday"):
        return today - timedelta(days=1)

    relative = re.fullmatch(r"-?(\d+)\s*([dw]?)", raw)
    if relative:
        days = int(relative.group(1)) * (7 if relative.group(2) == "w" else 1)
        return today - timedelta(days=days)

    for fmt in ("%Y-%m-%d", "%Y/%m/%d", "%d/%m/%Y", "%d-%m-%Y"):
        try:
            return datetime.strptime(raw, fmt).date()
        except ValueError:
            continue

    console.print(f"[red]✖ Fecha no reconocida en {flag}: '{escape(str(value))}'[/red]")
    console.print("[dim]Válido: semana, pasada, mes, hoy, ayer, 2026-09-29, 29/09/2026, 3d, 1w[/dim]")
    sys.exit(1)


def hours_range(args) -> tuple:
    """-w sin valor = semana actual (lunes → hoy). --from/--to mandan si se dan."""
    today  = date.today()
    monday = today - timedelta(days=today.weekday())

    if args.date_from or args.date_to:
        start = parse_day(args.date_from, "--from") if args.date_from else monday
        end   = parse_day(args.date_to,   "--to")   if args.date_to   else today
    else:
        value = normalize(args.hours).strip()
        if value in ("semana", "week"):
            start, end = monday, today
        elif value in ("pasada", "last", "semana-pasada"):
            start, end = monday - timedelta(days=7), monday - timedelta(days=1)
        elif value in ("mes", "month"):
            start, end = today.replace(day=1), today
        else:
            start = end = parse_day(args.hours, "-w")

    if start > end:
        start, end = end, start
    return start, end


def epoch_ms(day: date) -> int:
    return int(datetime(day.year, day.month, day.day).timestamp() * 1000)


def worklog_day(started: str):
    """'2026-09-29T09:00:00.000+0200' → date, en la zona horaria con que se imputó."""
    try:
        return datetime.strptime(started, "%Y-%m-%dT%H:%M:%S.%f%z").date()
    except (TypeError, ValueError):
        return None


def fmt_hours(seconds: int) -> str:
    if not seconds:
        return ""
    hours, minutes = divmod(round(seconds / 60), 60)
    return f"{hours}h{minutes:02d}" if minutes else f"{hours}h"


def show_hours(args, config, client):
    start, end = hours_range(args)

    project_key = resolve_project(client, args.project) if args.project else None
    if args.user is not None and str(args.user).strip().lower() not in ME_ALIASES:
        account_id = resolve_user(client, args.user, project_key=project_key)
    else:
        account_id = client.myself().get("accountId")

    conditions = [
        f"worklogAuthor = {jql_str(account_id)}",
        f"worklogDate >= {jql_str(start.isoformat())}",
        f"worklogDate <= {jql_str(end.isoformat())}",
    ]
    if project_key:
        conditions.append(f"project = {jql_str(project_key)}")
    jql = " AND ".join(conditions) + " ORDER BY key ASC"
    if args.debug:
        console.print(f"\n[dim]JQL: {escape(jql)}[/dim]\n")

    issues, _ = client.search(jql, limit=1000)

    # Margen de un día a cada lado por las zonas horarias; luego se filtra por fecha exacta
    after  = epoch_ms(start - timedelta(days=1))
    before = epoch_ms(end + timedelta(days=2))

    rows = {}
    for issue in issues:
        for w in client.worklogs(issue["key"], after, before):
            # El worklog de un ticket incluye las horas de los compañeros
            if (w.get("author") or {}).get("accountId") != account_id:
                continue
            day = worklog_day(w.get("started"))
            if day is None or not start <= day <= end:
                continue
            row = rows.setdefault(issue["key"], {
                "summary": issue["fields"].get("summary", ""),
                "days":    {},
            })
            row["days"][day] = row["days"].get(day, 0) + w.get("timeSpentSeconds", 0)

    render_hours(rows, start, end, config["base_url"])


def render_hours(rows: dict, start: date, end: date, base_url: str):
    span = f"{start:%d/%m/%Y}" if start == end else f"{start:%d/%m} – {end:%d/%m/%Y}"

    if not rows:
        console.print(f"\n[yellow]No hay horas imputadas ({span}).[/yellow]\n")
        return

    # Desglose por día solo si cabe (una semana como mucho)
    days = []
    if (end - start).days < 7:
        days = [start + timedelta(days=i) for i in range((end - start).days + 1)]

    table = Table(
        title=f"Horas imputadas · {span}",
        box=box.ROUNDED,
        header_style="bold cyan",
        border_style="dim",
    )
    table.add_column("Key", style="bold blue", no_wrap=True, min_width=10)
    table.add_column("Summary", min_width=30)
    for day in days:
        table.add_column(f"{WEEKDAYS[day.weekday()]} {day:%d}", justify="right", no_wrap=True)
    table.add_column("Total", justify="right", style="bold", no_wrap=True)

    day_totals = {day: 0 for day in days}
    grand_total = 0
    for key, row in rows.items():
        total = sum(row["days"].values())
        grand_total += total
        cells = []
        for day in days:
            secs = row["days"].get(day, 0)
            day_totals[day] += secs
            cells.append(fmt_hours(secs))
        table.add_row(issue_link(base_url, key),
                      Text(row["summary"]), *cells, fmt_hours(total))

    table.add_section()
    table.add_row("[bold]Total[/bold]", "",
                  *(f"[bold]{fmt_hours(day_totals[d]) or '-'}[/bold]" for d in days),
                  f"[bold green]{fmt_hours(grand_total)}[/bold green]")

    console.print()
    console.print(table)
    console.print(f"\n[dim]{len(rows)} incidencia(s)[/dim]\n")


# ─── Browser ───────────────────────────────────────────────────────────────────

def open_issues(args, config, client):
    """-o con claves abre cada incidencia; -o sin claves abre la búsqueda actual."""
    base_url = config["base_url"]

    if args.open:
        project_key = None
        keys = []
        for token in args.open:
            token = token.strip()
            if token.isdigit():
                # j -p PROJ -o 123 456  →  PROJ-123, PROJ-456
                if project_key is None:
                    if not args.project:
                        console.print(f"[red]✖ '{escape(token)}' necesita un proyecto: usa -p PROYECTO.[/red]")
                        sys.exit(1)
                    project_key = resolve_project(client, args.project)
                keys.append(f"{project_key}-{token}")
            else:
                keys.append(token.upper())

        for key in dict.fromkeys(keys):   # dedup conservando el orden
            url = f"{base_url}/browse/{key}"
            console.print(f"[cyan]↗ {escape(url)}[/cyan]")
            webbrowser.open_new_tab(url)
        return

    jql = resolve_jql(args, client)
    url = f"{base_url}/issues/?jql={quote(jql)}"
    console.print(f"[cyan]↗ {escape(url)}[/cyan]")
    webbrowser.open_new_tab(url)


# ─── CLI ───────────────────────────────────────────────────────────────────────

EXAMPLES = """
Ejemplos:
  j                                    Mis incidencias activas
  j -u                                 Idem (asignadas a mí, en estados no cerrados)
  j -u ana                             Las activas de 'ana'
  j -u ana -A                          Todas las suyas, sin filtro de estado
  j -s copia seguridad                 Palabras sueltas
  j -ss "copia de seguridad"           Frase exacta
  j PROJ-123 PROJ-456                  Muestra esas incidencias
  j -o PROJ-123 PROJ-456               Las abre en el navegador
  j -p PROJ -o 123 456                 Idem, completando la clave del proyecto
  j -u ana -o                          Abre la búsqueda entera en el navegador
  j -e curso                           Estado por trozo de nombre ('En curso')
  j -e @done                           Todo lo cerrado, sea cual sea el idioma
  j -e nueva,reabierta                 Varios estados
  j -t incid -u ana --from 2026-01-01 --to 2026-03-31
                                       Tipo + persona + rango de fechas
  j -t tarea -p demo -e @done --from 3m --date-field created
  --list-statuses / --list-types       Qué nombres existen en tu Jira
  j -w                                 Mis horas imputadas esta semana
  j -w pasada                          Las de la semana pasada (también: mes, hoy, ayer)
  j -w 2026-09-29                      Las de un día concreto
  j -w --from 2026-09-01 --to 2026-09-30
                                       Las de un rango
  j -w -u ana -p demo                  Las de 'ana' en un proyecto
"""


def main():
    parser = argparse.ArgumentParser(
        prog="jrquery",
        description="Simple CLI tool to display Jira tickets on the terminal",
        epilog=EXAMPLES,
        formatter_class=argparse.RawTextHelpFormatter,
    )

    parser.add_argument("-w", "--worklog", dest="hours", metavar="CUÁNDO", nargs="?", const="semana",
                        help="Horas imputadas. Sin valor = esta semana. "
                             "También: pasada, mes, hoy, ayer, una fecha o --from/--to")

    parser.add_argument("terms", nargs="*", metavar="TERM",
                        help="Texto libre a buscar y/o claves de incidencia (PROJ-123)")

    parser.add_argument("-d", "--debug",         action="store_true",  help="Muestra el JQL y las llamadas a la API")
    parser.add_argument("-u", "--user",           metavar="USER", nargs="?", const="@me",
                        help="Asignado (nombre parcial). Sin valor = tú mismo")
    parser.add_argument("-R", "--reporter",       metavar="USER", nargs="?", const="@me",
                        help="Reportador (nombre parcial). Sin valor = tú mismo")
    parser.add_argument("-p", "--project",        metavar="PROJECT",    help="Clave o nombre parcial del proyecto (ej: 'PROJ', 'demo')")
    parser.add_argument("-s", "--search",         metavar="TEXT",       help="Busca las palabras sueltas en resumen, descripción o comentarios")
    parser.add_argument("-ss", "--search-exact",  metavar="TEXT",       help="Busca la frase exacta (la línea completa)")
    parser.add_argument("-t", "--type",           metavar="TYPE", action="append", help="Tipo de incidencia, por trozo de nombre (repetible o con comas)")
    parser.add_argument("-e", "--status",         metavar="STATUS", action="append", help="Estado, por trozo de nombre o @todo/@prog/@done (repetible o con comas)")
    parser.add_argument("-L", "--label",          metavar="LABEL", action="append", help="Etiqueta (repetible o separada por comas)")
    parser.add_argument("--from",                 metavar="FECHA", dest="date_from", help="Desde: 2026-01-31, 31/01/2026, 7d, 2w, 3m")
    parser.add_argument("--to",                   metavar="FECHA", dest="date_to",   help="Hasta (mismo formato que --from)")
    parser.add_argument("--date-field",           choices=sorted(DATE_FIELDS), default="updated",
                        help="Campo de fecha para --from/--to (por defecto: updated)")
    parser.add_argument("-r", "--recent",         metavar="DAYS", type=int, help="Atajo de --from N días")
    parser.add_argument("-a", "--active",         action="store_true",  help="Solo tipos y estados 'activos' (ver ~/.jrquery.json)")
    parser.add_argument("-l", "--limit",          metavar="N", type=int, default=50, help="Limita la salida a N resultados (por defecto: 50)")
    parser.add_argument("-c", "--count",          action="store_true",  help="Solo imprime el número de incidencias")
    parser.add_argument("-S", "--sprint",         action="store_true",  help="Solo incidencias del sprint activo")
    parser.add_argument("-O", "--unresolved",     action="store_true",  help="Solo incidencias sin resolver")
    parser.add_argument("-A", "--all",            action="store_true",  help="Sin filtros de tipo ni estado (ignora -a, -e y -O)")
    parser.add_argument("-q", "--query",          metavar="JQL",        help="Ejecuta un JQL propio")
    parser.add_argument("-f", "--filter",         metavar="FILTER_ID",  help="Busca usando el ID de un filtro guardado")
    parser.add_argument("-o", "--open",           metavar="ISSUE", nargs="*",
                        help="Abre incidencias en el navegador. Sin claves abre la búsqueda actual")
    parser.add_argument("-T", "--order-by-time",  action="count", default=0, help="Ordena por última actualización (-TT para invertir)")
    parser.add_argument("-U", "--order-by-user",  action="count", default=0, help="Ordena por asignado (-UU para invertir)")
    parser.add_argument("--list-projects",        action="store_true",  help="Lista los proyectos visibles")
    parser.add_argument("--list-users",           action="store_true",  help="Lista los usuarios de Jira")
    parser.add_argument("--list-filters",         action="store_true",  help="Lista los filtros guardados")
    parser.add_argument("--list-statuses",        action="store_true",  help="Lista los estados disponibles")
    parser.add_argument("--list-types",           action="store_true",  help="Lista los tipos de incidencia disponibles")
    parser.add_argument("--reconfigure",          action="store_true",  help="Reinicia y reconfigura las credenciales")

    args = parser.parse_args()

    # Argumentos sueltos: claves de incidencia por un lado, texto libre por otro
    args.keys, words = split_terms(args.terms)
    if words:
        free_text = " ".join(words)
        args.search = f"{args.search} {free_text}".strip() if args.search else free_text

    if args.reconfigure:
        setup_config()
        return

    config = get_config()
    client = JiraClient(config, debug=args.debug)

    if args.open is not None:
        open_issues(args, config, client)
        return

    if args.hours is not None:
        show_hours(args, config, client)
        return

    if args.list_projects:
        render_projects(client.list_projects())
        return

    if args.list_users:
        render_users(client.list_users())
        return

    if args.list_filters:
        render_filters(client.list_filters())
        return

    if args.list_statuses:
        render_statuses(client.status_catalog())
        return

    if args.list_types:
        render_types(client.type_catalog())
        return

    jql = resolve_jql(args, client)

    if args.debug:
        console.print(f"\n[dim]JQL: {escape(jql)}[/dim]\n")

    if args.count:
        total = client.count(jql)
        if total >= 0:
            render_issues([], total, config["base_url"], count_only=True)
            return

    issues, total = client.search(jql, limit=args.limit)
    render_issues(issues, total, config["base_url"], count_only=args.count)


if __name__ == "__main__":
    main()
