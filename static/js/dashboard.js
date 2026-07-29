(function () {
    const {
        formatBytes, formatMbps, formatRate, formatNumber, createCard, createIfaceCard,
        renderCategoryBars, renderConsumption, spawnFlowParticle, renderAlerts,
        fetchStats, fetchFlows, fetchAlerts, categoryBadge, directionBadge,
        tweenMbps, prefersReducedMotion, updateSourceIP, exportDownload
    } = window.Inforflow;

    let prevFlows = [];
    let lastMbpsScaled = 0;

    document.getElementById('export-csv')?.addEventListener('click', e => {
        e.preventDefault();
        exportDownload('stats', 'csv');
    });

    function renderBGP(bgp) {
        const el = document.getElementById('bgp-summary');
        const hint = document.getElementById('bgp-summary-hint');
        if (!el || !bgp) return;
        if (hint) {
            hint.textContent = bgp.ok
                ? `${bgp.established}/${bgp.total} estabelecidas · AS${bgp.local_as || ''}`
                : (bgp.error || 'indisponível');
        }
        const peers = (bgp.peers || []).slice().sort((a, b) => (b.mbps || 0) - (a.mbps || 0));
        const down = peers.filter(p => !p.established).slice(0, 4);
        const up = peers.filter(p => p.established).slice(0, 8);
        el.innerHTML = [
            ...up.map(p => `<div class="dest-card cat-${p.role === 'ix' ? 'peer' : 'cdn'}">
                <div class="card-name">${p.name || p.asn}</div>
                <div class="card-mbps">${formatMbps(p.mbps)}</div>
                <div class="card-pct">${p.remote_addr || ''} · ${p.state_name || 'up'}</div>
            </div>`),
            ...down.map(p => `<div class="dest-card" style="opacity:0.65;border-color:#f87171">
                <div class="card-name">${p.name || p.asn} ↓</div>
                <div class="card-pct">${p.state_name || 'down'}</div>
            </div>`)
        ].join('') || '<div class="dest-card"><div class="card-name">Aguardando BGP…</div></div>';
    }

    function updateAlertBadge(count) {
        const b = document.getElementById('alert-badge');
        if (!b) return;
        if (count > 0) {
            b.style.display = 'inline-flex';
            b.textContent = count + ' alerta' + (count > 1 ? 's' : '');
        } else {
            b.style.display = 'none';
        }
    }

    async function updateDashboard() {
        const stats = await fetchStats();
        if (!stats) return;

        updateSourceIP(stats.source || stats.exporter);

        tweenMbps(document.getElementById('nf-mbps'), stats.mbps);
        const rateEl = document.getElementById('bytes-rate');
        if (rateEl) rateEl.textContent = formatRate(stats.bytes_per_sec || 0) + ' · ' + formatBytes(stats.total_bytes);

        tweenMbps(document.getElementById('nf-scaled-mbps'), stats.mbps_scaled || stats.mbps);
        lastMbpsScaled = stats.mbps_scaled || stats.mbps || 0;
        const sampInfo = document.getElementById('sampling-info');
        if (sampInfo && stats.sampling) {
            const mode = stats.sampling.mode || 'auto';
            const native = stats.sampling.native > 1 ? ` · nativo 1:${Math.round(stats.sampling.native)}` : '';
            const snmp = stats.snmp;
            let warn = '';
            if (mode === 'auto' && snmp && snmp.ok && stats.sampling.scaled_mbps && snmp.uplink_in_mbps) {
                const ratio = stats.sampling.scaled_mbps / Math.max((snmp.uplink_in_mbps + snmp.uplink_out_mbps) / 2, 1);
                if (ratio < 0.5 || ratio > 2) warn = ' ⚠ divergência SNMP';
            }
            sampInfo.textContent = `fator ~${(stats.sampling.effective || 1).toFixed(0)}× (${mode}${native})${warn}`;
            if (warn) sampInfo.style.color = '#fbbf24';
            else sampInfo.style.color = '';
        }

        renderAlerts(stats.alerts || [], 'alerts-list');
        updateAlertBadge((stats.alerts || []).length);
        renderBGP(stats.bgp);

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
