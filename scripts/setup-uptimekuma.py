#!/usr/bin/env python3
"""Configura monitores Inforflow no Uptime Kuma."""
from __future__ import annotations

import os
import secrets
import string
import sys
import time
import urllib.request
import base64

import socketio

KUMA_URL = os.environ.get("INFORFLOW_UPTIMEKUMA_URL", "http://170.245.127.16:3001").rstrip("/")
KUMA_API_KEY = os.environ.get("INFORFLOW_UPTIMEKUMA_API_KEY", "")
KUMA_USER = os.environ.get("INFORFLOW_UPTIMEKUMA_USER", "")
KUMA_PASS = os.environ.get("INFORFLOW_UPTIMEKUMA_PASSWORD", "")
PUBLIC_URL = os.environ.get("INFORFLOW_PUBLIC_URL", "https://inforflow.infornetmg.com.br").rstrip("/")
PUSH_ENV = os.environ.get("INFORFLOW_UPTIMEKUMA_PUSH_FILE", "/etc/inforflow/kuma-push.env")

MONITORS = [
    {
        "name": "Inforflow Health API",
        "type": "http",
        "url": f"{PUBLIC_URL}/api/health",
        "interval": 60,
        "maxretries": 2,
        "accepted_statuscodes": ["200-299"],
    },
    {
        "name": "Inforflow Dashboard",
        "type": "http",
        "url": f"{PUBLIC_URL}/",
        "interval": 120,
        "maxretries": 2,
        "accepted_statuscodes": ["200-299"],
    },
    {
        "name": "Inforflow Login",
        "type": "http",
        "url": f"{PUBLIC_URL}/login",
        "interval": 300,
        "maxretries": 2,
        "accepted_statuscodes": ["200-299"],
    },
    {
        "name": "Inforflow Collector Push",
        "type": "push",
        "interval": 120,
        "maxretries": 1,
    },
]

PUSH_TOKEN_CHARS = string.ascii_letters + string.digits
PUSH_TOKEN_LENGTH = 32
DEFAULT_STATUS_CODES = ["200-299"]


def gen_push_token(length: int = PUSH_TOKEN_LENGTH) -> str:
    return "".join(secrets.choice(PUSH_TOKEN_CHARS) for _ in range(length))


def verify_api_key() -> bool:
    if not KUMA_API_KEY:
        print("AVISO: INFORFLOW_UPTIMEKUMA_API_KEY não definida")
        return False
    auth = base64.b64encode(f":{KUMA_API_KEY}".encode()).decode()
    req = urllib.request.Request(
        f"{KUMA_URL}/metrics",
        headers={"Authorization": f"Basic {auth}"},
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            print("OK API key — /metrics responde")
            return resp.status == 200
    except Exception as e:
        print(f"FALHA /metrics: {e}")
        return False


def connect_kuma():
    if not KUMA_USER or not KUMA_PASS:
        raise RuntimeError("credenciais Kuma ausentes")
    state = {"monitors": {}}
    sio = socketio.Client()

    @sio.on("monitorList")
    def on_monitor_list(data):
        state["monitors"] = data or {}

    sio.connect(KUMA_URL, transports=["websocket"], wait_timeout=15)
    login = sio.call("login", {"username": KUMA_USER, "password": KUMA_PASS}, timeout=15)
    if not login or not login.get("ok"):
        sio.disconnect()
        raise RuntimeError(f"login falhou: {login}")
    time.sleep(1.5)
    by_name = {}
    for m in state["monitors"].values():
        if isinstance(m, dict) and m.get("name"):
            by_name[m["name"]] = m
    return sio, by_name


def monitor_payload(spec: dict, *, push_token: str | None = None) -> dict:
    body = dict(spec)
    body.setdefault("retryInterval", 60)
    body.setdefault("resendInterval", 0)
    body.setdefault("upsideDown", False)
    body.setdefault("notificationIDList", {})
    body.setdefault("active", True)
    body.setdefault("conditions", [])
    body.setdefault("accepted_statuscodes", DEFAULT_STATUS_CODES)
    if body["type"] == "http":
        body.setdefault("method", "GET")
        body.setdefault("maxredirects", 10)
        body.setdefault("timeout", 48)
    if body["type"] == "push":
        body["pushToken"] = push_token or gen_push_token()
    return body


def add_monitor(sio, spec: dict) -> dict:
    return sio.call("add", monitor_payload(spec), timeout=20)


def edit_monitor(sio, monitor_id: int, spec: dict, push_token: str) -> dict:
    body = monitor_payload(spec, push_token=push_token)
    body["id"] = monitor_id
    return sio.call("editMonitor", body, timeout=20)


def extract_monitor(detail) -> dict | None:
    if not isinstance(detail, dict):
        return None
    if isinstance(detail.get("monitor"), dict):
        return detail["monitor"]
    if detail.get("id"):
        return detail
    return None


def setup_monitors() -> int:
    sio, existing = connect_kuma()
    print("Login OK como", KUMA_USER)
    push_url = None

    try:
        for spec in MONITORS:
            name = spec["name"]
            if name in existing:
                m = existing[name]
                mid = m.get("id")
                print("  já existe:", name, "(id", mid, ")")
                if spec["type"] == "push" and mid:
                    tok = m.get("pushToken")
                    if not tok:
                        tok = gen_push_token()
                        r = edit_monitor(sio, mid, spec, tok)
                        if not r or not r.get("ok"):
                            print("  ERRO ao corrigir push token:", name, r)
                            continue
                        print("  push token gerado para monitor existente")
                    push_url = f"{KUMA_URL}/api/push/{tok}"
                continue

            r = add_monitor(sio, spec)
            if not r or not r.get("ok"):
                print("  ERRO:", name, r)
                continue
            mid = r.get("monitorID")
            print("  criado:", name, "id", mid)
            if spec["type"] == "push" and mid:
                detail = extract_monitor(sio.call("getMonitor", mid, timeout=15))
                tok = detail.get("pushToken") if detail else None
                if tok:
                    push_url = f"{KUMA_URL}/api/push/{tok}"

        if push_url:
            with open(PUSH_ENV, "w") as f:
                f.write(f"INFORFLOW_UPTIMEKUMA_PUSH_URL={push_url}\n")
            os.chmod(PUSH_ENV, 0o600)
            print(f"\nPush URL salva em {PUSH_ENV}")
            with urllib.request.urlopen(
                f"{push_url}?status=up&msg=setup&ping=0", timeout=10
            ) as resp:
                print("Heartbeat test HTTP", resp.status)
        else:
            print("\nAVISO: push URL não encontrada — verifique monitor Push no Kuma")
        return 0
    finally:
        sio.disconnect()


def main() -> int:
    print("=== Uptime Kuma — Inforflow ===")
    print("URL:", KUMA_URL)
    verify_api_key()
    try:
        return setup_monitors()
    except Exception as e:
        print("Erro:", e)
        return 1


if __name__ == "__main__":
    sys.exit(main())
