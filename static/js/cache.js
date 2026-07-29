(function () {
    const { formatMbps, formatBytes, createIfaceCard, fetchStats, apiGet } = window.Inforflow;

    async function update() {
        const [cache, stats, storage] = await Promise.all([
            apiGet('/cache').catch(() => null),
            fetchStats(),
            apiGet('/storage').catch(() => null)
        ]);
        if (cache) {
            document.getElementById('cache-hit-pct').textContent = (cache.estimated_hit_pct || 0).toFixed(1) + '%';
            document.getElementById('cache-snmp-total').textContent =
                formatMbps((cache.cache_snmp_in_mbps || 0) + (cache.cache_snmp_out_mbps || 0));
            document.getElementById('cache-stream-total').textContent = formatMbps(cache.streaming_scaled_mbps);
            document.getElementById('cache-netflix').textContent = formatMbps(cache.netflix_scaled_mbps);
            const ifaces = document.getElementById('cache-ifaces');
            if (ifaces) {
                ifaces.innerHTML = (cache.interfaces || []).length
                    ? cache.interfaces.map(createIfaceCard).join('')
                    : '<div class="dest-card"><div class="card-name">Nenhuma interface cache</div></div>';
            }
        }
        if (stats) {
            const el = document.getElementById('cache-ip-summary');
            if (el) {
                el.innerHTML = `
                    <div class="io-stat"><span class="io-label">IPv4</span><span class="io-value">${formatMbps(stats.ipv4_mbps)}</span></div>
                    <div class="io-stat"><span class="io-label">IPv6</span><span class="io-value">${formatMbps(stats.ipv6_mbps)}</span></div>
                    <div class="io-stat"><span class="io-label">Total NF</span><span class="io-value">${formatMbps(stats.mbps_scaled || stats.mbps)}</span></div>`;
            }
        }
        const si = document.getElementById('storage-info');
        if (si && storage) {
            si.innerHTML = `<p><strong>Local (VM):</strong> ${storage.local_hours || 72}h (~3 dias) · ${storage.local_points || 0} pontos · ${formatBytes(storage.local_db_bytes || 0)}</p>
                <p><strong>S3 Mega:</strong> ${storage.s3_enabled ? storage.s3_bucket + ' · ' + (storage.s3_days || 30) + ' dias' : 'desabilitado'}</p>
                <p><strong>Arquivos diários S3:</strong> ${(storage.s3_daily_archives || []).length} dias</p>
                <p><strong>Backups DB S3:</strong> ${(storage.s3_backups || []).slice(-3).join(', ') || 'nenhum'}</p>`;
        }
    }
    setInterval(update, 5000);
    update();
})();
