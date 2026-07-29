(function() {
    const {
        formatBytes, formatMbps, createCard, createTrafficRow,
        fetchStats, fetchFlows, fetchHistory, fetchHistoryCompare,
        categoryBadge, directionBadge, samplingFactor, exportURL,
        tweenMbps, renderFlipList, prefersReducedMotion
    } = window.Inforflow;

    const asnColors = [
        '#0ea5e9', '#6366f1', '#10b981', '#f59e0b', '#ef4444',
        '#8b5cf6', '#06b6d4', '#f43f5e', '#84cc16', '#f97316'
    ];

    let historyHours = 0;
    let compareMode = false;
    let chartSeries = [];
    let chartTs = [];
    let drawProgress = 1;

    function shortASN(asn, name) {
        if (name && !String(name).startsWith('AS')) {
            const base = String(name).split(/[\s/]/)[0];
            return base.length > 8 ? base.slice(0, 7) : base;
        }
        return (asn || 'AS?').replace(/^AS/, '');
    }

    function drillURL(asn) {
        return '/flows?asn=' + encodeURIComponent(asn || '');
    }

    function buildList(stats) {
        return (stats.asn_breakdown || []).map(a => ({
            asn: a.asn,
            name: a.name || a.asn,
            label: `${a.name || '—'} (${a.asn})`,
            bytes: a.bytes || 0,
            flows: a.flows || 0,
            mbps: a.mbps || 0,
            mbps_scaled: a.mbps_scaled || 0,
            in_mbps: a.in_mbps || 0,
            out_mbps: a.out_mbps || 0,
            ipv4_mbps: a.ipv4_mbps || 0,
            ipv6_mbps: a.ipv6_mbps || 0,
            percentage: a.percentage || 0,
            category: a.category || 'other',
            icon: a.icon || 'peer'
        })).sort((a, b) => (b.mbps_scaled || 0) - (a.mbps_scaled || 0));
    }

    function renderNodes(list) {
        const el = document.getElementById('asn-nodes');
        if (!el) return;
        const top = list.filter(r => r.mbps_scaled > 0 || r.bytes > 0).slice(0, 14);
        const max = Math.max(...top.map(r => r.mbps_scaled || 0), 1);
        renderFlipList(el, top, null, (r, i) => {
            const color = asnColors[i % asnColors.length];
            const intensity = Math.min(1, (r.mbps_scaled || 0) / max);
            const pulse = intensity > 0.15 && !prefersReducedMotion() ? ' asn-node-pulse' : '';
            return `<a class="cdn-node${pulse}" data-flip-key="${r.asn}" href="${drillURL(r.asn)}"
                title="${r.label}: ${formatMbps(r.mbps_scaled)} — ver flows"
                style="border-color:${color};--asn-pulse:${0.6 + intensity * 1.4}s;opacity:${0.55 + intensity * 0.45}">
                <span>${shortASN(r.asn, r.name)}</span>
                <small class="cdn-node-mbps">${formatMbps(r.mbps_scaled)}</small>
            </a>`;
        });
    }

    function renderTable(list) {
        const tbody = document.getElementById('asn-table-body');
        if (!tbody) return;
        tbody.innerHTML = list.map((a, i) => `<tr class="asn-row-click" data-asn="${a.asn}" style="cursor:pointer">
            <td>${i + 1}</td>
            <td><a href="${drillURL(a.asn)}"><code>${a.asn}</code></a></td>
            <td>${a.name}</td>
            <td>${formatMbps(a.mbps_scaled)}</td>
            <td>${formatMbps(a.ipv4_mbps)}</td>
            <td>${formatMbps(a.ipv6_mbps)}</td>
            <td>${formatMbps(a.in_mbps)}</td>
            <td>${formatMbps(a.out_mbps)}</td>
            <td>${formatBytes(a.bytes)}</td>
            <td>${(a.flows || 0).toLocaleString('pt-BR')}</td>
            <td>${(a.percentage || 0).toFixed(1)}%</td>
            <td>${categoryBadge(a.category)}</td>
        </tr>`).join('') || '<tr><td colspan="12">Aguardando flows com ASN de destino…</td></tr>';
        tbody.querySelectorAll('.asn-row-click').forEach(tr => {
            tr.addEventListener('click', (e) => {
                if (e.target.closest('a')) return;
                window.location.href = drillURL(tr.dataset.asn);
            });
        });
    }

    function topASNKeys(hist) {
        const scores = {};
        (hist || []).forEach(h => {
            const m = h.by_asn_mbps_scaled || h.by_asn_mbps || {};
            Object.entries(m).forEach(([k, v]) => { scores[k] = (scores[k] || 0) + (v || 0); });
        });
        return Object.entries(scores).sort((a, b) => b[1] - a[1]).slice(0, 6).map(([k]) => k);
    }

    function drawAsnChart() {
        const canvas = document.getElementById('asn-history-chart');
        if (!canvas || !chartTs.length) return;
        const rect = canvas.getBoundingClientRect();
        const w = rect.width || 800;
        const h = 260;
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
        let maxV = 1;
        chartSeries.forEach(s => s.values.forEach(v => { if (v > maxV) maxV = v; }));
        const n = chartTs.length;
        const visible = Math.max(2, Math.floor(n * drawProgress));

        ctx.strokeStyle = 'rgba(148,163,184,0.2)';
        ctx.lineWidth = 1;
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

        chartSeries.forEach((s, si) => {
            ctx.strokeStyle = s.color;
            ctx.lineWidth = 2;
            ctx.beginPath();
            for (let i = 0; i < visible; i++) {
                const x = pad.l + (i / Math.max(n - 1, 1)) * plotW;
                const y = pad.t + plotH - (s.values[i] / maxV) * plotH;
                if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
            }
            ctx.stroke();
            if (si === 0 && s.prevValues) {
                ctx.setLineDash([4, 4]);
                ctx.globalAlpha = 0.45;
                ctx.beginPath();
                for (let i = 0; i < Math.min(visible, s.prevValues.length); i++) {
                    const x = pad.l + (i / Math.max(n - 1, 1)) * plotW;
                    const y = pad.t + plotH - (s.prevValues[i] / maxV) * plotH;
                    if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
                }
                ctx.stroke();
                ctx.setLineDash([]);
                ctx.globalAlpha = 1;
            }
        });
    }

    function animateDraw() {
        if (prefersReducedMotion()) {
            drawProgress = 1;
            drawAsnChart();
            return;
        }
        drawProgress = 0;
        const start = performance.now();
        function frame(now) {
            drawProgress = Math.min(1, (now - start) / 700);
            drawAsnChart();
            if (drawProgress < 1) requestAnimationFrame(frame);
        }
        requestAnimationFrame(frame);
    }

    async function updateHistory() {
        const legend = document.getElementById('asn-chart-legend');
        if (historyHours <= 0) {
            chartTs = [];
            chartSeries = [];
            drawAsnChart();
            if (legend) legend.innerHTML = '<span class="legend-muted">Selecione 1h / 6h / 24h para o histórico</span>';
            document.getElementById('asn-compare-delta').textContent = '—';
            document.getElementById('asn-compare-hint').textContent = 'filtro de tempo';
            return;
        }

        let hist = await fetchHistory(historyHours) || [];
        let prev = [];
        if (compareMode) {
            const cmp = await fetchHistoryCompare(historyHours);
            hist = cmp.current || hist;
            prev = cmp.previous || [];
        }
        const keys = topASNKeys(hist);
        chartTs = hist.map(h => h.ts);
        chartSeries = keys.map((k, i) => ({
            key: k,
            color: asnColors[i % asnColors.length],
            values: hist.map(h => {
                const scaled = h.by_asn_mbps_scaled;
                if (scaled && scaled[k] != null) return scaled[k];
                const raw = (h.by_asn_mbps && h.by_asn_mbps[k]) || 0;
                return raw * (h.sampling_factor || 1);
            }),
            prevValues: compareMode ? hist.map((_, idx) => {
                const p = prev[Math.min(idx, prev.length - 1)];
                if (!p) return 0;
                const scaled = p.by_asn_mbps_scaled;
                if (scaled && scaled[k] != null) return scaled[k];
                return ((p.by_asn_mbps && p.by_asn_mbps[k]) || 0) * (p.sampling_factor || 1);
            }) : null
        }));

        if (legend) {
            legend.innerHTML = keys.map((k, i) =>
                `<span class="legend-item"><i style="background:${asnColors[i % asnColors.length]}"></i>${k}</span>`
            ).join('') || '<span class="legend-muted">Sem dados ASN no período</span>';
        }

        if (compareMode && hist.length && prev.length) {
            const sum = (arr, key) => arr.reduce((s, h) => {
                const m = h.by_asn_mbps_scaled || h.by_asn_mbps || {};
                return s + (m[key] || 0);
            }, 0);
            const top = keys[0];
            if (top) {
                const curAvg = sum(hist, top) / hist.length;
                const prevAvg = sum(prev, top) / prev.length;
                const delta = curAvg - prevAvg;
                const pct = prevAvg > 0 ? (delta / prevAvg) * 100 : 0;
                document.getElementById('asn-compare-delta').textContent =
                    (delta >= 0 ? '+' : '') + formatMbps(delta);
                document.getElementById('asn-compare-hint').textContent =
                    `${top} · ${pct >= 0 ? '+' : ''}${pct.toFixed(1)}% vs período anterior`;
            }
        } else {
            document.getElementById('asn-compare-delta').textContent = '—';
            document.getElementById('asn-compare-hint').textContent = compareMode ? 'sem dados' : 'ative Comparar';
        }
        animateDraw();
    }

    async function updateASN() {
        const [stats, flows] = await Promise.all([fetchStats(), fetchFlows()]);
        if (!stats) return;

        const list = buildList(stats);
        const totalScaled = list.reduce((s, a) => s + (a.mbps_scaled || 0), 0);
        const totalBytes = list.reduce((s, a) => s + (a.bytes || 0), 0) || 1;
        const ipv4 = list.reduce((s, a) => s + (a.ipv4_mbps || 0), 0);
        const ipv6 = list.reduce((s, a) => s + (a.ipv6_mbps || 0), 0);
        const eff = samplingFactor(stats);
        const maxMbps = Math.max(...list.map(a => a.mbps_scaled || 0), 1);
        const top = list[0];

        tweenMbps(document.getElementById('asn-total-mbps'), totalScaled);
        document.getElementById('asn-total-hint').textContent =
            `NF bruto: ${formatMbps(list.reduce((s, a) => s + (a.mbps || 0), 0))} · fator ~${eff.toFixed(0)}× · ${formatBytes(totalBytes)} acumulado`;
        document.getElementById('asn-count').textContent = String(list.filter(a => a.bytes > 0 || a.mbps_scaled > 0).length);
        document.getElementById('asn-sampling-hint').textContent =
            stats.sampling ? `${(totalScaled / Math.max(stats.sampling.snmp_mbps || 1, 1) * 100).toFixed(1)}% do uplink SNMP` : '—';
        tweenMbps(document.getElementById('asn-top-mbps'), top ? top.mbps_scaled : 0);
        document.getElementById('asn-top-name').textContent = top ? `${top.name} (${top.asn})` : '—';
        tweenMbps(document.getElementById('asn-ipv4-mbps'), ipv4);
        tweenMbps(document.getElementById('asn-ipv6-mbps'), ipv6);

        const sn = stats.snmp;
        if (sn && sn.ok) {
            document.getElementById('asn-snmp-uplink').textContent =
                `${formatMbps(sn.uplink_in_mbps)} ↓ / ${formatMbps(sn.uplink_out_mbps)} ↑`;
        }

        const exportLink = document.getElementById('export-csv');
        if (exportLink) exportLink.href = exportURL('asn', 'csv');

        renderNodes(list);

        const trafficList = document.getElementById('asn-traffic-list');
        if (trafficList) {
            if (!list.length) {
                trafficList.innerHTML = '<div class="dest-card"><div class="card-name">Sem tráfego atribuído a ASN de destino ainda</div></div>';
            } else {
                renderFlipList(trafficList, list.slice(0, 25), null, (a, i) =>
                    createTrafficRow({
                        name: a.label,
                        mbps: a.mbps,
                        mbps_scaled: a.mbps_scaled,
                        in_mbps: a.in_mbps,
                        out_mbps: a.out_mbps,
                        bytes: a.bytes,
                        percentage: a.percentage,
                        category: a.category
                    }, { maxMbps, rank: i + 1, color: asnColors[i % asnColors.length], flipKey: a.asn })
                );
                trafficList.querySelectorAll('.traffic-row').forEach((row, i) => {
                    const a = list[i];
                    if (!a) return;
                    row.style.cursor = 'pointer';
                    row.title = 'Ver flows de ' + a.asn;
                    row.addEventListener('click', () => { window.location.href = drillURL(a.asn); });
                });
            }
        }

        const cards = document.getElementById('asn-cards');
        if (cards) {
            renderFlipList(cards, list.slice(0, 12), null, a =>
                `<a class="dest-card-link" data-flip-key="${a.asn}" href="${drillURL(a.asn)}">${createCard({
                    name: a.label,
                    bytes: a.bytes,
                    mbps: a.mbps,
                    mbps_scaled: a.mbps_scaled,
                    in_mbps: a.in_mbps,
                    out_mbps: a.out_mbps,
                    percentage: a.percentage,
                    category: a.category
                })}</a>`
            );
            if (!list.length) {
                cards.innerHTML = '<div class="dest-card"><div class="card-name">Aguardando classificação ASN…</div></div>';
            }
        }

        renderTable(list);

        const flowsBody = document.getElementById('asn-flows-body');
        if (flowsBody && flows) {
            const withAsn = flows.filter(f => f.dst_asn || f.asn).slice(0, 25);
            flowsBody.innerHTML = withAsn.map(f => {
                const ts = f.timestamp ? new Date(f.timestamp * 1000).toLocaleTimeString('pt-BR') : '—';
                const asn = f.dst_asn || f.asn;
                return `<tr>
                    <td>${ts}</td>
                    <td><code>${f.src_ip}</code> → <code>${f.dst_ip}</code></td>
                    <td><a href="${drillURL(asn)}">${asn || '—'}</a></td>
                    <td>${f.ip_version || '4'}</td>
                    <td>${categoryBadge(f.category)}</td>
                    <td>${formatBytes(f.bytes)}</td>
                    <td>${directionBadge(f.direction)}</td>
                </tr>`;
            }).join('') || '<tr><td colspan="7">Nenhum flow com ASN de destino no momento</td></tr>';
        }
    }

    document.getElementById('asn-time-filter')?.addEventListener('click', (e) => {
        const btn = e.target.closest('.tf-btn[data-h]');
        if (!btn) return;
        document.querySelectorAll('#asn-time-filter .tf-btn[data-h]').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        historyHours = parseInt(btn.dataset.h, 10) || 0;
        updateHistory();
    });
    document.getElementById('asn-compare-btn')?.addEventListener('click', () => {
        compareMode = !compareMode;
        document.getElementById('asn-compare-btn').classList.toggle('active', compareMode);
        updateHistory();
    });

    setInterval(updateASN, 2500);
    updateASN();
    updateHistory();
    window.addEventListener('resize', () => drawAsnChart());
})();
