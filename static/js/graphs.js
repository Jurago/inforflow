(function () {
    const {
        formatMbps, formatBytes, fetchStats, fetchHistory, fetchHistoryCompare, fetchASN,
        updateSamplingChip, exportURL, samplingFactor, apiGet, apiHeaders
    } = window.Inforflow;

    let historyHours = 0;
    let chartMode = 'scaled';
    let catChartType = 'line';
    let compareMode = false;
    let seriesView = 'all';
    let asnRole = 'dest';
    let svcMode = 'cdn';
    let customFrom = null;
    let customTo = null;
    let zoomRange = null;
    let asnNameMap = {};
    let fullHist = [];
    let highlightCat = null;
    let highlightAsn = null;

    const urlInit = new URLSearchParams(window.location.search);
    if (urlInit.get('h')) historyHours = parseInt(urlInit.get('h'), 10) || 0;
    if (urlInit.get('mode') === 'raw') chartMode = 'raw';
    if (urlInit.get('compare') === '1') compareMode = true;
    if (urlInit.get('cat')) highlightCat = urlInit.get('cat');
    if (urlInit.get('asn')) highlightAsn = urlInit.get('asn');
    if (urlInit.get('view') === 'snmp') seriesView = 'snmp';
    if (urlInit.get('asnrole') === 'peer') asnRole = 'peer';
    if (urlInit.get('svc') === 'streaming') svcMode = 'streaming';

    function syncToggleUI() {
        document.querySelectorAll('#time-filter .tf-btn').forEach(b => {
            b.classList.toggle('active', !customFrom && parseInt(b.dataset.h, 10) === historyHours);
        });
        document.querySelectorAll('#chart-mode-toggle .tf-btn').forEach(b => {
            b.classList.toggle('active', b.dataset.mode === chartMode);
        });
        document.querySelectorAll('#chart-compare-toggle .tf-btn').forEach(b => {
            b.classList.toggle('active', (b.dataset.compare === '1') === compareMode);
        });
        document.querySelectorAll('#series-view-toggle .tf-btn').forEach(b => {
            b.classList.toggle('active', b.dataset.view === seriesView);
        });
        document.querySelectorAll('#asn-role-toggle .tf-btn').forEach(b => {
            b.classList.toggle('active', b.dataset.asnrole === asnRole);
        });
        document.querySelectorAll('#svc-toggle .tf-btn').forEach(b => {
            b.classList.toggle('active', b.dataset.svc === svcMode);
        });
        document.querySelectorAll('#cat-chart-type .tf-btn').forEach(b => {
            b.classList.toggle('active', b.dataset.type === catChartType);
        });
    }
    syncToggleUI();

    function persistGraphURL() {
        const q = new URLSearchParams();
        if (historyHours > 0 && !customFrom) q.set('h', String(historyHours));
        if (chartMode === 'raw') q.set('mode', 'raw');
        if (compareMode) q.set('compare', '1');
        if (highlightCat && CAT_KEYS.includes(highlightCat)) q.set('cat', highlightCat);
        if (highlightAsn) q.set('asn', highlightAsn);
        if (seriesView === 'snmp') q.set('view', 'snmp');
        if (asnRole === 'peer') q.set('asnrole', 'peer');
        if (svcMode === 'streaming') q.set('svc', 'streaming');
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
    const ASN_COLORS = ['#f59e0b', '#8b5cf6', '#06b6d4', '#ef4444', '#22c55e', '#3b82f6', '#ec4899', '#64748b'];
    const SVC_COLORS = ['#f59e0b', '#3b82f6', '#8b5cf6', '#06b6d4', '#ef4444', '#22c55e', '#ec4899', '#64748b'];

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

    function effFactor(h, fallbackEff) {
        return h.sampling_factor > 0 ? h.sampling_factor : (fallbackEff || 1);
    }

    function scaleHistoryPoint(h, field, fallbackEff) {
        const eff = effFactor(h, fallbackEff);
        if (chartMode === 'raw') {
            if (field === 'mbps') return h.mbps || 0;
            if (field === 'ipv4') return h.ipv4_mbps || 0;
            if (field === 'ipv6') return h.ipv6_mbps || 0;
            return (h.by_category_mbps && h.by_category_mbps[field]) || 0;
        }
        if (field === 'mbps') {
            return h.mbps_scaled != null ? h.mbps_scaled : (h.mbps || 0) * eff;
        }
        if (field === 'ipv4') return (h.ipv4_mbps || 0) * eff;
        if (field === 'ipv6') return (h.ipv6_mbps || 0) * eff;
        const scaled = h.by_category_mbps_scaled;
        if (scaled && scaled[field] != null) return scaled[field];
        const raw = (h.by_category_mbps && h.by_category_mbps[field]) || 0;
        return raw * eff;
    }

    function mapField(asnRole, scaled) {
        if (asnRole === 'peer') return scaled ? 'by_peer_asn_mbps_scaled' : 'by_peer_asn_mbps';
        return scaled ? 'by_asn_mbps_scaled' : 'by_asn_mbps';
    }

    function asnSeriesValue(h, k, fallbackEff) {
        const eff = effFactor(h, fallbackEff);
        if (asnRole === 'peer') {
            if (chartMode !== 'raw' && h.by_peer_asn_mbps_scaled && h.by_peer_asn_mbps_scaled[k] != null) {
                return h.by_peer_asn_mbps_scaled[k];
            }
            return ((h.by_peer_asn_mbps && h.by_peer_asn_mbps[k]) || 0) * (chartMode === 'raw' ? 1 : eff);
        }
        if (chartMode !== 'raw' && h.by_asn_mbps_scaled && h.by_asn_mbps_scaled[k] != null) {
            return h.by_asn_mbps_scaled[k];
        }
        return ((h.by_asn_mbps && h.by_asn_mbps[k]) || 0) * (chartMode === 'raw' ? 1 : eff);
    }

    function svcSeriesValue(h, k, fallbackEff) {
        const eff = effFactor(h, fallbackEff);
        const scaledField = svcMode === 'cdn' ? 'by_cdn_mbps_scaled' : 'by_streaming_mbps_scaled';
        const rawField = svcMode === 'cdn' ? 'by_cdn_mbps' : 'by_streaming_mbps';
        if (chartMode !== 'raw' && h[scaledField] && h[scaledField][k] != null) return h[scaledField][k];
        return ((h[rawField] && h[rawField][k]) || 0) * (chartMode === 'raw' ? 1 : eff);
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
            brushDragging: false,
            brushStartX: null,
            brushEndX: null,
            pad: { l: 58, r: 20, t: 20, b: 36 }
        };

        function plotMetrics() {
            const rect = canvas.getBoundingClientRect();
            const w = rect.width || canvas.clientWidth || 800;
            const h = state.opts.height || 300;
            return { w, h, plotW: w - state.pad.l - state.pad.r, plotH: h - state.pad.t - state.pad.b };
        }

        function indexAtX(clientX) {
            const { plotW } = plotMetrics();
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

        function drawBrush(ctx, plotH) {
            if (!state.opts.brush || state.brushStartX == null || state.brushEndX == null) return;
            const rect = canvas.getBoundingClientRect();
            const x0 = Math.min(state.brushStartX, state.brushEndX) - rect.left;
            const x1 = Math.max(state.brushStartX, state.brushEndX) - rect.left;
            const { plotW } = plotMetrics();
            const left = Math.max(state.pad.l, Math.min(x0, x1));
            const right = Math.min(state.pad.l + plotW, Math.max(x0, x1));
            if (right - left < 2) return;
            ctx.fillStyle = 'rgba(59,130,246,0.18)';
            ctx.fillRect(left, state.pad.t, right - left, plotH);
            ctx.strokeStyle = 'rgba(59,130,246,0.65)';
            ctx.lineWidth = 1;
            ctx.strokeRect(left, state.pad.t, right - left, plotH);
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

            if (state.hoverIdx != null && state.hoverIdx >= 0 && state.hoverIdx < n && !state.brushDragging) {
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
                    ctx.lineWidth = s.highlight ? 2.5 : 1.5;
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
                    ctx.lineWidth = s.highlight ? 3.5 : (s.id === 'nf' ? 2.5 : 2);
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

            if (state.hoverIdx != null && state.hoverIdx >= 0 && state.hoverIdx < n && !state.brushDragging) {
                series.forEach((s, si) => {
                    let v = s.data[state.hoverIdx] || 0;
                    let stackBelow = 0;
                    if (state.opts.stacked) {
                        for (let j = 0; j < si; j++) stackBelow += series[j].data[state.hoverIdx] || 0;
                    }
                    const x = xAt(state.hoverIdx, plotW, n);
                    const y = yAt(v + stackBelow, maxVal, plotH);
                    ctx.beginPath();
                    ctx.arc(x, y, s.highlight ? 6 : 5, 0, Math.PI * 2);
                    ctx.fillStyle = s.color;
                    ctx.fill();
                    ctx.strokeStyle = '#0f172a';
                    ctx.lineWidth = 2;
                    ctx.stroke();
                });
            }

            drawBrush(ctx, plotH);
        }

        function showTooltip(idx, clientX, clientY) {
            if (!tooltipEl || idx == null || state.brushDragging) {
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

        canvas.addEventListener('mousedown', (e) => {
            if (!state.opts.brush || e.button !== 0) return;
            state.brushDragging = true;
            state.brushStartX = e.clientX;
            state.brushEndX = e.clientX;
            state.hoverIdx = null;
            if (tooltipEl) tooltipEl.classList.remove('visible');
            draw();
        });

        canvas.addEventListener('mousemove', (e) => {
            if (state.brushDragging) {
                state.brushEndX = e.clientX;
                draw();
                return;
            }
            const idx = indexAtX(e.clientX);
            if (idx !== state.hoverIdx) {
                state.hoverIdx = idx;
                draw();
            }
            showTooltip(idx, e.clientX, e.clientY);
        });

        function finishBrush(e) {
            if (!state.brushDragging) return;
            state.brushDragging = false;
            const x0 = Math.min(state.brushStartX, state.brushEndX);
            const x1 = Math.max(state.brushStartX, state.brushEndX);
            state.brushStartX = null;
            state.brushEndX = null;
            draw();
            if (Math.abs(x1 - x0) > 10 && state.opts.onZoom) {
                const i0 = indexAtX(x0);
                const i1 = indexAtX(x1);
                if (i0 != null && i1 != null) state.opts.onZoom(Math.min(i0, i1), Math.max(i0, i1));
            }
        }

        canvas.addEventListener('mouseup', finishBrush);
        canvas.addEventListener('mouseleave', () => {
            if (state.brushDragging) finishBrush();
            state.hoverIdx = null;
            draw();
            if (tooltipEl) tooltipEl.classList.remove('visible');
        });

        return {
            setData(timestamps, series, setOpts) {
                state.timestamps = timestamps;
                state.series = series;
                state.hoverIdx = null;
                const animate = setOpts && setOpts.animate;
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
            setHidden(ids) {
                state.hidden = new Set(ids);
                draw();
            },
            showOnly(ids) {
                const keep = new Set(ids);
                state.hidden = new Set(state.series.map(s => s.id).filter(id => !keep.has(id)));
                draw();
            },
            setStacked(v) { state.opts.stacked = v; },
            redraw: draw
        };
    }

    function renderLegend(el, series, chart, onToggle, highlightId) {
        if (!el) return;
        el.innerHTML = series.map(s => {
            const off = chart && chart.isHidden(s.id);
            const hi = highlightId && s.id === highlightId ? ' legend-highlight' : '';
            return `<button type="button" class="legend-item${off ? ' legend-off' : ''}${hi}" data-id="${s.id}">
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

    async function loadHistory() {
        const q = new URLSearchParams();
        if (customFrom) {
            q.set('from', String(customFrom));
            q.set('to', String(customTo || Math.floor(Date.now() / 1000)));
        } else if (historyHours > 0) {
            q.set('hours', String(historyHours));
        }
        const mp = historyHours >= 168 ? 600
            : historyHours >= 72 ? 700
                : historyHours >= 24 ? 800
                    : historyHours >= 6 ? 600 : 400;
        if (historyHours > 0 || customFrom) q.set('max_points', String(mp));
        try {
            return await apiGet('/history' + (q.toString() ? '?' + q : ''));
        } catch (e) {
            if (historyHours > 0 && !customFrom) return await fetchHistory(historyHours) || [];
            return [];
        }
    }

    function sliceHist(hist) {
        if (!zoomRange || !hist.length) return hist;
        const start = Math.max(0, Math.min(zoomRange.start, hist.length - 1));
        const end = Math.max(start, Math.min(zoomRange.end, hist.length - 1));
        return hist.slice(start, end + 1);
    }

    function topMapKeys(hist, getter, limit) {
        const scores = {};
        (hist || []).forEach(h => {
            const m = getter(h) || {};
            Object.entries(m).forEach(([k, v]) => { scores[k] = (scores[k] || 0) + (v || 0); });
        });
        return Object.entries(scores).sort((a, b) => b[1] - a[1]).slice(0, limit).map(([k]) => k);
    }

    function asnLabel(k) {
        const name = asnNameMap[k];
        return name && !String(name).startsWith('AS') ? `${name} (${k})` : k;
    }

    function renderGapAlerts(nfVal, snmpAvg) {
        const gapEl = document.getElementById('g-gap');
        const gapPctEl = document.getElementById('g-gap-pct');
        const alertsEl = document.getElementById('graphs-alerts');
        const gap = Math.abs(nfVal - snmpAvg);
        const gapPct = snmpAvg > 0 ? (gap / snmpAvg) * 100 : 0;
        if (gapEl) gapEl.textContent = formatMbps(gap);
        if (gapPctEl) gapPctEl.textContent = gapPct.toFixed(1) + '% do SNMP médio';
        if (!alertsEl) return;
        if (gapPct > 30) {
            alertsEl.innerHTML = `<div class="alert-item alert-warning">
                <strong>Divergência NF × SNMP</strong>
                <span>Gap de ${formatMbps(gap)} (${gapPct.toFixed(1)}%) — acima de 30%</span>
            </div>`;
        } else {
            alertsEl.innerHTML = '<div class="alert-item alert-ok">NetFlow e SNMP alinhados (gap ≤ 30%)</div>';
        }
    }

    function exportHistCSV(hist) {
        const lines = ['ts,mbps,mbps_scaled,snmp_in,snmp_out,ipv4_mbps,ipv6_mbps,sampling_factor'];
        (hist || []).forEach(h => {
            lines.push([
                h.ts,
                h.mbps || 0,
                h.mbps_scaled != null ? h.mbps_scaled : h.mbps || 0,
                h.snmp_in_mbps || 0,
                h.snmp_out_mbps || 0,
                h.ipv4_mbps || 0,
                h.ipv6_mbps || 0,
                h.sampling_factor || 1
            ].join(','));
        });
        const blob = new Blob([lines.join('\n')], { type: 'text/csv' });
        const a = document.createElement('a');
        a.href = URL.createObjectURL(blob);
        a.download = 'inforflow-history.csv';
        a.click();
        URL.revokeObjectURL(a.href);
    }

    async function exportCSV() {
        if (fullHist && fullHist.length) {
            exportHistCSV(fullHist);
            return;
        }
        const q = new URLSearchParams({ kind: 'history', format: 'csv' });
        if (historyHours > 0) q.set('hours', String(historyHours));
        const res = await fetch(`${exportURL('history', 'csv').split('?')[0]}?${q}`, { headers: apiHeaders() });
        if (!res.ok) return;
        const blob = await res.blob();
        const a = document.createElement('a');
        a.href = URL.createObjectURL(blob);
        a.download = 'inforflow-history.csv';
        a.click();
        URL.revokeObjectURL(a.href);
    }

    function exportPNG() {
        const canvas = document.getElementById('chart-total');
        if (!canvas) return;
        const a = document.createElement('a');
        a.download = 'inforflow-total.png';
        a.href = canvas.toDataURL('image/png');
        a.click();
    }

    const chartTotal = createInteractiveChart(
        document.getElementById('chart-total'),
        document.getElementById('tooltip-total'),
        {
            height: 300,
            brush: true,
            onZoom(i0, i1) {
                const offset = zoomRange ? zoomRange.start : 0;
                zoomRange = { start: offset + i0, end: offset + i1 };
                renderCharts(fullHist, window._graphsEff || 1);
            }
        }
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
    const chartSvc = createInteractiveChart(
        document.getElementById('chart-svc'),
        document.getElementById('tooltip-svc'),
        { height: 260 }
    );

    function renderCharts(hist, eff) {
        const display = sliceHist(hist);
        const timestamps = display.map(h => h.ts);

        const totalSeries = [];
        if (seriesView !== 'snmp') {
            totalSeries.push({
                id: 'nf',
                label: chartMode === 'scaled' ? 'NetFlow estimado' : 'NetFlow bruto',
                color: '#3b82f6',
                fill: true,
                data: display.map(h => scaleHistoryPoint(h, 'mbps', eff))
            });
        }
        totalSeries.push(
            {
                id: 'snmp-in',
                label: 'SNMP In',
                color: '#10b981',
                data: display.map(h => h.snmp_in_mbps || 0)
            },
            {
                id: 'snmp-out',
                label: 'SNMP Out',
                color: '#06b6d4',
                data: display.map(h => h.snmp_out_mbps || 0)
            }
        );
        if (seriesView !== 'snmp') {
            totalSeries.push(
                {
                    id: 'ipv4',
                    label: 'IPv4',
                    color: '#a78bfa',
                    data: display.map(h => scaleHistoryPoint(h, 'ipv4', eff))
                },
                {
                    id: 'ipv6',
                    label: 'IPv6',
                    color: '#fb7185',
                    data: display.map(h => scaleHistoryPoint(h, 'ipv6', eff))
                }
            );
        }

        chartTotal.setData(timestamps, totalSeries, { animate: true });
        renderLegend(document.getElementById('legend-total'), totalSeries, chartTotal, (id) => chartTotal.toggleSeries(id));

        const catSeries = CAT_KEYS.map(k => ({
            id: k,
            label: CAT_LABELS[k] || k,
            color: CAT_COLORS[k] || '#64748b',
            highlight: highlightCat === k,
            data: display.map(h => scaleHistoryPoint(h, k, eff))
        })).filter(s => s.data.some(v => v > 0.05));

        chartCategories.setStacked(catChartType === 'stacked');
        chartCategories.setData(timestamps, catSeries, { animate: true });
        if (highlightCat && CAT_KEYS.includes(highlightCat) && catSeries.some(s => s.id === highlightCat)) {
            chartCategories.showOnly([highlightCat]);
        }
        renderLegend(
            document.getElementById('legend-categories'),
            catSeries,
            chartCategories,
            (id) => chartCategories.toggleSeries(id),
            highlightCat
        );

        const asnField = mapField(asnRole, chartMode !== 'raw');
        const asnKeys = topMapKeys(display, h => h[asnField] || h[mapField(asnRole, false)], 8);
        if (highlightAsn && !asnKeys.includes(highlightAsn)) asnKeys.unshift(highlightAsn);
        const asnSeries = asnKeys.slice(0, 8).map((k, i) => ({
            id: k,
            label: asnLabel(k),
            color: ASN_COLORS[i % ASN_COLORS.length],
            highlight: highlightAsn === k,
            data: display.map(h => asnSeriesValue(h, k, eff))
        }));
        chartAsn.setData(timestamps, asnSeries, { animate: true });
        renderLegend(
            document.getElementById('legend-asn'),
            asnSeries,
            chartAsn,
            (id) => chartAsn.toggleSeries(id),
            highlightAsn
        );

        const svcScaledField = svcMode === 'cdn' ? 'by_cdn_mbps_scaled' : 'by_streaming_mbps_scaled';
        const svcRawField = svcMode === 'cdn' ? 'by_cdn_mbps' : 'by_streaming_mbps';
        const svcKeys = topMapKeys(display, h => h[svcScaledField] || h[svcRawField], 8);
        const svcSeries = svcKeys.map((k, i) => ({
            id: k,
            label: k,
            color: SVC_COLORS[i % SVC_COLORS.length],
            data: display.map(h => svcSeriesValue(h, k, eff))
        }));
        chartSvc.setData(timestamps, svcSeries, { animate: true });
        renderLegend(document.getElementById('legend-svc'), svcSeries, chartSvc, (id) => chartSvc.toggleSeries(id));

        return { display, totalSeries, catSeries, asnSeries, svcSeries };
    }

    async function update() {
        const [stats, asnData] = await Promise.all([fetchStats(), fetchASN().catch(() => null)]);
        if (!stats) return;

        if (asnData && asnData.names) asnNameMap = asnData.names;

        const eff = samplingFactor(stats);
        window._graphsEff = eff;
        updateSamplingChip(stats.sampling);

        const nfDisplay = chartMode === 'scaled'
            ? (stats.mbps_scaled || stats.mbps * eff)
            : stats.mbps;
        const snmp = stats.snmp || {};
        const snmpAvg = ((snmp.uplink_in_mbps || 0) + (snmp.uplink_out_mbps || 0)) / 2;

        document.getElementById('g-nf-mbps').textContent = formatMbps(nfDisplay);
        document.getElementById('g-classified').textContent =
            (stats.classified_pct || 0).toFixed(1) + '% classificado · NF bruto ' + formatMbps(stats.mbps);
        document.getElementById('g-snmp-in').textContent = formatMbps(snmp.uplink_in_mbps);
        document.getElementById('g-snmp-out').textContent = formatMbps(snmp.uplink_out_mbps);
        document.getElementById('g-v4v6').textContent =
            formatMbps(stats.ipv4_mbps || 0).replace(' Mbps', '') + ' / ' +
            formatMbps(stats.ipv6_mbps || 0).replace(' Mbps', '');
        document.getElementById('g-sampling').textContent = '~' + eff.toFixed(0) + '×';
        renderGapAlerts(nfDisplay, snmpAvg);

        const cats = Object.keys(stats.by_category || {}).filter(c => (stats.by_category[c] || 0) > 0);
        const catCountEl = document.getElementById('g-cat-count');
        if (catCountEl) catCountEl.textContent = cats.length + ' categorias ativas';

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

        let hist = await loadHistory();
        if ((!hist || hist.length === 0) && stats) {
            hist = [{
                ts: Math.floor(Date.now() / 1000),
                mbps: stats.mbps,
                mbps_scaled: stats.mbps_scaled,
                by_category_mbps: stats.by_category_mbps,
                by_category_mbps_scaled: stats.by_category_mbps_scaled,
                snmp_in_mbps: snmp.uplink_in_mbps,
                snmp_out_mbps: snmp.uplink_out_mbps,
                ipv4_mbps: stats.ipv4_mbps,
                ipv6_mbps: stats.ipv6_mbps,
                sampling_factor: eff
            }];
        }

        if (zoomRange && hist.length) {
            if (zoomRange.end >= hist.length) zoomRange.end = hist.length - 1;
            if (zoomRange.start > zoomRange.end) zoomRange = null;
        }

        fullHist = hist;
        const rendered = renderCharts(hist, eff);

        if (compareMode && (historyHours > 0 || customFrom) && rendered.display.length) {
            const cmpHours = historyHours > 0 ? historyHours : 24;
            const cmp = await fetchHistoryCompare(cmpHours);
            const prev = cmp.previous || [];
            if (prev.length) {
                const totalSeries = rendered.totalSeries.slice();
                totalSeries.push({
                    id: 'prev-nf',
                    label: 'Período anterior (NF est.)',
                    color: '#94a3b8',
                    data: rendered.display.map((_, i) => {
                        const p = prev[Math.min(i, prev.length - 1)];
                        return p ? scaleHistoryPoint(p, 'mbps', eff) : 0;
                    })
                });
                chartTotal.setData(rendered.display.map(h => h.ts), totalSeries, { animate: false });
                renderLegend(document.getElementById('legend-total'), totalSeries, chartTotal, (id) => chartTotal.toggleSeries(id));
            }
        }
    }

    document.getElementById('time-filter')?.addEventListener('click', (e) => {
        const btn = e.target.closest('[data-h]');
        if (!btn) return;
        historyHours = parseInt(btn.dataset.h, 10) || 0;
        customFrom = null;
        customTo = null;
        zoomRange = null;
        syncToggleUI();
        persistGraphURL();
        update();
    });

    document.getElementById('chart-mode-toggle')?.addEventListener('click', (e) => {
        const btn = e.target.closest('[data-mode]');
        if (!btn) return;
        chartMode = btn.dataset.mode;
        syncToggleUI();
        persistGraphURL();
        update();
    });

    document.getElementById('cat-chart-type')?.addEventListener('click', (e) => {
        const btn = e.target.closest('[data-type]');
        if (!btn) return;
        catChartType = btn.dataset.type;
        syncToggleUI();
        update();
    });

    document.getElementById('chart-compare-toggle')?.addEventListener('click', (e) => {
        const btn = e.target.closest('[data-compare]');
        if (!btn) return;
        compareMode = btn.dataset.compare === '1';
        syncToggleUI();
        persistGraphURL();
        update();
    });

    document.getElementById('series-view-toggle')?.addEventListener('click', (e) => {
        const btn = e.target.closest('[data-view]');
        if (!btn) return;
        seriesView = btn.dataset.view;
        syncToggleUI();
        persistGraphURL();
        update();
    });

    document.getElementById('asn-role-toggle')?.addEventListener('click', (e) => {
        const btn = e.target.closest('[data-asnrole]');
        if (!btn) return;
        asnRole = btn.dataset.asnrole;
        syncToggleUI();
        persistGraphURL();
        update();
    });

    document.getElementById('svc-toggle')?.addEventListener('click', (e) => {
        const btn = e.target.closest('[data-svc]');
        if (!btn) return;
        svcMode = btn.dataset.svc;
        syncToggleUI();
        persistGraphURL();
        update();
    });

    document.getElementById('btn-range-apply')?.addEventListener('click', () => {
        const fromEl = document.getElementById('range-from');
        const toEl = document.getElementById('range-to');
        if (!fromEl || !fromEl.value) return;
        customFrom = Math.floor(new Date(fromEl.value).getTime() / 1000);
        customTo = toEl && toEl.value ? Math.floor(new Date(toEl.value).getTime() / 1000) : null;
        historyHours = 0;
        zoomRange = null;
        document.querySelectorAll('#time-filter .tf-btn').forEach(b => b.classList.remove('active'));
        persistGraphURL();
        update();
    });

    document.getElementById('btn-zoom-reset')?.addEventListener('click', () => {
        zoomRange = null;
        if (fullHist.length) renderCharts(fullHist, window._graphsEff || 1);
    });

    document.getElementById('btn-export-csv')?.addEventListener('click', () => exportCSV());
    document.getElementById('btn-export-png')?.addEventListener('click', () => exportPNG());

    setInterval(update, 5000);
    update();
    window.addEventListener('resize', () => {
        chartTotal.redraw();
        chartCategories.redraw();
        chartAsn.redraw();
        chartSvc.redraw();
    });
})();
