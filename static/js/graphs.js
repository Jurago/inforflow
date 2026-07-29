(function () {
    const { formatMbps, formatBytes, fetchStats, fetchHistory, fetchHistoryCompare } = window.Inforflow;

    let historyHours = 0;
    let chartMode = 'scaled'; // scaled | raw
    let catChartType = 'line'; // line | stacked
    let compareMode = false;

    const urlInit = new URLSearchParams(window.location.search);
    if (urlInit.get('h')) historyHours = parseInt(urlInit.get('h'), 10) || 0;
    if (urlInit.get('mode') === 'raw') chartMode = 'raw';
    if (urlInit.get('compare') === '1') compareMode = true;
    document.querySelectorAll('#time-filter .tf-btn').forEach(b => {
        b.classList.toggle('active', parseInt(b.dataset.h, 10) === historyHours);
    });
    document.querySelectorAll('#chart-mode-toggle .tf-btn').forEach(b => {
        b.classList.toggle('active', b.dataset.mode === chartMode);
    });
    document.querySelectorAll('#chart-compare-toggle .tf-btn').forEach(b => {
        b.classList.toggle('active', (b.dataset.compare === '1') === compareMode);
    });

    function persistGraphURL() {
        const q = new URLSearchParams();
        if (historyHours > 0) q.set('h', String(historyHours));
        if (chartMode === 'raw') q.set('mode', 'raw');
        if (compareMode) q.set('compare', '1');
        const s = q.toString();
        history.replaceState(null, '', s ? `${window.location.pathname}?${s}` : window.location.pathname);
    }

    const CAT_COLORS = {
        cdn: '#f59e0b', netflix: '#e50914', globo: '#0066cc',
        streaming: '#8b5cf6', peer: '#06b6d4', other: '#64748b',
        social: '#1877f2', gaming: '#22c55e', dns: '#a3e635',
        cloud: '#f97316', apple: '#94a3b8'
    };
    const CAT_LABELS = {
        cdn: 'CDN', netflix: 'Netflix', globo: 'Globo', streaming: 'Streaming',
        peer: 'Peers', other: 'Outros', social: 'Social', gaming: 'Games',
        dns: 'DNS', cloud: 'Cloud', apple: 'Apple'
    };
    const CAT_KEYS = ['social', 'cdn', 'streaming', 'netflix', 'globo', 'gaming', 'cloud', 'apple', 'dns', 'peer', 'other'];

    function formatAxis(v) {
        if (v >= 1000) return (v / 1000).toFixed(1) + ' G';
        if (v >= 100) return v.toFixed(0);
        if (v >= 10) return v.toFixed(1);
        return v.toFixed(2);
    }

    function formatTime(ts, spanSec) {
        const d = new Date(ts * 1000);
        if (spanSec > 86400) {
            return d.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' }) + ' ' +
                d.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' });
        }
        return d.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
    }

    function scaleHistoryPoint(h, field, fallbackEff) {
        const eff = h.sampling_factor > 0 ? h.sampling_factor : (fallbackEff || 1);
        if (chartMode === 'raw') {
            if (field === 'mbps') return h.mbps || 0;
            return (h.by_category_mbps && h.by_category_mbps[field]) || 0;
        }
        if (field === 'mbps') {
            return h.mbps_scaled != null ? h.mbps_scaled : (h.mbps || 0) * eff;
        }
        const scaled = h.by_category_mbps_scaled;
        if (scaled && scaled[field] != null) return scaled[field];
        const raw = (h.by_category_mbps && h.by_category_mbps[field]) || 0;
        return raw * eff;
    }

    function createInteractiveChart(canvas, tooltipEl, opts) {
        const state = {
            canvas,
            tooltipEl,
            opts: opts || {},
            timestamps: [],
            series: [],
            hidden: new Set(),
            hoverIdx: null,
            drawProgress: 1,
            pad: { l: 58, r: 20, t: 20, b: 36 }
        };

        function plotMetrics() {
            const rect = canvas.getBoundingClientRect();
            const w = rect.width || canvas.clientWidth || 800;
            const h = state.opts.height || 300;
            return { w, h, plotW: w - state.pad.l - state.pad.r, plotH: h - state.pad.t - state.pad.b };
        }

        function indexAtX(clientX) {
            const { w, plotW } = plotMetrics();
            const rect = canvas.getBoundingClientRect();
            const x = clientX - rect.left;
            const n = state.timestamps.length;
            if (n < 2 || x < state.pad.l || x > state.pad.l + plotW) return null;
            const ratio = (x - state.pad.l) / plotW;
            return Math.max(0, Math.min(n - 1, Math.round(ratio * (n - 1))));
        }

        function xAt(i, plotW, n) {
            return state.pad.l + (i / Math.max(n - 1, 1)) * plotW;
        }

        function yAt(v, maxY, plotH) {
            return state.pad.t + plotH - (v / maxY) * plotH;
        }

        function visibleSeries() {
            return state.series.filter(s => !state.hidden.has(s.id));
        }

        function maxY() {
            let max = 0;
            visibleSeries().forEach(s => {
                s.data.forEach(v => { if (v > max) max = v; });
            });
            if (state.opts.stacked && visibleSeries().length) {
                const n = state.timestamps.length;
                for (let i = 0; i < n; i++) {
                    let sum = 0;
                    visibleSeries().forEach(s => { sum += s.data[i] || 0; });
                    if (sum > max) max = sum;
                }
            }
            return Math.max(max * 1.12, 1);
        }

        function draw() {
            const ctx = canvas.getContext('2d');
            const dpr = window.devicePixelRatio || 1;
            const { w, h, plotW, plotH } = plotMetrics();
            canvas.width = w * dpr;
            canvas.height = h * dpr;
            ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
            ctx.clearRect(0, 0, w, h);

            const series = visibleSeries();
            const n = state.timestamps.length;
            if (n === 0) {
                ctx.fillStyle = '#64748b';
                ctx.font = '13px DM Sans, sans-serif';
                ctx.fillText('Aguardando dados históricos…', state.pad.l, state.pad.t + 40);
                return;
            }

            const maxVal = maxY();
            const spanSec = n > 1 ? state.timestamps[n - 1] - state.timestamps[0] : 0;

            // grid + Y labels
            ctx.strokeStyle = 'rgba(42,53,72,0.85)';
            ctx.fillStyle = '#64748b';
            ctx.font = '11px IBM Plex Mono, monospace';
            ctx.lineWidth = 1;
            for (let i = 0; i <= 5; i++) {
                const y = state.pad.t + (plotH * i) / 5;
                const val = maxVal * (1 - i / 5);
                ctx.beginPath();
                ctx.moveTo(state.pad.l, y);
                ctx.lineTo(state.pad.l + plotW, y);
                ctx.stroke();
                ctx.fillText(formatAxis(val), 6, y + 4);
            }

            // X labels
            const xSteps = Math.min(6, n);
            for (let i = 0; i < xSteps; i++) {
                const idx = Math.round((i / Math.max(xSteps - 1, 1)) * (n - 1));
                const x = xAt(idx, plotW, n);
                ctx.fillStyle = '#64748b';
                ctx.font = '10px IBM Plex Mono, monospace';
                ctx.textAlign = 'center';
                ctx.fillText(formatTime(state.timestamps[idx], spanSec), x, h - 10);
            }
            ctx.textAlign = 'left';

            // crosshair
            if (state.hoverIdx != null && state.hoverIdx >= 0 && state.hoverIdx < n) {
                const hx = xAt(state.hoverIdx, plotW, n);
                ctx.strokeStyle = 'rgba(148,163,184,0.45)';
                ctx.lineWidth = 1;
                ctx.setLineDash([4, 4]);
                ctx.beginPath();
                ctx.moveTo(hx, state.pad.t);
                ctx.lineTo(hx, state.pad.t + plotH);
                ctx.stroke();
                ctx.setLineDash([]);
            }

            if (state.opts.stacked && series.length) {
                // stacked area
                for (let si = series.length - 1; si >= 0; si--) {
                    const s = series[si];
                    ctx.beginPath();
                    s.data.forEach((v, i) => {
                        let stackBelow = 0;
                        for (let j = 0; j < si; j++) stackBelow += series[j].data[i] || 0;
                        const x = xAt(i, plotW, n);
                        const y = yAt(v + stackBelow, maxVal, plotH);
                        if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
                    });
                    for (let i = n - 1; i >= 0; i--) {
                        let stackBelow = 0;
                        for (let j = 0; j < si; j++) stackBelow += series[j].data[i] || 0;
                        const x = xAt(i, plotW, n);
                        const y = yAt(stackBelow, maxVal, plotH);
                        ctx.lineTo(x, y);
                    }
                    ctx.closePath();
                    ctx.fillStyle = s.color + '55';
                    ctx.fill();
                    ctx.strokeStyle = s.color;
                    ctx.lineWidth = 1.5;
                    ctx.beginPath();
                    s.data.forEach((v, i) => {
                        let stackBelow = 0;
                        for (let j = 0; j < si; j++) stackBelow += series[j].data[i] || 0;
                        const x = xAt(i, plotW, n);
                        const y = yAt(v + stackBelow, maxVal, plotH);
                        if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
                    });
                    ctx.stroke();
                }
            } else {
                const visibleN = Math.max(1, Math.floor(n * (state.drawProgress == null ? 1 : state.drawProgress)));
                series.forEach(s => {
                    if (s.data.length < 1) return;
                    ctx.strokeStyle = s.color;
                    ctx.lineWidth = s.id === 'nf' ? 2.5 : 2;
                    ctx.beginPath();
                    for (let i = 0; i < visibleN; i++) {
                        const v = s.data[i];
                        const x = xAt(i, plotW, n);
                        const y = yAt(v, maxVal, plotH);
                        if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
                    }
                    ctx.stroke();
                    if (s.fill && state.drawProgress >= 1) {
                        ctx.lineTo(xAt(n - 1, plotW, n), state.pad.t + plotH);
                        ctx.lineTo(xAt(0, plotW, n), state.pad.t + plotH);
                        ctx.closePath();
                        ctx.fillStyle = s.color + '18';
                        ctx.fill();
                    }
                });
            }

            // hover dots
            if (state.hoverIdx != null && state.hoverIdx >= 0 && state.hoverIdx < n) {
                series.forEach((s, si) => {
                    let v = s.data[state.hoverIdx] || 0;
                    let stackBelow = 0;
                    if (state.opts.stacked) {
                        for (let j = 0; j < si; j++) stackBelow += series[j].data[state.hoverIdx] || 0;
                    }
                    const x = xAt(state.hoverIdx, plotW, n);
                    const y = yAt(v + stackBelow, maxVal, plotH);
                    ctx.beginPath();
                    ctx.arc(x, y, 5, 0, Math.PI * 2);
                    ctx.fillStyle = s.color;
                    ctx.fill();
                    ctx.strokeStyle = '#0f172a';
                    ctx.lineWidth = 2;
                    ctx.stroke();
                });
            }
        }

        function showTooltip(idx, clientX, clientY) {
            if (!tooltipEl || idx == null) {
                if (tooltipEl) tooltipEl.classList.remove('visible');
                return;
            }
            const n = state.timestamps.length;
            if (idx < 0 || idx >= n) return;
            const spanSec = n > 1 ? state.timestamps[n - 1] - state.timestamps[0] : 0;
            const rows = visibleSeries()
                .map(s => {
                    const v = s.data[idx] || 0;
                    return `<div class="tt-row"><span class="tt-dot" style="background:${s.color}"></span><span class="tt-label">${s.label}</span><span class="tt-val">${formatMbps(v)}</span></div>`;
                }).join('');
            tooltipEl.innerHTML = `
                <div class="tt-time">${formatTime(state.timestamps[idx], spanSec)}</div>
                ${rows}`;
            tooltipEl.classList.add('visible');

            const wrap = canvas.parentElement;
            const wrapRect = wrap.getBoundingClientRect();
            let left = clientX - wrapRect.left + 14;
            let top = clientY - wrapRect.top - 10;
            const tw = tooltipEl.offsetWidth || 180;
            if (left + tw > wrapRect.width - 8) left = clientX - wrapRect.left - tw - 14;
            if (top < 8) top = 8;
            tooltipEl.style.left = left + 'px';
            tooltipEl.style.top = top + 'px';
        }

        canvas.addEventListener('mousemove', (e) => {
            const idx = indexAtX(e.clientX);
            if (idx !== state.hoverIdx) {
                state.hoverIdx = idx;
                draw();
            }
            showTooltip(idx, e.clientX, e.clientY);
        });
        canvas.addEventListener('mouseleave', () => {
            state.hoverIdx = null;
            draw();
            if (tooltipEl) tooltipEl.classList.remove('visible');
        });

        return {
            setData(timestamps, series, opts) {
                state.timestamps = timestamps;
                state.series = series;
                state.hoverIdx = null;
                const animate = opts && opts.animate;
                const reduce = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
                if (!animate || reduce) {
                    state.drawProgress = 1;
                    draw();
                    return;
                }
                state.drawProgress = 0;
                const start = performance.now();
                function frame(now) {
                    state.drawProgress = Math.min(1, (now - start) / 750);
                    draw();
                    if (state.drawProgress < 1) requestAnimationFrame(frame);
                }
                requestAnimationFrame(frame);
            },
            toggleSeries(id) {
                if (state.hidden.has(id)) state.hidden.delete(id);
                else state.hidden.add(id);
                draw();
            },
            isHidden(id) { return state.hidden.has(id); },
            setStacked(v) { state.opts.stacked = v; },
            redraw: draw
        };
    }

    function renderLegend(el, series, chart, onToggle) {
        if (!el) return;
        el.innerHTML = series.map(s => {
            const off = chart && chart.isHidden(s.id);
            return `<button type="button" class="legend-item${off ? ' legend-off' : ''}" data-id="${s.id}">
                <i style="background:${s.color}"></i>${s.label}
            </button>`;
        }).join('');
        el.querySelectorAll('.legend-item').forEach(btn => {
            btn.addEventListener('click', () => {
                const id = btn.dataset.id;
                if (onToggle) onToggle(id);
                btn.classList.toggle('legend-off');
            });
        });
    }

    function drawPie(canvas, tooltipEl, items) {
        const ctx = canvas.getContext('2d');
        const dpr = window.devicePixelRatio || 1;
        const wrap = canvas.parentElement;
        const size = Math.min(wrap ? wrap.clientWidth : 320, 320);
        canvas.width = size * dpr;
        canvas.height = size * dpr;
        ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
        ctx.clearRect(0, 0, size, size);

        const filtered = items.filter(i => i.value > 0);
        const total = filtered.reduce((a, b) => a + b.value, 0) || 1;
        const cx = size / 2, cy = size / 2, r = size * 0.34;
        let angle = -Math.PI / 2;
        const slices = [];

        filtered.forEach(it => {
            const slice = (it.value / total) * Math.PI * 2;
            ctx.beginPath();
            ctx.moveTo(cx, cy);
            ctx.arc(cx, cy, r, angle, angle + slice);
            ctx.closePath();
            ctx.fillStyle = it.color;
            ctx.fill();
            ctx.strokeStyle = '#0f172a';
            ctx.lineWidth = 2;
            ctx.stroke();
            slices.push({ ...it, start: angle, end: angle + slice, pct: it.value / total * 100 });
            angle += slice;
        });

        // center hole donut
        ctx.beginPath();
        ctx.arc(cx, cy, r * 0.55, 0, Math.PI * 2);
        ctx.fillStyle = '#0f172a';
        ctx.fill();
        ctx.fillStyle = '#e2e8f0';
        ctx.font = 'bold 14px IBM Plex Mono, monospace';
        ctx.textAlign = 'center';
        ctx.fillText(formatMbps(filtered.reduce((s, i) => s + (i.mbps || 0), 0)), cx, cy + 2);
        ctx.font = '10px DM Sans, sans-serif';
        ctx.fillStyle = '#64748b';
        ctx.fillText('total est.', cx, cy + 16);
        ctx.textAlign = 'left';

        canvas.onmousemove = (e) => {
            const rect = canvas.getBoundingClientRect();
            const x = e.clientX - rect.left - cx;
            const y = e.clientY - rect.top - cy;
            let a = Math.atan2(y, x);
            if (a < -Math.PI / 2) a += Math.PI * 2;
            const hit = slices.find(s => a >= s.start && a <= s.end);
            if (hit && tooltipEl) {
                tooltipEl.innerHTML = `<div class="tt-time">${hit.label}</div>
                    <div class="tt-row"><span class="tt-val">${formatMbps(hit.mbps || 0)}</span></div>
                    <div class="tt-row"><span class="tt-label">${hit.pct.toFixed(1)}% · ${formatBytes(hit.value)}</span></div>`;
                tooltipEl.classList.add('visible');
                const wr = wrap.getBoundingClientRect();
                tooltipEl.style.left = (e.clientX - wr.left + 12) + 'px';
                tooltipEl.style.top = (e.clientY - wr.top - 8) + 'px';
            } else if (tooltipEl) {
                tooltipEl.classList.remove('visible');
            }
        };
        canvas.onmouseleave = () => { if (tooltipEl) tooltipEl.classList.remove('visible'); };
    }

    const chartTotal = createInteractiveChart(
        document.getElementById('chart-total'),
        document.getElementById('tooltip-total'),
        { height: 300 }
    );
    const chartCategories = createInteractiveChart(
        document.getElementById('chart-categories'),
        document.getElementById('tooltip-categories'),
        { height: 340, stacked: false }
    );
    const chartAsn = createInteractiveChart(
        document.getElementById('chart-asn'),
        document.getElementById('tooltip-asn'),
        { height: 280 }
    );

    function downsampleHistory(hist, maxPoints) {
        if (!hist || hist.length <= maxPoints) return hist || [];
        const step = Math.ceil(hist.length / maxPoints);
        const out = [];
        for (let i = 0; i < hist.length; i += step) out.push(hist[i]);
        const last = hist[hist.length - 1];
        if (out[out.length - 1] !== last) out.push(last);
        return out;
    }

    async function update() {
        const stats = await fetchStats();
        if (!stats) return;

        const eff = (stats.sampling && stats.sampling.effective) || 1;
        const nfDisplay = chartMode === 'scaled'
            ? (stats.mbps_scaled || stats.mbps * eff)
            : stats.mbps;

        document.getElementById('g-nf-mbps').textContent = formatMbps(nfDisplay);
        document.getElementById('g-classified').textContent =
            (stats.classified_pct || 0).toFixed(1) + '% classificado · NF bruto ' + formatMbps(stats.mbps);
        const snmp = stats.snmp || {};
        document.getElementById('g-snmp-in').textContent = formatMbps(snmp.uplink_in_mbps);
        document.getElementById('g-snmp-out').textContent = formatMbps(snmp.uplink_out_mbps);
        document.getElementById('g-sampling').textContent = '~' + eff.toFixed(0) + '×';
        const cats = Object.keys(stats.by_category || {}).filter(c => (stats.by_category[c] || 0) > 0);
        document.getElementById('g-cat-count').textContent = cats.length + ' categorias ativas';

        const scaledByCat = stats.by_category_mbps_scaled || stats.by_category_mbps || {};
        const pieItems = (stats.consumption || []).map(c => ({
            label: c.label || c.category,
            value: c.bytes || 0,
            mbps: c.mbps_scaled != null ? c.mbps_scaled : (c.mbps || 0) * eff,
            color: CAT_COLORS[c.category] || '#64748b'
        })).filter(i => i.value > 0);
        drawPie(document.getElementById('chart-pie'), document.getElementById('tooltip-pie'), pieItems);

        const barsHost = document.getElementById('graph-bars');
        if (barsHost) {
            const total = stats.total_bytes || 1;
            const catEntries = Object.entries(stats.by_category || {}).sort((a, b) => b[1] - a[1]);
            barsHost.innerHTML = catEntries.map(([cat, bytes]) => {
                const pct = bytes / total * 100;
                const mbps = scaledByCat[cat] != null ? formatMbps(scaledByCat[cat]) : '—';
                return `<div class="category-bar-item">
                    <span class="category-bar-label">${CAT_LABELS[cat] || cat}</span>
                    <div class="category-bar-track">
                        <div class="category-bar-fill cat-fill-${cat}" style="width:${pct}%">${pct.toFixed(1)}%</div>
                    </div>
                    <span class="category-bar-value">${mbps} · ${formatBytes(bytes)}</span>
                </div>`;
            }).join('');
        }

        const compareEl = document.getElementById('compare-bars');
        if (compareEl) {
            const snmpAvg = ((snmp.uplink_in_mbps || 0) + (snmp.uplink_out_mbps || 0)) / 2;
            const nfVal = stats.mbps_scaled || stats.mbps * eff;
            const max = Math.max(nfVal, snmpAvg, snmp.uplink_in_mbps || 0, snmp.uplink_out_mbps || 0, 1);
            const items = [
                { label: 'NetFlow est.', val: nfVal, color: '#3b82f6' },
                { label: 'SNMP médio', val: snmpAvg, color: '#10b981' },
                { label: 'SNMP In', val: snmp.uplink_in_mbps || 0, color: '#34d399' },
                { label: 'SNMP Out', val: snmp.uplink_out_mbps || 0, color: '#06b6d4' }
            ];
            compareEl.innerHTML = items.map(it => `
                <div class="compare-row">
                    <span class="compare-label">${it.label}</span>
                    <div class="compare-track"><div class="compare-fill" style="width:${(it.val / max * 100).toFixed(1)}%;background:${it.color}"></div></div>
                    <span class="compare-val">${formatMbps(it.val)}</span>
                </div>`).join('');
        }

        const history = await fetchHistory(historyHours > 0 ? historyHours : null);
        let hist = history || [];
        if (hist.length === 0 && stats) {
            hist = [{
                ts: Math.floor(Date.now() / 1000),
                mbps: stats.mbps,
                mbps_scaled: stats.mbps_scaled,
                by_category_mbps: stats.by_category_mbps,
                by_category_mbps_scaled: stats.by_category_mbps_scaled,
                snmp_in_mbps: snmp.uplink_in_mbps,
                snmp_out_mbps: snmp.uplink_out_mbps,
                sampling_factor: eff
            }];
        }

        hist = downsampleHistory(hist, historyHours >= 24 ? 800 : historyHours >= 6 ? 600 : 400);

        const timestamps = hist.map(h => h.ts);

        const totalSeries = [
            {
                id: 'nf',
                label: chartMode === 'scaled' ? 'NetFlow estimado' : 'NetFlow bruto',
                color: '#3b82f6',
                fill: true,
                data: hist.map(h => scaleHistoryPoint(h, 'mbps', eff))
            },
            {
                id: 'snmp-in',
                label: 'SNMP In',
                color: '#10b981',
                data: hist.map(h => h.snmp_in_mbps || 0)
            },
            {
                id: 'snmp-out',
                label: 'SNMP Out',
                color: '#06b6d4',
                data: hist.map(h => h.snmp_out_mbps || 0)
            }
        ];
        if (compareMode && historyHours > 0) {
            const cmp = await fetchHistoryCompare(historyHours);
            const prev = cmp.previous || [];
            if (prev.length) {
                totalSeries.push({
                    id: 'prev-nf',
                    label: 'Período anterior (NF est.)',
                    color: '#94a3b8',
                    data: hist.map((_, i) => {
                        const p = prev[Math.min(i, prev.length - 1)];
                        return p ? scaleHistoryPoint(p, 'mbps', eff) : 0;
                    })
                });
            }
        }
        chartTotal.setData(timestamps, totalSeries, { animate: true });
        renderLegend(document.getElementById('legend-total'), totalSeries, chartTotal, (id) => chartTotal.toggleSeries(id));

        const catSeries = CAT_KEYS.map(k => ({
            id: k,
            label: CAT_LABELS[k] || k,
            color: CAT_COLORS[k] || '#64748b',
            data: hist.map(h => scaleHistoryPoint(h, k, eff))
        })).filter(s => s.data.some(v => v > 0.05));

        chartCategories.setStacked(catChartType === 'stacked');
        chartCategories.setData(timestamps, catSeries, { animate: true });
        renderLegend(document.getElementById('legend-categories'), catSeries, chartCategories, (id) => chartCategories.toggleSeries(id));

        const asnKeys = new Set();
        hist.forEach(h => {
            const m = h.by_asn_mbps_scaled || h.by_asn_mbps || {};
            Object.keys(m).forEach(k => { if (m[k] > 0.05) asnKeys.add(k); });
        });
        const topAsns = [...asnKeys].slice(0, 8);
        const asnColors = ['#f59e0b', '#8b5cf6', '#06b6d4', '#ef4444', '#22c55e', '#3b82f6', '#ec4899', '#64748b'];
        const asnSeries = topAsns.map((k, i) => ({
            id: k,
            label: k,
            color: asnColors[i % asnColors.length],
            data: hist.map(h => {
                const m = h.by_asn_mbps_scaled || h.by_asn_mbps || {};
                return m[k] || 0;
            })
        }));
        chartAsn.setData(timestamps, asnSeries, { animate: true });
        renderLegend(document.getElementById('legend-asn'), asnSeries, chartAsn, (id) => chartAsn.toggleSeries(id));
    }

    document.getElementById('time-filter')?.addEventListener('click', (e) => {
        const btn = e.target.closest('[data-h]');
        if (!btn) return;
        historyHours = parseInt(btn.dataset.h, 10) || 0;
        document.querySelectorAll('#time-filter .tf-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        persistGraphURL();
        update();
    });

    document.getElementById('chart-mode-toggle')?.addEventListener('click', (e) => {
        const btn = e.target.closest('[data-mode]');
        if (!btn) return;
        chartMode = btn.dataset.mode;
        document.querySelectorAll('#chart-mode-toggle .tf-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        persistGraphURL();
        update();
    });

    document.getElementById('cat-chart-type')?.addEventListener('click', (e) => {
        const btn = e.target.closest('[data-type]');
        if (!btn) return;
        catChartType = btn.dataset.type;
        document.querySelectorAll('#cat-chart-type .tf-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        update();
    });

    document.getElementById('chart-compare-toggle')?.addEventListener('click', (e) => {
        const btn = e.target.closest('[data-compare]');
        if (!btn) return;
        compareMode = btn.dataset.compare === '1';
        document.querySelectorAll('#chart-compare-toggle .tf-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        persistGraphURL();
        update();
    });

    setInterval(update, 5000);
    update();
    window.addEventListener('resize', () => {
        chartTotal.redraw();
        chartCategories.redraw();
        chartAsn.redraw();
    });
})();
