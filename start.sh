#!/bin/bash
set -e
DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$DIR"
mkdir -p bin data/feeds

echo "╔══════════════════════════════════════════╗"
echo "║     INFORFLOW — NetFlow + SNMP + BGP     ║"
echo "╚══════════════════════════════════════════╝"

echo "[1/4] Compilando coletor Go..."
(cd collector && go build -o ../bin/inforflow-collector .)

echo "[2/4] Compilando servidor Rust..."
(cd server && cargo build --release)
TARGET="${CARGO_TARGET_DIR:-$DIR/server/target}/release/inforflow"
cp -f "$TARGET" bin/inforflow

echo "[3/4] Reiniciando serviços..."
for pid in $(pgrep -f '/bin/inforflow-collector' || true); do kill "$pid" 2>/dev/null || true; done
for pid in $(pgrep -f '/bin/inforflow$' || true); do kill "$pid" 2>/dev/null || true; done
sleep 1

"$DIR/bin/inforflow-collector" >> /tmp/inforflow-collector.log 2>&1 &
COL_PID=$!
echo "  Coletor NetFlow :2055 + API :9090 (pid $COL_PID)"
sleep 2
"$DIR/bin/inforflow" >> /tmp/inforflow-server.log 2>&1 &
WEB_PID=$!
echo "  UI http://localhost:8080 (pid $WEB_PID)"

echo "[4/4] Health check..."
ok=0
for i in 1 2 3 4 5 6 7 8; do
  if curl -sf http://127.0.0.1:9090/api/health >/dev/null 2>&1 \
     && curl -sf -o /dev/null http://127.0.0.1:8080/; then
    ok=1
    break
  fi
  sleep 1
done

if [ "$ok" = 1 ]; then
  health=$(curl -sf http://127.0.0.1:9090/api/health)
  echo "  OK — $health" | head -c 200; echo
  echo ""
  echo "  Dashboard:  http://localhost:8080"
  echo "  Stats:      http://localhost:9090/api/stats"
  echo "  Alerts:     http://localhost:9090/api/alerts"
  echo "  Export:     http://localhost:9090/api/export?kind=cdn&format=csv"
  echo "  Config:     $DIR/config.json"
else
  echo "  FALHA no health check — veja /tmp/inforflow-collector.log e /tmp/inforflow-server.log"
  exit 1
fi
