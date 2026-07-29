#!/bin/bash
# Smoke test pós-deploy (requer coletor rodando)
set -e
curl -sf http://127.0.0.1:9090/api/health | python3 -c "
import sys,json
d=json.load(sys.stdin)
assert d.get('status')=='ok', d
print('smoke OK', d.get('flows'), 'flows')
"
