(function () {
    const { formatMbps, fetchStats, fetchSampling, createCard } = window.Inforflow;

    async function update() {
        const [stats, samp] = await Promise.all([fetchStats(), fetchSampling()]);
        const s = samp || stats?.sampling;
        if (!s) return;

        document.getElementById('samp-effective').textContent = '~' + Math.round(s.effective || 1) + '×';
        document.getElementById('samp-mode').textContent = (s.mode || '—') + ' · ' + (s.source || '');
        document.getElementById('samp-native').textContent = s.native > 1 ? ('1:' + Math.round(s.native)) : 'não detectado';
        document.getElementById('samp-estimated').textContent = s.estimated > 1 ? ('~' + Math.round(s.estimated) + '×') : '—';
        document.getElementById('samp-scaled').textContent = formatMbps(s.scaled_mbps);
        document.getElementById('samp-compare').textContent =
            'NF ' + formatMbps(s.netflow_mbps) + ' · SNMP ' + formatMbps(s.snmp_mbps);
        const src = document.getElementById('samp-source');
        if (src) src.textContent = 'fonte atual: ' + (s.source || s.mode || '—');

        if (!stats) return;
        const cats = document.getElementById('samp-cat-inout');
        if (cats) {
            const keys = new Set([
                ...Object.keys(stats.by_category_in_mbps || {}),
                ...Object.keys(stats.by_category_out_mbps || {})
            ]);
            const items = [...keys].map(k => {
                const inn = (stats.by_category_in_mbps || {})[k] || 0;
                const out = (stats.by_category_out_mbps || {})[k] || 0;
                return {
                    name: k,
                    category: k,
                    bytes: 0,
                    mbps: inn + out,
                    percentage: 0,
                    _extra: formatMbps(inn) + ' ↓ · ' + formatMbps(out) + ' ↑'
                };
            }).sort((a, b) => b.mbps - a.mbps);
            cats.innerHTML = items.slice(0, 12).map(it => `
                <div class="dest-card cat-${it.category}">
                    <div class="card-name">${it.name}</div>
                    <div class="card-mbps">${formatMbps(it.mbps)}</div>
                    <div class="card-pct">${it._extra}</div>
                </div>`).join('') || '<div class="dest-card"><div class="card-name">Aguardando…</div></div>';
        }

        const roles = document.getElementById('samp-iface-roles');
        if (roles) {
            const entries = Object.entries(stats.by_iface_role_mbps || {}).sort((a, b) => b[1] - a[1]);
            roles.innerHTML = entries.length
                ? entries.map(([name, mbps]) => createCard({
                    name: name.toUpperCase(),
                    bytes: 0,
                    mbps,
                    percentage: 0,
                    category: name === 'cache' || name === 'ix' ? 'cdn' : 'peer'
                })).join('')
                : '<div class="dest-card"><div class="card-name">Sem ifIndex mapeado ainda</div></div>';
        }
    }

    setInterval(update, 3000);
    update();
})();
