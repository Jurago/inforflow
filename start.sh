#!/bin/bash
set -e
DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$DIR"

echo "╔══════════════════════════════════════════╗"
echo "║     INFORFLOW — NetFlow + SNMP + BGP     ║"
echo "╚══════════════════════════════════════════╝"

verify_assets() {
  local missing=0
  local f
  for f in \
    bin/inforflow-collector \
    bin/inforflow \
    static/js/inforflow.js \
    static/js/dashboard.js \
    static/js/auth.js \
    static/css/inforflow.css
  do
    if [ ! -s "$DIR/$f" ]; then
      echo "  ✗ ausente: $f"
      missing=1
    fi
  done
  # Scripts referenciados pelo layout
  while IFS= read -r js; do
    [ -z "$js" ] && continue
    if [ ! -s "$DIR/static/js/$js" ]; then
      echo "  ✗ layout referencia static/js/$js (arquivo ausente)"
      missing=1
    fi
  done < <(grep -oE 'static/js/[a-z0-9_]+\.js' server/src/pages/layout.rs 2>/dev/null | sed 's|static/js/||' | sort -u || true)
  # Page scripts
  for js in asn.js asndetail.js cdns.js cdndetail.js peers.js peersdetail.js \
            router.js routerdetail.js streaming.js streamingdetail.js \
            graphs.js flows.js cache.js sampling.js; do
    if [ ! -s "$DIR/static/js/$js" ]; then
      echo "  ✗ ausente: static/js/$js"
      missing=1
    fi
  done
  if [ "$missing" != "0" ]; then
    echo "ERRO: deploy abortado — assets incompletos"
    exit 1
  fi
  echo "  ✓ assets OK"
}

echo "[1/4] Compilando coletor Go..."
(cd collector && go build -o ../bin/inforflow-collector .)

echo "[2/4] Compilando servidor Rust..."
(cd server && cargo build --release)
TARGET="${CARGO_TARGET_DIR:-$DIR/server/target}/release/inforflow"
if [ ! -x "$TARGET" ]; then
  # fallback sandbox / alt target dirs
  for cand in \
    "$DIR/server/target/release/inforflow" \
    /var/tmp/cargo-target/release/inforflow \
    "${CARGO_TARGET_DIR}/release/inforflow"
  do
    if [ -x "$cand" ]; then TARGET="$cand"; break; fi
  done
fi
cp -f "$TARGET" bin/inforflow
cp -f "$TARGET" bin/inforflow-web

echo "[3/4] Verificando assets..."
verify_assets

echo "[4/4] Instalando systemd + sysctl + nginx..."
mkdir -p data/feeds data/backups/local
[ -f deploy/peers.json ] && [ ! -f data/peers.json ] && cp deploy/peers.json data/peers.json

# Kernel UDP buffers
if [ -f deploy/99-inforflow-udp.conf ]; then
  cp -f deploy/99-inforflow-udp.conf /etc/sysctl.d/99-inforflow-udp.conf
  sysctl --system >/dev/null 2>&1 || sysctl -p /etc/sysctl.d/99-inforflow-udp.conf || true
fi

cp -f deploy/inforflow-collector.service deploy/inforflow-web.service \
  deploy/inforflow-healthcheck.service deploy/inforflow-healthcheck.timer \
  deploy/inforflow-backup.service deploy/inforflow-backup.timer \
  /etc/systemd/system/
systemctl daemon-reload
systemctl enable inforflow-collector inforflow-web \
  inforflow-healthcheck.timer inforflow-backup.timer
systemctl restart inforflow-collector inforflow-web
systemctl start inforflow-healthcheck.timer inforflow-backup.timer

if [ -f deploy/nginx-inforflow.conf ]; then
  cp -f deploy/nginx-inforflow.conf /etc/nginx/sites-available/inforflow
  ln -sf /etc/nginx/sites-available/inforflow /etc/nginx/sites-enabled/inforflow
  if [ -f deploy/nginx-rate-limit.conf ]; then
    cp -f deploy/nginx-rate-limit.conf /etc/nginx/conf.d/inforflow-rate-limit.conf
  fi
  nginx -t && systemctl reload nginx
fi

chmod +x scripts/*.sh 2>/dev/null || true

sleep 3
if /var/www/html/inforflow/scripts/healthcheck.sh; then
  echo ""
  echo "  Dashboard:  https://inforflow.infornetmg.com.br/"
  echo "  Health:     http://127.0.0.1:9090/api/health"
else
  echo "  FALHA no healthcheck — journalctl -u inforflow-collector -n 50"
  exit 1
fi
