#!/bin/bash
set -e
DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$DIR"

echo "╔══════════════════════════════════════════╗"
echo "║     INFORFLOW — NetFlow + SNMP + BGP     ║"
echo "╚══════════════════════════════════════════╝"

echo "[1/3] Compilando coletor Go..."
(cd collector && go build -o ../bin/inforflow-collector .)

echo "[2/3] Compilando servidor Rust..."
(cd server && cargo build --release)
TARGET="${CARGO_TARGET_DIR:-$DIR/server/target}/release/inforflow"
cp -f "$TARGET" bin/inforflow

	echo "[3/3] Reiniciando via systemd..."
mkdir -p data/feeds
[ -f deploy/peers.json ] && [ ! -f data/peers.json ] && cp deploy/peers.json data/peers.json
cp -f deploy/inforflow-collector.service deploy/inforflow-web.service \
  deploy/inforflow-healthcheck.service deploy/inforflow-healthcheck.timer /etc/systemd/system/
systemctl daemon-reload
systemctl enable inforflow-collector inforflow-web inforflow-healthcheck.timer
systemctl restart inforflow-collector inforflow-web
systemctl start inforflow-healthcheck.timer

if [ -f deploy/nginx-inforflow.conf ]; then
  cp -f deploy/nginx-inforflow.conf /etc/nginx/sites-available/inforflow
  ln -sf /etc/nginx/sites-available/inforflow /etc/nginx/sites-enabled/inforflow
  if [ -f deploy/nginx-rate-limit.conf ]; then
    cp -f deploy/nginx-rate-limit.conf /etc/nginx/conf.d/inforflow-rate-limit.conf
  fi
  nginx -t && systemctl reload nginx
fi

sleep 3
if /var/www/html/inforflow/scripts/healthcheck.sh; then
  echo ""
  echo "  Dashboard:  https://inforflow.infornetmg.com.br/"
  echo "  Health:     http://127.0.0.1:9090/api/health"
else
  echo "  FALHA no healthcheck — journalctl -u inforflow-collector -n 50"
  exit 1
fi
