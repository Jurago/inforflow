(function() {
    const {
        formatBytes, formatMbps, createCard, createIfaceCard,
        fetchPeersPage, fetchHistory, directionBadge, categoryBadge,
        updateSamplingChip, exportURL, prefersReducedMotion, tweenMbps
    } = window.Inforflow;

    const peerColors = [
        '#0ea5e9', '#6366f1', '#10b981', '#f59e0b', '#ef4444',
        '#8b5cf6', '#06b6d4', '#f43f5e', '#84cc16', '#f97316'
    ];

    const roleLabel = {
        ix: 'IX', content: 'Conteúdo', transit: 'Trânsito',
        regional: 'Regional', local: 'Local', private: 'Privado'
    };

    let historyHours = 0;
    let viewMode = 'session'; // session | asn
    let searchQ = '';
    let roleFilter = '';
    let stateFilter = '';
    let lastPeers = [];
    let chartSeries = [];
    let chartTs = [];
    let drawProgress = 1;
    let nameMap = {};

    function detailURL(asn) {
        return '/peers/detail?asn=' + encodeURIComponent(asn || '');
    }

    function stateBadge(stateName, established) {
        const cls = established ? 'inbound' : (stateName === 'idle' ? 'other' : 'outbound');
        const label = established ? 'established' : (stateName || '—');
        return `<span class="badge badge-${cls}">${label}</span>`;
    }

    function shortLabel(name, asn) {
        if (!name) return (asn || '?').replace('AS', '');
        const base = name.split(' ')[0];
        return base.length > 6 ? base.slice(0, 5) : base;
    }

    function formatUptime(sec) {
        if (!sec || sec < 0) return '—';
        if (sec < 60) return sec + 's';
        if (sec < 3600) return Math.floor(sec / 60) + 'm';
        if (sec < 86400) return (sec / 3600).toFixed(1) + 'h';
        return (sec / 86400).toFixed(1) + 'd';
    }

    function renderAlerts(data) {
        const el = document.getElementById('peers-alerts');
        if (!el) return;
        const items = [];
        const down = data.down_peers || [];
        if (down.length) {
            items.push(`<div class="alert-item alert-warning"><strong>${down.length} sessão(ões) BGP down</strong>
                <span>${down.slice(0, 6).map(p => `${p.name || p.asn} (${p.state_name})`).join(' · ')}</span></div>`);
        }
        if (data.divergence_warn) {
            items.push(`<div class="alert-item alert-warning"><strong>Divergência SNMP</strong><span>${data.divergence_warn}</span></div>`);
        }
        if (!items.length) {
            el.innerHTML = '<div class="alert-item alert-ok">Todas as sessões monitoradas ok</div>';
            return;
        }
        el.innerHTML = items.join('');
    }

    function renderOrbit(peers) {
        const orbit = document.getElementById('peer-orbit');
        if (!orbit) return;

        const established = (peers || []).filter(p => p.established);
        const uniq = [];
        const seen = new Set();
        for (const p of established.sort((a, b) => (b.mbps_scaled || b.mbps || 0) - (a.mbps_scaled || a.mbps || 0))) {
            const key = p.remote_as || p.asn;
            if (seen.has(key)) continue;
            seen.add(key);
            uniq.push(p);
            if (uniq.length >= 14) break;
        }

        const maxMbps = Math.max(...uniq.map(p => p.mbps_scaled || p.mbps || 0), 0.01);
        orbit.innerHTML = '';

        const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
        svg.classList.add('peer-rays');
        svg.setAttribute('viewBox', '0 0 400 400');
        svg.style.cssText = 'position:absolute;inset:0;width:100%;height:100%;pointer-events:none;overflow:visible';
        orbit.appendChild(svg);

        uniq.forEach((p, i) => {
            const angle = (i / Math.max(uniq.length, 1)) * 360 - 90;
            const radius = 130 + (i % 3) * 36;
            const x = Math.cos(angle * Math.PI / 180) * radius;
            const y = Math.sin(angle * Math.PI / 180) * radius;
            const mbps = p.mbps_scaled || p.mbps || 0;
            const active = mbps > 0.05;
            const thickness = Math.max(1, Math.min(6, (mbps / maxMbps) * 6));

            const line = document.createElementNS('http://www.w3.org/2000/svg', 'line');
            line.setAttribute('x1', '200');
            line.setAttribute('y1', '200');
            line.setAttribute('x2', String(200 + x));
            line.setAttribute('y2', String(200 + y));
            line.setAttribute('stroke', active ? '#06b6d4' : '#334155');
            line.setAttribute('stroke-width', String(thickness));
            line.setAttribute('stroke-opacity', active ? '0.55' : '0.2');
            if (active) line.classList.add('peer-ray-active');
            svg.appendChild(line);

            const node = document.createElement('a');
            node.href = detailURL(p.asn);
            node.className = 'peer-node-orbit' + (active ? ' active' : ' idle');
            node.textContent = shortLabel(p.name, p.asn);
            node.style.left = `calc(50% + ${x}px - 24px)`;
            node.style.top = `calc(50% + ${y}px - 24px)`;
            if (!active) node.style.opacity = '0.4';
            node.title = `${p.name} ${p.asn}\n${p.remote_addr}\n${formatMbps(mbps)} est.`;
            orbit.appendChild(node);
        });
    }

    function aggregateByASN(peers) {
        const m = {};
        (peers || []).forEach(p => {
            const key = p.asn || ('AS' + p.remote_as);
            if (!m[key]) {
                m[key] = {
                    asn: key,
                    name: p.name,
                    role: p.role,
                    remote_addr: p.remote_addr,
                    established: false,
                    sessions: 0,
                    up: 0,
                    mbps: 0,
                    mbps_scaled: 0,
                    bytes: p.bytes || 0,
                    flows: p.flows || 0,
                    in_updates: 0,
                    out_updates: 0,
                    uptime_sec: 0,
                    flap_count: 0,
                    state_name: 'idle'
                };
            }
            const a = m[key];
            a.sessions++;
            a.in_updates += p.in_updates || 0;
            a.out_updates += p.out_updates || 0;
            a.flap_count += p.flap_count || 0;
            if ((p.mbps_scaled || 0) > a.mbps_scaled) a.mbps_scaled = p.mbps_scaled || 0;
            if ((p.mbps || 0) > a.mbps) a.mbps = p.mbps || 0;
            if ((p.bytes || 0) > a.bytes) a.bytes = p.bytes || 0;
            if (p.established) {
                a.established = true;
                a.up++;
                a.state_name = 'established';
                if ((p.uptime_sec || 0) > a.uptime_sec) a.uptime_sec = p.uptime_sec;
            }
        });
        return Object.values(m);
    }

    function filteredRows() {
        let list = viewMode === 'asn' ? aggregateByASN(lastPeers) : (lastPeers || []).slice();
        const q = searchQ.trim().toLowerCase();
        if (roleFilter) list = list.filter(p => p.role === roleFilter);
        if (stateFilter === 'up') list = list.filter(p => p.established);
        if (stateFilter === 'down') list = list.filter(p => !p.established);
        if (q) {
            list = list.filter(p =>
                (p.asn || '').toLowerCase().includes(q) ||
                (p.name || '').toLowerCase().includes(q) ||
                (p.remote_addr || '').toLowerCase().includes(q) ||
                (p.role || '').toLowerCase().includes(q)
            );
        }
        return list.sort((a, b) => {
            if (a.established !== b.established) return a.established ? -1 : 1;
            return (b.mbps_scaled || b.mbps || 0) - (a.mbps_scaled || a.mbps || 0);
        });
    }

    function renderBgpTable() {
        const tbody = document.getElementById('bgp-table-body');
        if (!tbody) return;
        const list = filteredRows();
        tbody.innerHTML = list.map(p => {
            const asn = p.asn || '—';
            const left = viewMode === 'asn'
                ? `<a href="${detailURL(asn)}"><code>${asn}</code></a> · ${p.up || 0}/${p.sessions || 0} sessões`
                : `<a href="${detailURL(asn)}"><code>${p.remote_addr}</code></a><br><small>${asn}</small>`;
            return `<tr class="asn-row-click ${p.established ? '' : 'row-dim'}" data-asn="${asn}" style="cursor:pointer">
                <td>${left}</td>
                <td>${p.name || '—'}</td>
                <td>${stateBadge(p.state_name, p.established)}</td>
                <td><span class="badge badge-peer">${roleLabel[p.role] || p.role || '—'}</span></td>
                <td>${formatMbps(p.mbps_scaled != null ? p.mbps_scaled : p.mbps)}</td>
                <td>${formatBytes(p.bytes)}</td>
                <td>${formatUptime(p.uptime_sec)}</td>
                <td>${p.flap_count || 0}</td>
                <td>${(p.in_updates || 0).toLocaleString('pt-BR')} / ${(p.out_updates || 0).toLocaleString('pt-BR')}</td>
            </tr>`;
        }).join('') || '<tr><td colspan="9">Nenhum peer correspondente</td></tr>';
        tbody.querySelectorAll('.asn-row-click').forEach(tr => {
            tr.addEventListener('click', (e) => {
                if (e.target.closest('a')) return;
                window.location.href = detailURL(tr.dataset.asn);
            });
        });
    }

    function topPeerKeys(hist) {
        const scores = {};
        (hist || []).forEach(h => {
            const m = h.by_peer_asn_mbps_scaled || h.by_peer_asn_mbps || {};
            Object.entries(m).forEach(([k, v]) => { scores[k] = (scores[k] || 0) + (v || 0); });
        });
        return Object.entries(scores).sort((a, b) => b[1] - a[1]).slice(0, 6).map(([k]) => k);
    }

    function seriesValue(h, k) {
        if (h.by_peer_asn_mbps_scaled && h.by_peer_asn_mbps_scaled[k] != null) return h.by_peer_asn_mbps_scaled[k];
        return ((h.by_peer_asn_mbps && h.by_peer_asn_mbps[k]) || 0) * (h.sampling_factor || 1);
    }

    function drawPeerChart() {
        const canvas = document.getElementById('peer-history-chart');
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
            drawPeerChart();
            return;
        }
        drawProgress = 0;
        const start = performance.now();
        function frame(now) {
            drawProgress = Math.min(1, (now - start) / 700);
            drawPeerChart();
            if (drawProgress < 1) requestAnimationFrame(frame);
        }
        requestAnimationFrame(frame);
    }

    async function updateHistory() {
        const legend = document.getElementById('peer-chart-legend');
        if (historyHours <= 0) {
            chartTs = [];
            chartSeries = [];
            drawPeerChart();
            if (legend) legend.innerHTML = '<span class="legend-muted">Selecione 1h / 6h / 24h para o histórico</span>';
            return;
        }
        const hist = await fetchHistory(historyHours) || [];
        const keys = topPeerKeys(hist);
        chartTs = hist.map(h => h.ts);
        chartSeries = keys.map((k, i) => ({
            key: k,
            color: peerColors[i % peerColors.length],
            values: hist.map(h => seriesValue(h, k))
        }));
        if (legend) {
            legend.innerHTML = keys.map((k, i) => {
                const name = nameMap[k] || k;
                return `<span class="legend-item"><i style="background:${peerColors[i % peerColors.length]}"></i>${name}</span>`;
            }).join('') || '<span class="legend-muted">Sem dados peer no período</span>';
        }
        animateDraw();
    }

    async function updatePeers() {
        const data = await fetchPeersPage();
        if (!data) return;
        updateSamplingChip(data.sampling);

        const exportLink = document.getElementById('export-csv');
        if (exportLink) exportLink.href = exportURL('peers', 'csv');

        renderAlerts(data);

        const snap = data.bgp;
        lastPeers = (snap && snap.peers) || [];
        nameMap = {};
        lastPeers.forEach(p => { if (p.asn) nameMap[p.asn] = p.name || p.asn; });
        (data.peer_asn_breakdown || []).forEach(a => { nameMap[a.asn] = a.name || a.asn; });

        if (snap) {
            document.getElementById('peer-established').textContent =
                `${snap.established || 0} / ${snap.total || 0}`;
            document.getElementById('peer-total').textContent = String(snap.total || 0);
            document.getElementById('peer-down-hint').textContent =
                (snap.down || 0) > 0 ? `${snap.down} down` : 'todas up';
            const localEl = document.getElementById('bgp-local-as');
            if (localEl) localEl.textContent = snap.local_asn || 'AS—';
            document.getElementById('bgp-local-hint').textContent = snap.local_asn || 'AS local';
            renderOrbit(snap.peers);
        }

        const peerMbps = data.peer_mbps_scaled || 0;
        tweenMbps(document.getElementById('peer-traffic'), peerMbps);
        const snmpAvg = ((data.snmp_peer_mbps_in || 0) + (data.snmp_peer_mbps_out || 0)) / 2;
        document.getElementById('peer-traffic-hint').textContent = snmpAvg > 0
            ? `SNMP IX/transit ~${formatMbps(snmpAvg)} · ${(peerMbps / Math.max(snmpAvg, 1) * 100).toFixed(0)}%`
            : (data.window_hint || 'Mbps estimado');

        const ixLabel = document.getElementById('peer-ix-label');
        if (ixLabel) ixLabel.textContent = `${data.ix_name || 'IX'} (AS${data.ix_asn || '—'})`;
        tweenMbps(document.getElementById('peer-ixbr'), data.ix_mbps_scaled || 0);
        document.getElementById('peer-ix-hint').textContent = 'Mbps estimado · 10s';

        const breakdown = document.getElementById('peer-breakdown');
        if (breakdown) {
            const items = (data.peer_asn_breakdown || []).length
                ? data.peer_asn_breakdown.map(a => ({
                    name: `${a.name} (${a.asn})`,
                    bytes: a.bytes,
                    mbps: a.mbps,
                    mbps_scaled: a.mbps_scaled,
                    percentage: a.percentage,
                    category: 'peer'
                }))
                : (data.peer_breakdown || []);
            breakdown.innerHTML = items.length
                ? items.slice(0, 12).map(o => {
                    const asnMatch = (o.name || '').match(/AS\d+/);
                    const asn = asnMatch ? asnMatch[0] : '';
                    return `<a class="dest-card-link" href="${detailURL(asn)}">${createCard(o)}</a>`;
                }).join('')
                : '<div class="dest-card"><div class="card-name">Sem tráfego atribuído a ASN BGP ainda</div></div>';
        }

        const snmpBox = document.getElementById('peer-snmp-ifaces');
        if (snmpBox) {
            const ifaces = data.snmp_peer_ifaces || [];
            snmpBox.innerHTML = ifaces.length
                ? ifaces.slice(0, 8).map(createIfaceCard).join('')
                : '<div class="dest-card"><div class="card-name">Sem interfaces IX/transit no SNMP</div></div>';
        }

        renderBgpTable();

        const tbody = document.getElementById('peer-table-body');
        if (tbody) {
            const peerFlows = data.flows || [];
            tbody.innerHTML = peerFlows.map(f => {
                const peer = f.peer_name
                    ? `${f.peer_name} (${f.peer_asn || '—'})`
                    : (f.peer_asn || '—');
                const asn = f.peer_asn || f.asn || '';
                return `<tr>
                    <td><a href="${detailURL(asn)}">${peer}</a></td>
                    <td><a href="${detailURL(asn)}">${asn || '—'}</a></td>
                    <td><code>${f.src_ip}</code> → <code>${f.dst_ip}</code></td>
                    <td>${categoryBadge(f.category)}</td>
                    <td>${formatBytes(f.bytes)}</td>
                    <td>${directionBadge(f.direction)}</td>
                </tr>`;
            }).join('') || '<tr><td colspan="6">Nenhum flow associado a peer BGP no momento</td></tr>';
        }
    }

    document.getElementById('peer-time-filter')?.addEventListener('click', (e) => {
        const btn = e.target.closest('.tf-btn[data-h]');
        if (!btn) return;
        document.querySelectorAll('#peer-time-filter .tf-btn[data-h]').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        historyHours = parseInt(btn.dataset.h, 10) || 0;
        updateHistory();
    });
    document.getElementById('peer-view-session')?.addEventListener('click', () => {
        viewMode = 'session';
        document.getElementById('peer-view-session').classList.add('active');
        document.getElementById('peer-view-asn').classList.remove('active');
        renderBgpTable();
    });
    document.getElementById('peer-view-asn')?.addEventListener('click', () => {
        viewMode = 'asn';
        document.getElementById('peer-view-asn').classList.add('active');
        document.getElementById('peer-view-session').classList.remove('active');
        renderBgpTable();
    });
    document.getElementById('peer-search')?.addEventListener('input', (e) => {
        searchQ = e.target.value || '';
        renderBgpTable();
    });
    document.getElementById('peer-role-filter')?.addEventListener('change', (e) => {
        roleFilter = e.target.value || '';
        renderBgpTable();
    });
    document.getElementById('peer-state-filter')?.addEventListener('change', (e) => {
        stateFilter = e.target.value || '';
        renderBgpTable();
    });

    setInterval(updatePeers, 2500);
    updatePeers();
    updateHistory();
    window.addEventListener('resize', () => drawPeerChart());
})();
