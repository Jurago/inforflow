(function () {
    const {
        formatBytes, formatMbps, fetchRouterDetail, categoryBadge, directionBadge,
        updateSamplingChip, exportURL
    } = window.Inforflow;

    const params = new URLSearchParams(window.location.search);
    const ifindex = parseInt(params.get('ifindex') || '0', 10);
    const name = params.get('name') || '';
    let hours = 1;

    if (!ifindex && !name) {
        window.location.href = '/router';
        return;
    }

    function drawChart(hist) {
        const canvas = document.getElementById('rd-chart');
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
            maxV = Math.max(maxV, p.in_mbps || 0, p.out_mbps || 0);
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
        if (!pts.length) {
            ctx.fillStyle = '#64748b';
            ctx.font = '13px DM Sans, sans-serif';
            ctx.fillText('Aguardando histórico SNMP desta iface…', pad.l, pad.t + 40);
            return;
        }
        [['#10b981', 'in_mbps'], ['#06b6d4', 'out_mbps']].forEach(([color, key]) => {
            ctx.strokeStyle = color;
            ctx.lineWidth = 2;
            ctx.beginPath();
            pts.forEach((p, i) => {
                const x = pad.l + (i / Math.max(n - 1, 1)) * plotW;
                const y = pad.t + plotH - ((p[key] || 0) / maxV) * plotH;
                if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
            });
            ctx.stroke();
        });
    }

    async function refresh() {
        const data = await fetchRouterDetail(ifindex, hours, name);
        if (!data) return;
        updateSamplingChip(data.sampling);
        const exportLink = document.getElementById('export-csv');
        if (exportLink) exportLink.href = exportURL('router', 'csv');

        const iface = data.iface;
        if (!iface) {
            document.getElementById('rd-title').textContent = 'Interface não encontrada';
            return;
        }
        document.getElementById('rd-title').textContent =
            `${iface.alias || iface.name} (if ${iface.index})`;
        document.getElementById('rd-sub').textContent =
            `${iface.name} · ${data.host || ''}`;
        document.getElementById('rd-snmp').textContent =
            formatMbps(iface.in_mbps) + ' / ' + formatMbps(iface.out_mbps);
        const util = Math.max(iface.in_util_pct || 0, iface.out_util_pct || 0);
        document.getElementById('rd-util').textContent = 'util ' + util.toFixed(2) + '%';
        document.getElementById('rd-nf').textContent = formatMbps(data.nf_mbps_scaled || 0);
        document.getElementById('rd-speed').textContent = (iface.speed_mbps || 0) + ' Mbps';
        document.getElementById('rd-role').textContent = iface.role || '—';
        document.getElementById('rd-status').textContent = iface.oper_status === 1 ? 'up' : 'down';
        document.getElementById('rd-host').textContent = data.host || '—';

        const alerts = document.getElementById('rd-alerts');
        if (alerts) {
            const items = [];
            if (data.divergence_warn) {
                items.push(`<div class="alert-item alert-warning"><strong>Divergência</strong><span>${data.divergence_warn}</span></div>`);
            }
            if (util >= 80) {
                items.push(`<div class="alert-item alert-warning"><strong>Util alta</strong><span>${util.toFixed(1)}%</span></div>`);
            }
            alerts.innerHTML = items.length
                ? items.join('')
                : '<div class="alert-item alert-ok">Iface ok</div>';
        }

        drawChart(data.history || []);

        const fb = document.getElementById('rd-flows');
        fb.innerHTML = (data.flows || []).map(f => {
            const ts = f.timestamp ? new Date(f.timestamp * 1000).toLocaleTimeString('pt-BR') : '—';
            return `<tr>
                <td>${ts}</td>
                <td><code>${f.src_ip}</code> → <code>${f.dst_ip}</code></td>
                <td>${categoryBadge(f.category)}</td>
                <td>${formatBytes(f.bytes)}</td>
                <td>${directionBadge(f.direction)}</td>
            </tr>`;
        }).join('') || '<tr><td colspan="5">Nenhum flow com este ifIndex no buffer</td></tr>';
    }

    document.getElementById('rd-time-filter')?.addEventListener('click', (e) => {
        const btn = e.target.closest('.tf-btn[data-h]');
        if (!btn) return;
        document.querySelectorAll('#rd-time-filter .tf-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        hours = parseInt(btn.dataset.h, 10) || 1;
        refresh();
    });

    setInterval(refresh, 5000);
    refresh();
})();
