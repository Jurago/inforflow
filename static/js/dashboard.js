(function () {
    const {
        formatBytes, formatMbps, formatRate, formatNumber, createCard, createIfaceCard,
        renderCategoryBars, renderConsumption, spawnFlowParticle, renderAlerts,
        fetchStats, fetchFlows, fetchAlerts, categoryBadge, directionBadge, exportURL,
        tweenMbps, prefersReducedMotion
    } = window.Inforflow;

    let prevFlows = [];
    let lastMbpsScaled = 0;

    async function updateDashboard() {
        const stats = await fetchStats();
        if (!stats) return;

        tweenMbps(document.getElementById('nf-mbps'), stats.mbps);
        const rateEl = document.getElementById('bytes-rate');
        if (rateEl) rateEl.textContent = formatRate(stats.bytes_per_sec || 0) + ' · ' + formatBytes(stats.total_bytes);

        tweenMbps(document.getElementById('nf-scaled-mbps'), stats.mbps_scaled || stats.mbps);
        lastMbpsScaled = stats.mbps_scaled || stats.mbps || 0;
        const sampInfo = document.getElementById('sampling-info');
        if (sampInfo && stats.sampling) {
            const mode = stats.sampling.mode || 'auto';
            const native = stats.sampling.native > 1 ? ` · nativo 1:${Math.round(stats.sampling.native)}` : '';
            sampInfo.textContent = `fator ~${(stats.sampling.effective || 1).toFixed(0)}× (${mode}${native})`;
        }

        const exp = document.getElementById('export-csv');
        if (exp) exp.href = exportURL('stats', 'csv');

        renderAlerts(stats.alerts || [], 'alerts-list');

        const snmp = stats.snmp;
        if (snmp && snmp.ok) {
            document.getElementById('snmp-uplink-mbps').textContent =
                formatMbps(snmp.uplink_in_mbps) + ' / ' + formatMbps(snmp.uplink_out_mbps);
            document.getElementById('snmp-router-name').textContent =
                (snmp.sys_name || 'roteador') + ' · util ' + (snmp.uplink_util_pct || 0).toFixed(1) + '%' +
                (snmp.deduped ? ' · dedupe' : '');
            document.getElementById('snmp-cpu-mem').textContent =
                (snmp.cpu_pct || 0).toFixed(0) + '% / ' + (snmp.mem_pct || 0).toFixed(0) + '%';
            document.getElementById('snmp-uptime').textContent = snmp.uptime_human || '—';

            const crit = document.getElementById('snmp-critical');
            if (crit) {
                const roles = ['bras', 'uplink', 'ix', 'cgnat', 'cache'];
                const list = (snmp.interfaces || [])
                    .filter(i => i.oper_status === 1 && roles.includes(i.role))
                    .slice(0, 8);
                crit.innerHTML = list.map(createIfaceCard).join('') || '<div class="dest-card"><div class="card-name">Aguardando SNMP…</div></div>';
            }
        }

        const destCount = document.getElementById('dest-count');
        if (destCount) destCount.textContent = Object.keys(stats.by_destination || {}).length + ' serviços';

        renderConsumption(stats.consumption || []);
        renderCategoryBars(
            stats.by_category || {},
            stats.total_bytes,
            stats.by_category_mbps_scaled || stats.by_category_mbps
        );

        const talkersBody = document.getElementById('talkers-body');
        if (talkersBody) {
            const list = stats.top_talkers || [];
            talkersBody.innerHTML = list.length
                ? list.slice(0, 12).map(t => `<tr>
                    <td><code>${t.ip}</code></td>
                    <td>${formatMbps(t.mbps_scaled || t.mbps)}</td>
                    <td>${categoryBadge(t.top_category)}</td>
                    <td>${formatBytes(t.bytes)}</td>
                    <td>${formatNumber(t.flows)}</td>
                </tr>`).join('')
                : '<tr><td colspan="5">Aguardando clientes CGNAT…</td></tr>';
        }

        const destContainer = document.getElementById('top-destinations');
        if (destContainer && stats.top_destinations) {
            destContainer.innerHTML = stats.top_destinations.map(createCard).join('');
        }
        const origContainer = document.getElementById('top-origins');
        if (origContainer && stats.top_origins) {
            origContainer.innerHTML = stats.top_origins.map(createCard).join('');
        }

        const flows = await fetchFlows();
        const tbody = document.getElementById('flow-table-body');
        if (tbody && flows.length) {
            tbody.innerHTML = flows.slice(0, 25).map(f => {
                const time = new Date(f.timestamp * 1000).toLocaleTimeString('pt-BR');
                return `<tr>
                    <td>${time}</td>
                    <td title="${f.src_ip}">${f.origin}</td>
                    <td title="${f.dst_ip}">${f.destination}</td>
                    <td>${f.protocol}</td>
                    <td>${formatBytes(f.bytes)}</td>
                    <td>${categoryBadge(f.category)}</td>
                    <td>${directionBadge(f.direction)}</td>
                </tr>`;
            }).join('');
        }

        flows.slice(0, 3).forEach(f => {
            if (!prevFlows.find(p => p.id === f.id)) {
                spawnFlowParticle('pipeline-track', f.category, lastMbpsScaled);
            }
        });
        prevFlows = flows;
    }

    setInterval(updateDashboard, 2000);
    updateDashboard();
    setInterval(() => {
        if (prefersReducedMotion()) return;
        const cats = ['cdn', 'netflix', 'globo', 'streaming', 'peer'];
        spawnFlowParticle('pipeline-track', cats[Math.floor(Math.random() * cats.length)], lastMbpsScaled);
    }, 1600);
})();
