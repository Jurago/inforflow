(function() {
    const {
        formatBytes, formatMbps, fetchStreamingDetail, categoryBadge, directionBadge,
        updateSamplingChip, createIfaceCard, prefersReducedMotion, exportURL
    } = window.Inforflow;

    const params = new URLSearchParams(window.location.search);
    const name = params.get('name') || '';
    let hours = 6;

    if (!name) {
        window.location.href = '/streaming';
        return;
    }

    function drawChart(hist) {
        const canvas = document.getElementById('sd-chart');
        if (!canvas) return;
        const rect = canvas.getBoundingClientRect();
        const w = rect.width || 800;
        const h = 240;
        const dpr = window.devicePixelRatio || 1;
        canvas.width = w * dpr;
        canvas.height = h * dpr;
        canvas.style.width = w + 'px';
        canvas.style.height = h + 'px';
        const ctx = canvas.getContext('2d');
        ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
        ctx.clearRect(0, 0, w, h);
        const pad = { l: 50, r: 16, t: 16, b: 28 };
        const plotW = w - pad.l - pad.r;
        const plotH = h - pad.t - pad.b;
        const pts = hist || [];
        let maxV = 1;
        pts.forEach(p => { maxV = Math.max(maxV, p.mbps_scaled || 0); });
        const n = Math.max(pts.length, 2);

        ctx.strokeStyle = 'rgba(148,163,184,0.2)';
        for (let i = 0; i <= 4; i++) {
            const y = pad.t + (plotH * i) / 4;
            ctx.beginPath();
            ctx.moveTo(pad.l, y);
            ctx.lineTo(pad.l + plotW, y);
            ctx.stroke();
            ctx.fillStyle = '#94a3b8';
            ctx.font = '11px IBM Plex Mono, monospace';
            ctx.fillText(formatMbps(maxV * (1 - i / 4)).replace(' Mbps', ''), 4, y + 4);
        }

        if (!pts.length) return;
        ctx.strokeStyle = '#8b5cf6';
        ctx.lineWidth = 2;
        ctx.beginPath();
        pts.forEach((p, i) => {
            const x = pad.l + (i / Math.max(n - 1, 1)) * plotW;
            const y = pad.t + plotH - ((p.mbps_scaled || 0) / maxV) * plotH;
            if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
        });
        ctx.stroke();
    }

    async function refresh() {
        const data = await fetchStreamingDetail(name, hours);
        if (!data) return;
        updateSamplingChip(data.sampling);

        const exportLink = document.getElementById('export-csv');
        if (exportLink) exportLink.href = exportURL('streaming', 'csv');

        const live = data.live;
        const title = data.name || name;
        document.getElementById('sd-title').textContent = title;
        document.getElementById('sd-sub').textContent =
            (live && live.category ? 'Categoria: ' + live.category + ' · ' : '') +
            'Mbps = janela ~10s × amostragem';

        document.getElementById('sd-mbps').textContent = formatMbps(live ? live.mbps_scaled : 0);
        document.getElementById('sd-mbps-hint').textContent = live
            ? `NF amostra: ${formatMbps(live.mbps || 0)}`
            : 'sem taxa ao vivo';
        document.getElementById('sd-bytes').textContent = formatBytes(live ? live.bytes : 0);
        document.getElementById('sd-pct').textContent = live && live.percentage != null
            ? live.percentage.toFixed(1) + '% do streaming (bytes)'
            : '—';
        document.getElementById('sd-v4v6').textContent = live
            ? formatMbps(live.ipv4_mbps || 0).replace(' Mbps', '') + ' / ' +
              formatMbps(live.ipv6_mbps || 0).replace(' Mbps', '')
            : '—';
        document.getElementById('sd-io').textContent = live
            ? `${formatMbps(live.in_mbps || 0)} ↓ · ${formatMbps(live.out_mbps || 0)} ↑`
            : '↓ / ↑';

        document.getElementById('sd-snmp').textContent = formatMbps(data.snmp_match_mbps || 0);
        const ifaces = data.cache_ifaces || [];
        document.getElementById('sd-snmp-hint').textContent =
            ifaces.length ? `${ifaces.length} iface(s)` : 'sem match SNMP';

        const alerts = document.getElementById('sd-alerts');
        if (alerts) {
            alerts.innerHTML = data.divergence_warn
                ? `<div class="alert-item alert-warning"><strong>Divergência</strong><span>${data.divergence_warn}</span></div>`
                : '<div class="alert-item alert-ok">SNMP e NetFlow alinhados (ou sem iface match)</div>';
        }

        const snmpBox = document.getElementById('sd-snmp-ifaces');
        if (snmpBox) {
            snmpBox.innerHTML = ifaces.length
                ? ifaces.map(createIfaceCard).join('')
                : '<div class="dest-card"><div class="card-name">Nenhuma interface SNMP correlacionada a este serviço</div></div>';
        }

        drawChart(data.history || []);

        const fb = document.getElementById('sd-flows');
        const flows = data.flows || [];
        fb.innerHTML = flows.map(f => {
            const ts = f.timestamp ? new Date(f.timestamp * 1000).toLocaleTimeString('pt-BR') : '—';
            return `<tr>
                <td>${ts}</td>
                <td><code>${f.src_ip || '—'}</code> → <code>${f.dst_ip || '—'}</code></td>
                <td>${categoryBadge(f.category)}</td>
                <td>${formatBytes(f.bytes)}</td>
                <td>${directionBadge(f.direction)}</td>
            </tr>`;
        }).join('') || '<tr><td colspan="5">Nenhum flow recente</td></tr>';
    }

    document.getElementById('sd-time-filter')?.addEventListener('click', (e) => {
        const btn = e.target.closest('.tf-btn[data-h]');
        if (!btn) return;
        document.querySelectorAll('#sd-time-filter .tf-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        hours = parseInt(btn.dataset.h, 10) || 6;
        refresh();
    });

    setInterval(refresh, 3000);
    refresh();
    if (!prefersReducedMotion()) { /* noop */ }
})();
