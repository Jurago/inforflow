(function() {
    const {
        formatBytes, formatMbps, createTrafficRow, createIfaceCard,
        fetchHistory, samplingFactor, tweenMbps, renderFlipList, spawnFlowParticle,
        prefersReducedMotion, fetchStreamingPage, updateSamplingChip, exportURL,
        categoryBadge, directionBadge
    } = window.Inforflow;

    const streamColors = {
        'Netflix': '#e50914', 'Globo': '#0066cc', 'YouTube': '#ff0000',
        'Spotify': '#1db954', 'Twitch': '#9146ff', 'Disney+': '#0063e5',
        'HBO Max': '#b535f6', 'RTMP': '#f97316', 'Amazon Prime': '#00a8e1',
        'Paramount+': '#0064ff', 'Apple TV+': '#a2aaad', 'Apple': '#a2aaad',
        'TikTok': '#010101', 'Meta': '#0668E1', 'Instagram': '#E1306C'
    };

    const heroStyles = {
        'Netflix': 'netflix-card',
        'Globo': 'globo-card',
        'YouTube': 'youtube-card',
        'Spotify': 'spotify-card'
    };

    const chartColors = [
        '#8b5cf6', '#e50914', '#0066cc', '#ff0000', '#1db954',
        '#9146ff', '#f97316', '#00a8e1', '#a2aaad', '#06b6d4'
    ];

    const streamPositions = [
        { top: '10%', left: '70%' }, { top: '30%', left: '85%' },
        { top: '60%', left: '80%' }, { top: '80%', left: '65%' },
        { top: '75%', left: '30%' }, { top: '50%', left: '15%' },
        { top: '20%', left: '20%' }, { top: '10%', left: '45%' },
        { top: '45%', left: '50%' }, { top: '85%', left: '40%' }
    ];

    let historyHours = 0;
    let searchQ = '';
    let catFilter = '';
    let lastRates = [];
    let lastRelated = [];
    let chartSeries = [];
    let chartTs = [];
    let drawProgress = 1;

    function detailURL(name) {
        return '/streaming/detail?name=' + encodeURIComponent(name || '');
    }

    function colorFor(name) {
        return streamColors[name] || chartColors[(name || '').length % chartColors.length];
    }

    function streamingTotalScaled(stats) {
        if (stats.total_mbps_scaled != null && stats.total_mbps_scaled > 0) {
            return stats.total_mbps_scaled;
        }
        const scaled = stats.by_category_mbps_scaled || {};
        return (scaled.streaming || 0) + (scaled.netflix || 0) + (scaled.globo || 0) + (scaled.apple || 0);
    }

    function buildRates(stats) {
        if (stats.streaming_rates && stats.streaming_rates.length) {
            return stats.streaming_rates.map(r => ({
                name: r.name,
                bytes: r.bytes,
                mbps: r.mbps,
                mbps_scaled: r.mbps_scaled,
                in_mbps: r.in_mbps,
                out_mbps: r.out_mbps,
                ipv4_mbps: r.ipv4_mbps,
                ipv6_mbps: r.ipv6_mbps,
                percentage: r.percentage,
                category: r.category || 'streaming'
            }));
        }
        const breakdown = stats.streaming_breakdown || {};
        return Object.entries(breakdown)
            .filter(([, b]) => b > 0)
            .map(([name, bytes]) => ({
                name, bytes, mbps: 0, mbps_scaled: 0,
                category: 'streaming', percentage: 0
            }))
            .sort((a, b) => (b.bytes || 0) - (a.bytes || 0));
    }

    function filteredRates() {
        let list = lastRates.slice();
        if (catFilter === 'social') {
            list = (lastRelated || []).slice();
        } else if (catFilter) {
            list = list.filter(r => (r.category || '') === catFilter ||
                (catFilter === 'streaming' && !['netflix', 'globo', 'apple'].includes(r.category)));
        }
        const q = searchQ.trim().toLowerCase();
        if (q) list = list.filter(r => (r.name || '').toLowerCase().includes(q));
        return list.sort((a, b) => (b.mbps_scaled || 0) - (a.mbps_scaled || 0));
    }

    function renderAlerts(data) {
        const el = document.getElementById('stream-alerts');
        if (!el) return;
        if (data.divergence_warn) {
            el.innerHTML = `<div class="alert-item alert-warning"><strong>Divergência SNMP</strong><span>${data.divergence_warn}</span></div>`;
            return;
        }
        el.innerHTML = '<div class="alert-item alert-ok">Cache SNMP e classificação alinhados</div>';
    }

    function initStreamTargets(rates) {
        const container = document.getElementById('stream-targets');
        if (!container) return;
        const active = rates.filter(r => r.mbps_scaled > 0 || r.bytes > 0);
        container.innerHTML = '';
        active.slice(0, 10).forEach((r, i) => {
            const pos = streamPositions[i % streamPositions.length];
            const el = document.createElement('a');
            el.href = detailURL(r.name);
            el.className = 'stream-target';
            el.dataset.service = r.name;
            el.textContent = r.name;
            el.style.top = pos.top;
            el.style.left = pos.left;
            el.style.borderColor = colorFor(r.name);
            container.appendChild(el);
        });

        const rays = document.getElementById('stream-rays');
        if (rays && !rays.children.length) {
            for (let i = 0; i < 8; i++) {
                const ray = document.createElement('div');
                ray.className = 'stream-ray';
                ray.style.transform = `rotate(${i * 45}deg)`;
                ray.style.animationDelay = (i * 0.3) + 's';
                rays.appendChild(ray);
            }
        }
    }

    function renderHero(rates, totalScaled) {
        const el = document.getElementById('stream-hero-cards');
        if (!el) return;
        const top = rates.filter(r => r.mbps_scaled > 0 || r.bytes > 0).slice(0, 4);
        if (!top.length) {
            el.innerHTML = '<div class="dest-card"><div class="card-name">Aguardando tráfego de streaming…</div></div>';
            return;
        }
        el.innerHTML = top.map(r => {
            const cls = heroStyles[r.name] || 'stream-generic-card';
            const color = colorFor(r.name);
            const pct = totalScaled > 0 ? Math.min(100, (r.mbps_scaled / totalScaled) * 100) : 0;
            return `
                <a class="dest-card-link" href="${detailURL(r.name)}">
                <div class="hero-card ${cls}" style="--stream-color:${color}">
                    <div class="hero-card-bg"></div>
                    <div class="hero-card-content">
                        <span class="hero-label">${r.name}</span>
                        <span class="hero-value">${formatMbps(r.mbps_scaled)}</span>
                        <span class="hero-rate">${formatBytes(r.bytes)} acumulado · ${pct.toFixed(1)}% do streaming</span>
                        <div class="hero-bar"><div class="hero-bar-fill" style="width:${pct}%;background:${color}"></div></div>
                    </div>
                </div></a>`;
        }).join('');
    }

    function renderIoSummary(stats, totalScaled) {
        const el = document.getElementById('stream-io-summary');
        if (!el) return;
        const inM = ((stats.by_category_in_mbps || {}).streaming || 0)
            + ((stats.by_category_in_mbps || {}).netflix || 0)
            + ((stats.by_category_in_mbps || {}).globo || 0)
            + ((stats.by_category_in_mbps || {}).apple || 0);
        const outM = ((stats.by_category_out_mbps || {}).streaming || 0)
            + ((stats.by_category_out_mbps || {}).netflix || 0)
            + ((stats.by_category_out_mbps || {}).globo || 0)
            + ((stats.by_category_out_mbps || {}).apple || 0);
        el.innerHTML = `
            <div class="io-stat"><span class="io-label">Entrada (cat.)</span><span class="io-value">${formatMbps(inM)}</span></div>
            <div class="io-stat"><span class="io-label">Saída (cat.)</span><span class="io-value">${formatMbps(outM)}</span></div>
            <div class="io-stat"><span class="io-label">Total est.</span><span class="io-value">${formatMbps(totalScaled)}</span></div>
            <div class="io-stat"><span class="io-label">Cache SNMP</span><span class="io-value">${formatMbps((stats.cache_snmp_in_mbps || 0) + (stats.cache_snmp_out_mbps || 0))}</span></div>`;
    }

    function renderTrafficList() {
        const list = document.getElementById('streaming-traffic-list');
        if (!list) return;
        const rates = filteredRates();
        const maxMbps = Math.max(...rates.map(r => r.mbps_scaled || 0), 1);
        if (!rates.length) {
            list.innerHTML = '<div class="dest-card"><div class="card-name">Nenhum serviço neste filtro…</div></div>';
            return;
        }
        renderFlipList(list, rates, null, (r, i) => {
            const row = createTrafficRow({
                ...r,
                percentage: r.percentage != null ? r.percentage : 0
            }, {
                rank: i + 1,
                maxMbps,
                color: colorFor(r.name),
                flipKey: r.name
            });
            return `<a class="dest-card-link traffic-row-link" href="${detailURL(r.name)}">${row}</a>`;
        });
    }

    function topStreamKeys(hist) {
        const scores = {};
        (hist || []).forEach(h => {
            const m = h.by_streaming_mbps_scaled || h.by_streaming_mbps || {};
            Object.entries(m).forEach(([k, v]) => { scores[k] = (scores[k] || 0) + (v || 0); });
        });
        return Object.entries(scores).sort((a, b) => b[1] - a[1]).slice(0, 8).map(([k]) => k);
    }

    function seriesValue(h, k) {
        if (h.by_streaming_mbps_scaled && h.by_streaming_mbps_scaled[k] != null) {
            return h.by_streaming_mbps_scaled[k];
        }
        return ((h.by_streaming_mbps && h.by_streaming_mbps[k]) || 0) * (h.sampling_factor || 1);
    }

    function drawStreamChart() {
        const canvas = document.getElementById('stream-history-chart');
        if (!canvas || !chartTs.length) return;
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
        let maxV = 1;
        chartSeries.forEach(s => s.values.forEach(v => { if (v > maxV) maxV = v; }));
        const n = chartTs.length;
        const visible = Math.max(2, Math.floor(n * drawProgress));

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

        chartSeries.forEach(s => {
            ctx.strokeStyle = s.color;
            ctx.lineWidth = 2;
            ctx.beginPath();
            for (let i = 0; i < visible; i++) {
                const x = pad.l + (i / Math.max(n - 1, 1)) * plotW;
                const y = pad.t + plotH - (s.values[i] / maxV) * plotH;
                if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
            }
            ctx.stroke();
        });
    }

    function animateDraw() {
        if (prefersReducedMotion()) {
            drawProgress = 1;
            drawStreamChart();
            return;
        }
        drawProgress = 0;
        const start = performance.now();
        function frame(now) {
            drawProgress = Math.min(1, (now - start) / 700);
            drawStreamChart();
            if (drawProgress < 1) requestAnimationFrame(frame);
        }
        requestAnimationFrame(frame);
    }

    async function updateHistory() {
        const legend = document.getElementById('stream-chart-legend');
        if (historyHours <= 0) {
            chartTs = [];
            chartSeries = [];
            drawStreamChart();
            if (legend) legend.innerHTML = '<span class="legend-muted">Selecione 1h / 6h / 24h para o histórico</span>';
            return;
        }
        const hist = await fetchHistory(historyHours) || [];
        const keys = topStreamKeys(hist);
        chartTs = hist.map(h => h.ts);
        chartSeries = keys.map((k, i) => ({
            key: k,
            color: colorFor(k) || chartColors[i % chartColors.length],
            values: hist.map(h => seriesValue(h, k))
        }));
        if (legend) {
            legend.innerHTML = keys.map((k, i) => {
                const c = colorFor(k) || chartColors[i % chartColors.length];
                return `<a class="legend-item" href="${detailURL(k)}"><i style="background:${c}"></i>${k}</a>`;
            }).join('') || '<span class="legend-muted">Sem dados de streaming no período (aguarde pontos novos)</span>';
        }
        animateDraw();
    }

    async function updateStreaming() {
        const stats = await fetchStreamingPage();
        if (!stats) return;
        updateSamplingChip(stats.sampling);

        const exportLink = document.getElementById('export-csv');
        if (exportLink) exportLink.href = exportURL('streaming', 'csv');

        renderAlerts(stats);

        lastRates = buildRates(stats);
        lastRelated = (stats.related_social || []).map(r => ({
            name: r.name,
            bytes: r.bytes,
            mbps: r.mbps,
            mbps_scaled: r.mbps_scaled,
            percentage: r.percentage,
            category: 'social'
        }));

        const totalScaled = streamingTotalScaled(stats);
        const eff = samplingFactor(stats);

        tweenMbps(document.getElementById('stream-total-mbps'), totalScaled);
        const raw = ((stats.by_category_mbps || {}).streaming || 0)
            + ((stats.by_category_mbps || {}).netflix || 0)
            + ((stats.by_category_mbps || {}).globo || 0)
            + ((stats.by_category_mbps || {}).apple || 0);
        document.getElementById('stream-total-hint').textContent =
            `NF bruto: ${formatMbps(raw)} · fator ~${eff.toFixed(0)}×`;

        const share = stats.uplink_share_pct || 0;
        document.getElementById('stream-uplink-share').textContent =
            share > 0 ? share.toFixed(1) + '%' : '—';

        const sn = stats.snmp;
        if (sn && sn.ok) {
            document.getElementById('stream-snmp-uplink').textContent =
                formatMbps(sn.uplink_in_mbps) + ' / ' + formatMbps(sn.uplink_out_mbps);
        } else {
            document.getElementById('stream-snmp-uplink').textContent = 'SNMP offline';
        }

        const hit = stats.cache_hit_pct;
        document.getElementById('stream-cache-hit').textContent =
            hit != null && hit > 0 ? hit.toFixed(1) + '%' : '—';
        const cacheTotal = (stats.cache_snmp_in_mbps || 0) + (stats.cache_snmp_out_mbps || 0);
        const caches = stats.cache_ifaces || [];
        document.getElementById('stream-snmp-cache-label').textContent = caches.length
            ? `${caches.length} ifaces · ${formatMbps(cacheTotal)}`
            : 'sem iface cache';

        document.getElementById('stream-v4v6').textContent =
            formatMbps(stats.ipv4_mbps || 0).replace(' Mbps', '') + ' / ' +
            formatMbps(stats.ipv6_mbps || 0).replace(' Mbps', '');
        document.getElementById('stream-count').textContent =
            `${lastRates.filter(r => r.bytes > 0 || r.mbps_scaled > 0).length} serviços · Mbps`;

        const sub = document.getElementById('stream-subtitle');
        if (sub) {
            const exp = stats.exporter || '—';
            sub.textContent = `${stats.window_hint || 'Mbps estimado'} · exporter ${exp}`;
        }
        const expLabel = document.getElementById('stream-exporter-label');
        if (expLabel) expLabel.textContent = stats.exporter || 'exporter';

        const bytesHint = document.getElementById('stream-bytes-hint');
        if (bytesHint && stats.bytes_hint) {
            bytesHint.textContent = stats.bytes_hint + (stats.include_note ? ' · ' + stats.include_note : '');
        }

        const snmpCards = document.getElementById('streaming-snmp-ifaces');
        if (snmpCards) {
            snmpCards.innerHTML = caches.length
                ? caches.slice(0, 8).map(createIfaceCard).join('')
                : '<div class="dest-card"><div class="card-name">Nenhuma interface CACHE no SNMP</div></div>';
        }

        renderHero(lastRates, totalScaled);
        renderIoSummary(stats, totalScaled);
        renderTrafficList();
        initStreamTargets(lastRates);

        document.querySelectorAll('.stream-target').forEach(el => {
            const svc = el.dataset.service;
            const r = lastRates.find(x => x.name === svc);
            if (r && (r.mbps_scaled > 0 || r.bytes > 0)) {
                el.classList.add('active');
                setTimeout(() => el.classList.remove('active'), 1500);
                if (!prefersReducedMotion()) {
                    spawnFlowParticle('stream-rays', r.category || 'streaming', r.mbps_scaled);
                }
            }
        });

        const tbody = document.getElementById('streaming-flows-body');
        if (tbody) {
            const flows = stats.flows || [];
            tbody.innerHTML = flows.map(f => {
                const time = f.timestamp ? new Date(f.timestamp * 1000).toLocaleTimeString('pt-BR') : '—';
                const label = f.direction === 'outbound' ? f.destination : f.origin;
                const name = label || f.category || '—';
                return `<tr>
                    <td>${time}</td>
                    <td><a href="${detailURL(name)}">${name}</a></td>
                    <td><code>${f.src_ip || '—'}</code> → <code>${f.dst_ip || '—'}</code></td>
                    <td>${categoryBadge(f.category)}</td>
                    <td>${formatBytes(f.bytes)}</td>
                    <td>${directionBadge(f.direction)}</td>
                </tr>`;
            }).join('') || '<tr><td colspan="6">Sem flows de streaming no buffer</td></tr>';
        }
    }

    document.getElementById('stream-time-filter')?.addEventListener('click', (e) => {
        const btn = e.target.closest('.tf-btn[data-h]');
        if (!btn) return;
        document.querySelectorAll('#stream-time-filter .tf-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        historyHours = parseInt(btn.dataset.h, 10) || 0;
        updateHistory();
    });
    document.getElementById('stream-cat-filter')?.addEventListener('click', (e) => {
        const btn = e.target.closest('.tf-btn[data-cat]');
        if (!btn) return;
        document.querySelectorAll('#stream-cat-filter .tf-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        catFilter = btn.dataset.cat || '';
        renderTrafficList();
    });
    document.getElementById('stream-search')?.addEventListener('input', (e) => {
        searchQ = e.target.value || '';
        renderTrafficList();
    });

    setInterval(updateStreaming, 2000);
    updateStreaming();
    updateHistory();
    window.addEventListener('resize', () => drawStreamChart());
})();
