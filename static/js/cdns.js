(function() {
    const {
        formatBytes, formatMbps, createTrafficRow, createIfaceCard,
        fetchHistory, fetchHistoryCompare, samplingFactor, tweenMbps, renderFlipList,
        spawnFlowParticle, prefersReducedMotion, fetchCDNPage, updateSamplingChip,
        exportURL, directionBadge
    } = window.Inforflow;

    const cdnColors = {
        'Cloudflare': '#f48120', 'Akamai': '#0099cc', 'Fastly': '#ff0744',
        'AWS CloudFront': '#ff9900', 'Google Cache': '#4285f4', 'CDN77': '#00b67a',
        'BunnyCDN': '#ff6600', 'Edgecast': '#6b7280', 'Limelight': '#8b5cf6',
        'Imperva': '#ef4444', 'Cachefly': '#14b8a6', 'G-Core': '#f43f5e',
        'QUIC.cloud': '#f59e0b', 'Azure CDN': '#0078d4'
    };

    const cdnShort = {
        'Cloudflare': 'CF', 'Akamai': 'AK', 'Fastly': 'FS', 'AWS CloudFront': 'CFNT',
        'Google Cache': 'GGC', 'CDN77': '77', 'BunnyCDN': 'BN', 'Edgecast': 'ED',
        'Limelight': 'LL', 'Imperva': 'IM', 'Cachefly': 'CY', 'G-Core': 'GC',
        'QUIC.cloud': 'QC', 'Azure CDN': 'AZ'
    };

    const chartColors = [
        '#f48120', '#0099cc', '#ff0744', '#ff9900', '#4285f4',
        '#00b67a', '#ff6600', '#8b5cf6', '#14b8a6', '#0078d4'
    ];

    let historyHours = 0;
    let searchQ = '';
    let chipFilter = '';
    let lastRates = [];
    let chartSeries = [];
    let chartTs = [];
    let drawProgress = 1;

    function detailURL(name) {
        return '/cdn/detail?name=' + encodeURIComponent(name || '');
    }

    function asnURL(asn) {
        return asn ? '/asn/detail?asn=' + encodeURIComponent(asn) : '/asn';
    }

    function colorFor(name) {
        return cdnColors[name] || chartColors[(name || '').length % chartColors.length];
    }

    function matchesChip(name, chip) {
        if (!chip) return true;
        const n = (name || '').toLowerCase();
        if (chip === 'other') {
            return !/cloudflare|akamai|google|aws|cloudfront/.test(n);
        }
        if (chip === 'AWS') return /aws|cloudfront/.test(n);
        if (chip === 'Google') return /google|ggc/.test(n);
        return n.includes(chip.toLowerCase());
    }

    function buildRates(stats) {
        if (stats.cdn_rates && stats.cdn_rates.length) {
            return stats.cdn_rates.map(r => ({
                name: r.name,
                asn: r.asn,
                bytes: r.bytes,
                mbps: r.mbps,
                mbps_scaled: r.mbps_scaled,
                in_mbps: r.in_mbps,
                out_mbps: r.out_mbps,
                ipv4_mbps: r.ipv4_mbps,
                ipv6_mbps: r.ipv6_mbps,
                percentage: r.percentage,
                category: 'cdn'
            }));
        }
        return [];
    }

    function filteredRates() {
        let list = lastRates.slice();
        if (chipFilter) list = list.filter(r => matchesChip(r.name, chipFilter));
        const q = searchQ.trim().toLowerCase();
        if (q) {
            list = list.filter(r =>
                (r.name || '').toLowerCase().includes(q) ||
                (r.asn || '').toLowerCase().includes(q)
            );
        }
        return list.sort((a, b) => (b.mbps_scaled || 0) - (a.mbps_scaled || 0));
    }

    function renderAlerts(data) {
        const el = document.getElementById('cdn-alerts');
        if (!el) return;
        const items = [];
        if (data.divergence_warn) {
            items.push(`<div class="alert-item alert-warning"><strong>Divergência SNMP</strong><span>${data.divergence_warn}</span></div>`);
        }
        if (data.overlap_note) {
            items.push(`<div class="alert-item alert-warning"><strong>Overlap GGC ↔ Streaming</strong><span>${data.overlap_note}</span></div>`);
        }
        if (!items.length) {
            el.innerHTML = '<div class="alert-item alert-ok">CDN e cache SNMP alinhados</div>';
            return;
        }
        el.innerHTML = items.join('');
    }

    function renderNodes(rates) {
        const el = document.getElementById('cdn-nodes');
        if (!el) return;
        const top = rates.filter(r => r.mbps_scaled > 0 || r.bytes > 0).slice(0, 10);
        el.innerHTML = top.map(r => {
            const short = cdnShort[r.name] || r.name.slice(0, 3).toUpperCase();
            const color = colorFor(r.name);
            return `<a class="cdn-node" href="${detailURL(r.name)}" data-cdn="${r.name}" title="${r.name}: ${formatMbps(r.mbps_scaled)}" style="border-color:${color}">
                <span>${short}</span>
                <small class="cdn-node-mbps">${formatMbps(r.mbps_scaled)}</small>
            </a>`;
        }).join('') || '<div class="cdn-node"><span>—</span></div>';
    }

    function renderTrafficList() {
        const list = document.getElementById('cdn-traffic-list');
        if (!list) return;
        const rates = filteredRates();
        const maxMbps = Math.max(...rates.map(r => r.mbps_scaled || 0), 1);
        if (!rates.length) {
            list.innerHTML = '<div class="dest-card"><div class="card-name">Nenhum CDN neste filtro…</div></div>';
            return;
        }
        renderFlipList(list, rates, null, (r, i) => {
            const row = createTrafficRow({
                ...r,
                name: r.asn ? `${r.name} (${r.asn})` : r.name,
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

    function renderFeeds(feeds) {
        const el = document.getElementById('cdn-feeds');
        if (!el) return;
        if (!feeds) {
            el.textContent = 'Feeds indisponíveis';
            return;
        }
        const src = (feeds.sources || []).map(s => {
            const age = s.age_sec != null ? Math.round(s.age_sec / 60) + 'm' : '—';
            const flag = s.from_cache ? 'cache' : (s.last_ok ? 'ok' : 'fail');
            return `${s.name}: ${s.prefixes || 0} pref. (${flag}, ${age})`;
        }).join('<br>');
        el.innerHTML = `<strong>${feeds.total_rules || 0} regras</strong><br>${src || '—'}`;
    }

    function topCDNKeys(hist) {
        const scores = {};
        (hist || []).forEach(h => {
            const m = h.by_cdn_mbps_scaled || h.by_cdn_mbps || {};
            Object.entries(m).forEach(([k, v]) => { scores[k] = (scores[k] || 0) + (v || 0); });
        });
        return Object.entries(scores).sort((a, b) => b[1] - a[1]).slice(0, 8).map(([k]) => k);
    }

    function seriesValue(h, k) {
        if (h.by_cdn_mbps_scaled && h.by_cdn_mbps_scaled[k] != null) return h.by_cdn_mbps_scaled[k];
        return ((h.by_cdn_mbps && h.by_cdn_mbps[k]) || 0) * (h.sampling_factor || 1);
    }

    function drawCDNChart() {
        const canvas = document.getElementById('cdn-history-chart');
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
            drawCDNChart();
            return;
        }
        drawProgress = 0;
        const start = performance.now();
        function frame(now) {
            drawProgress = Math.min(1, (now - start) / 700);
            drawCDNChart();
            if (drawProgress < 1) requestAnimationFrame(frame);
        }
        requestAnimationFrame(frame);
    }

    async function updateHistory() {
        const legend = document.getElementById('cdn-chart-legend');
        if (historyHours <= 0) {
            chartTs = [];
            chartSeries = [];
            drawCDNChart();
            if (legend) legend.innerHTML = '<span class="legend-muted">Selecione 1h / 6h / 24h para o histórico</span>';
            return;
        }
        const hist = await fetchHistory(historyHours) || [];
        const keys = topCDNKeys(hist);
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
            }).join('') || '<span class="legend-muted">Sem dados CDN no período (aguarde pontos novos)</span>';
        }
        animateDraw();
    }

    async function updateCDNs() {
        const stats = await fetchCDNPage();
        if (!stats) return;
        updateSamplingChip(stats.sampling);

        const exportLink = document.getElementById('export-csv');
        if (exportLink) exportLink.href = exportURL('cdn', 'csv');

        renderAlerts(stats);
        lastRates = buildRates(stats);

        const totalScaled = stats.total_mbps_scaled != null
            ? stats.total_mbps_scaled
            : ((stats.by_category_mbps_scaled || {}).cdn || 0);
        const eff = samplingFactor(stats);
        const top = lastRates[0];

        tweenMbps(document.getElementById('cdn-total-mbps'), totalScaled);
        document.getElementById('cdn-total-hint').textContent =
            `NF bruto: ${formatMbps((stats.by_category_mbps || {}).cdn || 0)} · fator ~${eff.toFixed(0)}×`;

        const share = stats.uplink_share_pct || 0;
        document.getElementById('cdn-uplink-share').textContent =
            share > 0 ? share.toFixed(1) + '%' : '—';

        const sn = stats.snmp;
        if (sn && sn.ok) {
            document.getElementById('cdn-snmp-uplink').textContent =
                formatMbps(sn.uplink_in_mbps) + ' / ' + formatMbps(sn.uplink_out_mbps);
        } else {
            document.getElementById('cdn-snmp-uplink').textContent = 'SNMP offline';
        }

        const hit = stats.cache_hit_pct;
        document.getElementById('cdn-cache-hit').textContent =
            hit != null && hit > 0 ? hit.toFixed(1) + '%' : '—';
        const caches = stats.cache_ifaces || [];
        const cacheTotal = (stats.cache_snmp_in_mbps || 0) + (stats.cache_snmp_out_mbps || 0);
        document.getElementById('cdn-cache-hint').textContent = caches.length
            ? `${caches.length} ifaces · ${formatMbps(cacheTotal)}`
            : 'sem iface cache';

        document.getElementById('cdn-v4v6').textContent =
            formatMbps(stats.ipv4_mbps || 0).replace(' Mbps', '') + ' / ' +
            formatMbps(stats.ipv6_mbps || 0).replace(' Mbps', '');
        document.getElementById('cdn-top-name').textContent = top
            ? `Top: ${top.name} ${formatMbps(top.mbps_scaled)}`
            : `${lastRates.filter(r => r.bytes > 0 || r.mbps_scaled > 0).length} CDNs`;

        const sub = document.getElementById('cdn-subtitle');
        if (sub) sub.textContent = `${stats.window_hint || 'Mbps estimado'} · exporter ${stats.exporter || '—'}`;
        const expLabel = document.getElementById('cdn-exporter-label');
        if (expLabel) expLabel.textContent = stats.exporter || 'exporter';

        const bytesHint = document.getElementById('cdn-bytes-hint');
        if (bytesHint && stats.bytes_hint) bytesHint.textContent = stats.bytes_hint;

        const snmpCards = document.getElementById('cdn-snmp-ifaces');
        if (snmpCards) {
            snmpCards.innerHTML = caches.length
                ? caches.slice(0, 8).map(createIfaceCard).join('')
                : '<div class="dest-card"><div class="card-name">Sem interfaces cache/CDN no SNMP</div></div>';
        }

        renderFeeds(stats.feeds);
        renderNodes(lastRates);
        renderTrafficList();

        document.querySelectorAll('.cdn-node').forEach(node => {
            const name = node.dataset.cdn;
            const r = lastRates.find(x => x.name === name);
            if (r && (r.mbps_scaled > 0 || r.bytes > 0)) {
                node.classList.add('active');
                setTimeout(() => node.classList.remove('active'), 900);
            }
        });

        const compareEl = document.getElementById('cdn-compare-hint');
        if (compareEl) {
            fetchHistoryCompare(24).then(cmp => {
                const cur = (cmp.current || []).slice(-12);
                const prev = (cmp.previous || []).slice(-12);
                if (!cur.length || !prev.length) {
                    compareEl.textContent = 'Comparativo 24h: aguardando histórico';
                    return;
                }
                const avg = (arr) => {
                    const vals = arr.map(h => (h.by_category_mbps_scaled && h.by_category_mbps_scaled.cdn)
                        || ((h.by_category_mbps && h.by_category_mbps.cdn) || 0) * (h.sampling_factor || 1));
                    return vals.reduce((s, v) => s + v, 0) / vals.length;
                };
                const a = avg(cur), b = avg(prev);
                const d = a - b;
                const pct = b > 0 ? (d / b) * 100 : 0;
                compareEl.textContent = `vs 24h: ${d >= 0 ? '+' : ''}${formatMbps(d)} (${pct >= 0 ? '+' : ''}${pct.toFixed(1)}%)`;
            }).catch(() => {});
        }

        const tbody = document.getElementById('cdn-table-body');
        if (tbody) {
            const flows = stats.flows || [];
            tbody.innerHTML = flows.map(f => {
                const name = f.direction === 'outbound' ? f.destination : f.origin;
                const asn = f.asn || f.dst_asn || '';
                return `<tr>
                    <td><a href="${detailURL(name || '')}">${name || '—'}</a></td>
                    <td>${asn ? `<a href="${asnURL(asn)}">${asn}</a>` : '—'}</td>
                    <td><code>${f.src_ip || '—'}</code> → <code>${f.dst_ip || '—'}</code></td>
                    <td>${formatBytes(f.bytes)}</td>
                    <td>${directionBadge(f.direction)}</td>
                </tr>`;
            }).join('') || '<tr><td colspan="5">Sem flows CDN no buffer recente</td></tr>';
        }
    }

    document.getElementById('cdn-time-filter')?.addEventListener('click', (e) => {
        const btn = e.target.closest('.tf-btn[data-h]');
        if (!btn) return;
        document.querySelectorAll('#cdn-time-filter .tf-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        historyHours = parseInt(btn.dataset.h, 10) || 0;
        updateHistory();
    });
    document.getElementById('cdn-chip-filter')?.addEventListener('click', (e) => {
        const btn = e.target.closest('.tf-btn[data-chip]');
        if (!btn) return;
        document.querySelectorAll('#cdn-chip-filter .tf-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        chipFilter = btn.dataset.chip || '';
        renderTrafficList();
    });
    document.getElementById('cdn-search')?.addEventListener('input', (e) => {
        searchQ = e.target.value || '';
        renderTrafficList();
    });

    setInterval(updateCDNs, 2000);
    updateCDNs();
    updateHistory();
    window.addEventListener('resize', () => drawCDNChart());
    setInterval(() => {
        if (prefersReducedMotion()) return;
        spawnFlowParticle('cdn-pipeline', 'cdn', 80);
    }, 1200);
})();
