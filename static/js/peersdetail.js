(function() {
    const {
        formatBytes, formatMbps, fetchPeersDetail, categoryBadge, directionBadge,
        updateSamplingChip, prefersReducedMotion
    } = window.Inforflow;

    const params = new URLSearchParams(window.location.search);
    const asn = params.get('asn') || '';
    let hours = 6;

    if (!asn) {
        window.location.href = '/peers';
        return;
    }

    document.getElementById('pd-asn-link').href = '/asn/detail?asn=' + encodeURIComponent(asn);

    function formatUptime(sec) {
        if (!sec || sec < 0) return '—';
        if (sec < 60) return sec + 's';
        if (sec < 3600) return Math.floor(sec / 60) + 'm';
        if (sec < 86400) return (sec / 3600).toFixed(1) + 'h';
        return (sec / 86400).toFixed(1) + 'd';
    }

    function stateBadge(stateName, established) {
        const cls = established ? 'inbound' : 'outbound';
        return `<span class="badge badge-${cls}">${established ? 'established' : (stateName || '—')}</span>`;
    }

    function drawChart(hist) {
        const canvas = document.getElementById('pd-chart');
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
        pts.forEach(p => { maxV = Math.max(maxV, p.peer_mbps_scaled || p.mbps_scaled || 0); });
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
        ctx.strokeStyle = '#10b981';
        ctx.lineWidth = 2;
        ctx.beginPath();
        pts.forEach((p, i) => {
            const x = pad.l + (i / Math.max(n - 1, 1)) * plotW;
            const y = pad.t + plotH - ((p.peer_mbps_scaled || p.mbps_scaled || 0) / maxV) * plotH;
            if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
        });
        ctx.stroke();
    }

    async function refresh() {
        const data = await fetchPeersDetail(asn, hours);
        if (!data) return;
        updateSamplingChip(data.sampling);

        const live = data.live;
        const sessions = data.sessions || [];
        const name = (live && live.name) || (sessions[0] && sessions[0].name) || data.asn;
        const role = (sessions[0] && sessions[0].role) || 'peer';
        document.getElementById('pd-title').textContent = `${name} (${data.asn})`;
        document.getElementById('pd-sub').textContent = 'Mbps = janela ~10s × amostragem · sessões via SNMP BGP4-MIB';
        document.getElementById('pd-mbps').textContent = formatMbps(live ? live.mbps_scaled : (sessions[0] && sessions[0].mbps_scaled) || 0);
        const up = sessions.filter(s => s.established).length;
        document.getElementById('pd-sessions').textContent = `${up} / ${sessions.length}`;
        document.getElementById('pd-bytes').textContent = formatBytes(live ? live.bytes : (sessions[0] && sessions[0].bytes) || 0);
        document.getElementById('pd-role').textContent = role;

        drawChart(data.history || []);

        const sb = document.getElementById('pd-sessions-body');
        sb.innerHTML = sessions.map(s => `<tr class="${s.established ? '' : 'row-dim'}">
            <td><code>${s.remote_addr}</code></td>
            <td>${stateBadge(s.state_name, s.established)}</td>
            <td>${formatMbps(s.mbps_scaled != null ? s.mbps_scaled : s.mbps)}</td>
            <td>${formatUptime(s.uptime_sec)}</td>
            <td>${s.flap_count || 0}</td>
            <td>${(s.in_updates || 0).toLocaleString('pt-BR')} / ${(s.out_updates || 0).toLocaleString('pt-BR')}</td>
        </tr>`).join('') || '<tr><td colspan="6">Sem sessões BGP para este ASN</td></tr>';

        const fb = document.getElementById('pd-flows');
        const flows = data.flows || [];
        fb.innerHTML = flows.map(f => {
            const ts = f.timestamp ? new Date(f.timestamp * 1000).toLocaleTimeString('pt-BR') : '—';
            return `<tr>
                <td>${ts}</td>
                <td><code>${f.src_ip}</code> → <code>${f.dst_ip}</code></td>
                <td>${categoryBadge(f.category)}</td>
                <td>${formatBytes(f.bytes)}</td>
                <td>${directionBadge(f.direction)}</td>
            </tr>`;
        }).join('') || '<tr><td colspan="5">Nenhum flow recente</td></tr>';
    }

    document.getElementById('pd-time-filter')?.addEventListener('click', (e) => {
        const btn = e.target.closest('.tf-btn[data-h]');
        if (!btn) return;
        document.querySelectorAll('#pd-time-filter .tf-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        hours = parseInt(btn.dataset.h, 10) || 6;
        refresh();
    });

    setInterval(refresh, 3000);
    refresh();
    if (!prefersReducedMotion()) { /* noop */ }
})();
