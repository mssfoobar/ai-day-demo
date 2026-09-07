#!/usr/bin/env python3
"""wfe catalog seed — register a worker's activities/events with the WFE manager.

OPTIONAL — enable it only when the catalog must be populated before first use.
The activities/events the designer offers come from rows in the manager's
database. This one-shot waits for the manager to migrate the schema, then upserts
the rows from catalog.yaml. Idempotent (ON CONFLICT on the unique type) — re-runs
and a fresh `down -v && up` converge.

Edit catalog.yaml with your worker's activities/events. It seeds the manager's
database (the worker has none).
"""

from __future__ import annotations
import logging
import os
import sys
import time

import psycopg
from psycopg.types.json import Jsonb
import yaml

log = logging.getLogger("wfe-init")


class Env:
    host = os.environ.get("SQL_HOST", "wfe-db")
    port = int(os.environ.get("SQL_PORT", "5432"))
    db = os.environ.get("SQL_DATABASE_NAME", "wfe")
    schema = os.environ.get("SQL_SCHEMA_NAME", "wfe")
    user = os.environ["DEV_USER"]
    password = os.environ["DEV_PASSWORD"]


def connect(attempts: int = 30, delay: float = 3.0) -> psycopg.Connection:
    last: Exception | None = None
    for i in range(attempts):
        try:
            return psycopg.connect(
                host=Env.host, port=Env.port, dbname=Env.db,
                user=Env.user, password=Env.password, autocommit=True,
            )
        except psycopg.OperationalError as e:
            last = e
            log.info("waiting for %s:%d/%s (%d/%d)", Env.host, Env.port, Env.db, i + 1, attempts)
            time.sleep(delay)
    sys.exit(f"could not connect to {Env.host}:{Env.port}/{Env.db}: {last}")


def wait_for_schema(conn: psycopg.Connection, attempts: int = 60, delay: float = 3.0) -> None:
    """The manager migrates the catalog tables on boot; seeding before that fails
    with 'relation does not exist'."""
    target = f"{Env.schema}.service_activity"
    for i in range(attempts):
        with conn.cursor() as cur:
            cur.execute("SELECT to_regclass(%s)", (target,))
            if cur.fetchone()[0] is not None:
                return
        log.info("waiting for %s (manager migration) (%d/%d)", target, i + 1, attempts)
        time.sleep(delay)
    sys.exit(f"{target} never appeared — is the manager running and migrating?")


# Per-activity lifecycle defaults (service_activity only; events run a fixed
# policy). Optional in catalog.yaml — fall back to the DB defaults when omitted.
LIFECYCLE_DEFAULTS = {"timeout_in_second": 300, "heartbeat_timeout_in_second": 60, "maximum_retry": 10}


def upsert(conn: psycopg.Connection, table: str, type_col: str, name: str, rows: list[dict]) -> None:
    prefix = "activity" if table == "service_activity" else "event"
    icon_col, param_col, result_col = f"{prefix}_icon", f"{prefix}_param", f"{prefix}_result"
    lifecycle = tuple(LIFECYCLE_DEFAULTS) if table == "service_activity" else ()
    cols = ["id", "service_name", type_col, icon_col, param_col, result_col, *lifecycle]
    placeholders = ", ".join(["gen_random_uuid()", *["%s"] * (len(cols) - 1)])
    updates = ", ".join(f"{c} = EXCLUDED.{c}" for c in cols if c not in ("id", type_col))
    sql = (
        f'INSERT INTO "{Env.schema}".{table} ({", ".join(cols)}) '
        f"VALUES ({placeholders}) "
        f"ON CONFLICT ({type_col}) DO UPDATE SET {updates}"
    )
    with conn.cursor() as cur:
        for r in rows:
            values = [
                name, r["type"], r.get("icon", "none"),
                Jsonb(r.get("param", [])), Jsonb(r.get("result")),
                *(r.get(c, LIFECYCLE_DEFAULTS[c]) for c in lifecycle),
            ]
            cur.execute(sql, values)
            log.info("%s %s: upserted", table, r["type"])


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    spec_file = sys.argv[1] if len(sys.argv) > 1 else "catalog.yaml"
    spec = yaml.safe_load(open(spec_file)) or {}
    name = spec.get("service_name", "workflow-worker")

    with connect() as conn:
        wait_for_schema(conn)
        upsert(conn, "service_activity", "activity_type", name, spec.get("activities") or [])
        upsert(conn, "service_event", "event_type", name, spec.get("events") or [])
    log.info("catalog seed complete")


if __name__ == "__main__":
    main()
