(function() {
    const {
        formatBytes, formatMbps, fetchASNDetail, categoryBadge, directionBadge,
        updateSamplingChip, prefersReducedMotion
    } = window.Inforflow;

    const params = new URLSearchParams(window.location.search);
    const asn = params.get('asn') || '';
    let hours = 6;

    if (!asn) {
        window.location.href = '/asn';
        return;
    }

    document.getElementById('asn-detail-flows-link').href = '/flows?asn=' + encodeURIComponent(asn);

    function drawChart(hist) {
        const canvas = document.getElementById('asd-chart');
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
        pts.forEach(p => {
            maxV = Math.max(maxV, p.mbps_scaled || 0, p.peer_mbps_scaled || 0);
        });
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

        function stroke(color, key) {
            ctx.strokeStyle = color;
            ctx.lineWidth = 2;
            ctx.beginPath();
            pts.forEach((p, i) => {
                const x = pad.l + (i / Math.max(n - 1, 1)) * plotW;
                const y = pad.t + plotH - ((p[key] || 0) / maxV) * plotH;
                if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
            });
            ctx.stroke();
        }
        if (pts.length) {
            stroke('#0ea5e9', 'mbps_scaled');
            stroke('#10b981', 'peer_mbps_scaled');
        }
        const legend = document.getElementById('asd-legend');
        if (legend) {
            legend.innerHTML = `
                <span class="legend-item"><i style="background:#0ea5e9"></i>Destino</span>
                <span class="legend-item"><i style="background:#10b981"></i>Peer</span>`;
        }
    }

    async function refresh() {
        const data = await fetchASNDetail(asn, hours);
        if (!data) return;
        updateSamplingChip(data.sampling);

        const live = data.live;
        const peer = data.peer_live;
        const daily = data.daily;
        const title = (live && live.name) || (peer && peer.name) || data.asn;
        document.getElementById('asn-detail-title').textContent = `${title} (${data.asn})`;
        document.getElementById('asn-detail-sub').textContent =
            (live && live.pending) ? 'ASN pendente de verificação ip-api/BGP'
                : 'Mbps = janela ~10s × amostragem · Bytes = acumulado do dia';

        document.getElementById('asd-mbps').textContent = formatMbps(live ? live.mbps_scaled : 0);
        document.getElementById('asd-peer-mbps').textContent = formatMbps(peer ? peer.mbps_scaled : 0);
        document.getElementById('asd-bytes').textContent = formatBytes(daily ? daily.bytes : (live && live.bytes) || 0);
        document.getElementById('asd-pct').textContent = daily
            ? `${(daily.percentage || 0).toFixed(1)}% do dia`
            : (live ? `${(live.percentage || 0).toFixed(1)}% acumulado` : '—');
        document.getElementById('asd-io').textContent = live
            ? `${formatMbps(live.in_mbps)} ↓ / ${formatMbps(live.out_mbps)} ↑`
            : '—';
        document.getElementById('asd-role').textContent = live
            ? `destino · ${live.category || 'other'}`
            : (peer ? 'só como peer BGP' : 'sem tráfego ao vivo');

        drawChart(data.history || []);

        const tbody = document.getElementById('asd-flows');
        const flows = data.flows || [];
        tbody.innerHTML = flows.map(f => {
            const ts = f.timestamp ? new Date(f.timestamp * 1000).toLocaleTimeString('pt-BR') : '—';
            return `<tr>
                <td>${ts}</td>
                <td><code>${f.src_ip}</code> → <code>${f.dst_ip}</code></td>
                <td>${f.peer_asn || '—'}</td>
                <td>${categoryBadge(f.category)}</td>
                <td>${formatBytes(f.bytes)}</td>
                <td>${directionBadge(f.direction)}</td>
            </tr>`;
        }).join('') || '<tr><td colspan="6">Nenhum flow recente para este ASN</td></tr>';
    }

    document.getElementById('asd-time-filter')?.addEventListener('click', (e) => {
        const btn = e.target.closest('.tf-btn[data-h]');
        if (!btn) return;
        document.querySelectorAll('#asd-time-filter .tf-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        hours = parseInt(btn.dataset.h, 10) || 6;
        refresh();
    });

    setInterval(refresh, 3000);
    refresh();
    window.addEventListener('resize', () => { /* redraw on next refresh */ });
    if (!prefersReducedMotion()) { /* noop */ }
})();
