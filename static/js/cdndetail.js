(function() {
    const {
        formatBytes, formatMbps, fetchCDNDetail, directionBadge,
        updateSamplingChip, createIfaceCard, prefersReducedMotion, exportURL
    } = window.Inforflow;

    const params = new URLSearchParams(window.location.search);
    const name = params.get('name') || '';
    let hours = 6;

    if (!name) {
        window.location.href = '/cdns';
        return;
    }

    function drawChart(hist) {
        const canvas = document.getElementById('cd-chart');
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
        ctx.strokeStyle = '#f48120';
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
        const data = await fetchCDNDetail(name, hours);
        if (!data) return;
        updateSamplingChip(data.sampling);

        const exportLink = document.getElementById('export-csv');
        if (exportLink) exportLink.href = exportURL('cdn', 'csv');

        const live = data.live;
        const title = data.name || name;
        document.getElementById('cd-title').textContent = title;
        document.getElementById('cd-sub').textContent =
            (data.asn ? data.asn + ' · ' : '') + 'Mbps = janela ~10s × amostragem';

        const asnLink = document.getElementById('cd-asn-link');
        if (asnLink) {
            asnLink.href = data.asn
                ? '/asn/detail?asn=' + encodeURIComponent(data.asn)
                : '/asn';
            asnLink.textContent = data.asn ? 'Ver ' + data.asn : 'Ver ASN';
        }

        document.getElementById('cd-mbps').textContent = formatMbps(live ? live.mbps_scaled : 0);
        document.getElementById('cd-mbps-hint').textContent = live
            ? `NF amostra: ${formatMbps(live.mbps || 0)}`
            : 'sem taxa ao vivo';
        document.getElementById('cd-bytes').textContent = formatBytes(live ? live.bytes : 0);
        document.getElementById('cd-pct').textContent = live && live.percentage != null
            ? live.percentage.toFixed(1) + '% do CDN (bytes)'
            : '—';
        document.getElementById('cd-v4v6').textContent = live
            ? formatMbps(live.ipv4_mbps || 0).replace(' Mbps', '') + ' / ' +
              formatMbps(live.ipv6_mbps || 0).replace(' Mbps', '')
            : '—';
        document.getElementById('cd-io').textContent = live
            ? `${formatMbps(live.in_mbps || 0)} ↓ · ${formatMbps(live.out_mbps || 0)} ↑`
            : '↓ / ↑';

        document.getElementById('cd-snmp').textContent = formatMbps(data.snmp_match_mbps || 0);
        const ifaces = data.cache_ifaces || [];
        document.getElementById('cd-snmp-hint').textContent =
            ifaces.length ? `${ifaces.length} iface(s)` : 'sem match SNMP';

        const alerts = document.getElementById('cd-alerts');
        if (alerts) {
            const items = [];
            if (data.divergence_warn) {
                items.push(`<div class="alert-item alert-warning"><strong>Divergência</strong><span>${data.divergence_warn}</span></div>`);
            }
            if (data.overlap_note) {
                items.push(`<div class="alert-item alert-warning"><strong>Overlap</strong><span>${data.overlap_note}</span></div>`);
            }
            alerts.innerHTML = items.length
                ? items.join('')
                : '<div class="alert-item alert-ok">SNMP e NetFlow alinhados (ou sem iface match)</div>';
        }

        const snmpBox = document.getElementById('cd-snmp-ifaces');
        if (snmpBox) {
            snmpBox.innerHTML = ifaces.length
                ? ifaces.map(createIfaceCard).join('')
                : '<div class="dest-card"><div class="card-name">Nenhuma interface SNMP correlacionada a este CDN</div></div>';
        }

        drawChart(data.history || []);

        const fb = document.getElementById('cd-flows');
        const flows = data.flows || [];
        fb.innerHTML = flows.map(f => {
            const ts = f.timestamp ? new Date(f.timestamp * 1000).toLocaleTimeString('pt-BR') : '—';
            const asn = f.asn || f.dst_asn || '';
            return `<tr>
                <td>${ts}</td>
                <td><code>${f.src_ip || '—'}</code> → <code>${f.dst_ip || '—'}</code></td>
                <td>${asn ? `<a href="/asn/detail?asn=${encodeURIComponent(asn)}">${asn}</a>` : '—'}</td>
                <td>${formatBytes(f.bytes)}</td>
                <td>${directionBadge(f.direction)}</td>
            </tr>`;
        }).join('') || '<tr><td colspan="5">Nenhum flow recente</td></tr>';
    }

    document.getElementById('cd-time-filter')?.addEventListener('click', (e) => {
        const btn = e.target.closest('.tf-btn[data-h]');
        if (!btn) return;
        document.querySelectorAll('#cd-time-filter .tf-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        hours = parseInt(btn.dataset.h, 10) || 6;
        refresh();
    });

    setInterval(refresh, 3000);
    refresh();
    if (!prefersReducedMotion()) { /* noop */ }
})();
