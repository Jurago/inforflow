#!/bin/bash
# Backup local diário: config, peers, cache ASN (sem secrets)
set -euo pipefail
ROOT="${INFORFLOW_ROOT:-/var/www/html/inforflow}"
DEST="${INFORFLOW_BACKUP_DIR:-$ROOT/data/backups/local}"
KEEP_DAYS="${INFORFLOW_BACKUP_KEEP_DAYS:-14}"
STAMP=$(date +%Y%m%d-%H%M%S)
DIR="$DEST/$STAMP"
mkdir -p "$DIR"

copy_if() {
  local src="$1" name="$2"
  if [ -f "$src" ]; then
    cp -a "$src" "$DIR/$name"
  fi
}

copy_if "$ROOT/config.json" "config.json"
copy_if "$ROOT/data/peers.json" "peers.json"
copy_if "$ROOT/deploy/peers.json" "peers.deploy.json"
copy_if "$ROOT/data/ipapi_asn.json" "ipapi_asn.json"
copy_if "$ROOT/secrets.env.example" "secrets.env.example"

# Lista de assets JS (para auditoria pós-restore)
find "$ROOT/static/js" -type f -name '*.js' | sort > "$DIR/static-js.list" || true

# Compacta
tar -C "$DEST" -czf "$DEST/inforflow-local-$STAMP.tar.gz" "$STAMP"
rm -rf "$DIR"

# Retenção
find "$DEST" -name 'inforflow-local-*.tar.gz' -mtime +"$KEEP_DAYS" -delete 2>/dev/null || true

echo "OK backup $DEST/inforflow-local-$STAMP.tar.gz"
