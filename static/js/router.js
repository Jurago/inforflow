(function () {
    const {
        formatMbps, createIfaceCard, fetchRouterPage, fetchHistory,
        updateSamplingChip, exportURL
    } = window.Inforflow;

    let historyHours = 0;
    let searchQ = '';
    let roleFilter = '';
    let stateFilter = '';
    let lastIfaces = [];
    let nfRoles = {};
    let chartTs = [];
    let chartSeries = [];
    let drawProgress = 1;
    let pollMs = 5000;

    const roleColors = {
        uplink: '#3b82f6', ix: '#06b6d4', cache: '#f59e0b',
        bras: '#8b5cf6', cgnat: '#22c55e', other: '#64748b'
    };

    function detailURL(idx) {
        return '/router/detail?ifindex=' + encodeURIComponent(idx || '');
    }

    function roleBadge(role) {
        return `<span class="badge badge-${role === 'ix' ? 'peer' : role === 'cache' ? 'cdn' : 'other'}">${role || '—'}</span>`;
    }

    function renderAlerts(data) {
        const el = document.getElementById('router-alerts');
        if (!el) return;
        const items = [];
        if (!data.ok) {
            items.push(`<div class="alert-item alert-critical"><strong>SNMP offline</strong><span>${data.error || 'sem resposta'}</span></div>`);
        }
        if (data.divergence_warn) {
            items.push(`<div class="alert-item alert-warning"><strong>Divergência NF×SNMP</strong><span>${data.divergence_warn}</span></div>`);
        }
        (data.high_util || []).slice(0, 5).forEach(i => {
            const u = Math.max(i.in_util_pct || 0, i.out_util_pct || 0);
            items.push(`<div class="alert-item alert-warning"><strong>Util ${u.toFixed(0)}%</strong><span><a href="${detailURL(i.index)}">${i.alias || i.name}</a></span></div>`);
        });
        (data.alerts || []).forEach(a => {
            items.push(`<div class="alert-item alert-${a.severity || 'warning'}"><strong>${a.title}</strong><span>${a.detail || ''}</span></div>`);
        });
        if (!items.length) {
            el.innerHTML = '<div class="alert-item alert-ok">SNMP ok · sem util crítica</div>';
            return;
        }
        el.innerHTML = items.join('');
    }

    function filteredIfaces() {
        let list = lastIfaces.slice();
        if (roleFilter) {
            list = list.filter(i => (i.role || 'other') === roleFilter ||
                (roleFilter === 'other' && !i.role));
        }
        if (stateFilter === 'up') list = list.filter(i => i.oper_status === 1);
        if (stateFilter === 'down') list = list.filter(i => i.oper_status !== 1);
        const q = searchQ.trim().toLowerCase();
        if (q) {
            list = list.filter(i =>
                (i.name || '').toLowerCase().includes(q) ||
                (i.alias || '').toLowerCase().includes(q) ||
                String(i.index).includes(q)
            );
        }
        return list.sort((a, b) => {
            const ua = Math.max(a.in_util_pct || 0, a.out_util_pct || 0);
            const ub = Math.max(b.in_util_pct || 0, b.out_util_pct || 0);
            if (ub !== ua) return ub - ua;
            return (b.in_mbps + b.out_mbps) - (a.in_mbps + a.out_mbps);
        });
    }

    function renderTable() {
        const tbody = document.getElementById('snmp-iface-body');
        if (!tbody) return;
        const rows = filteredIfaces().slice(0, 80);
        tbody.innerHTML = rows.map(i => {
            const util = Math.max(i.in_util_pct || 0, i.out_util_pct || 0);
            const st = i.oper_status === 1
                ? '<span class="badge badge-inbound">up</span>'
                : '<span class="badge badge-other">down</span>';
            const hi = util >= 80 ? ' style="background:rgba(245,158,11,0.12)"' : '';
            const nf = nfRoles[i.role] != null ? formatMbps(nfRoles[i.role]) : '—';
            return `<tr class="asn-row-click" data-idx="${i.index}" style="cursor:pointer"${hi}>
                <td>${i.index}</td>
                <td><a href="${detailURL(i.index)}">${i.name}</a></td>
                <td>${i.alias || '—'}</td>
                <td>${roleBadge(i.role)}</td>
                <td>${st}</td>
                <td>${i.speed_mbps || 0}</td>
                <td>${formatMbps(i.in_mbps)}</td>
                <td>${formatMbps(i.out_mbps)}</td>
                <td>${util.toFixed(2)}%</td>
                <td>${nf}</td>
            </tr>`;
        }).join('') || '<tr><td colspan="10">Nenhuma interface neste filtro</td></tr>';
        tbody.querySelectorAll('.asn-row-click').forEach(tr => {
            tr.addEventListener('click', (e) => {
                if (e.target.closest('a')) return;
                window.location.href = detailURL(tr.dataset.idx);
            });
        });
    }

    function renderRoleGroups(data) {
        const el = document.getElementById('router-role-groups');
        if (!el) return;
        const order = ['uplink', 'ix', 'cache', 'bras', 'cgnat'];
        const sums = data.role_sums || [];
        const byRole = {};
        sums.forEach(r => { byRole[r.role] = r; });
        const groups = order.filter(r => byRole[r]).map(r => byRole[r]);
        el.innerHTML = groups.map(r => {
            const ifaces = (data.interfaces || [])
                .filter(i => i.role === r.role && i.oper_status === 1)
                .sort((a, b) => (b.in_mbps + b.out_mbps) - (a.in_mbps + a.out_mbps))
                .slice(0, 4);
            return `<section class="card-section">
                <h3 class="section-title">${r.role.toUpperCase()} · ${formatMbps(r.in_mbps + r.out_mbps)}</h3>
                <p class="section-hint">${r.count} ifaces · util máx ${r.util_max_pct.toFixed(1)}%</p>
                <div class="cards-grid">${ifaces.map(i =>
                    `<a class="dest-card-link" href="${detailURL(i.index)}">${createIfaceCard(i)}</a>`
                ).join('') || '<div class="dest-card"><div class="card-name">—</div></div>'}</div>
            </section>`;
        }).join('');
    }

    function drawChart() {
        const canvas = document.getElementById('router-history-chart');
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

    async function updateHistory() {
        const legend = document.getElementById('router-chart-legend');
        if (historyHours <= 0) {
            chartTs = [];
            chartSeries = [];
            drawChart();
            if (legend) legend.innerHTML = '<span class="legend-muted">Selecione 1h / 6h / 24h</span>';
            return;
        }
        const hist = await fetchHistory(historyHours) || [];
        chartTs = hist.map(h => h.ts);
        chartSeries = [
            { key: 'in', color: '#10b981', values: hist.map(h => h.snmp_in_mbps || 0) },
            { key: 'out', color: '#06b6d4', values: hist.map(h => h.snmp_out_mbps || 0) }
        ];
        const roles = ['uplink', 'ix', 'cache', 'bras'];
        roles.forEach(r => {
            const vals = hist.map(h => (h.by_snmp_role_mbps && h.by_snmp_role_mbps[r]) || 0);
            if (vals.some(v => v > 0.5)) {
                chartSeries.push({ key: r, color: roleColors[r] || '#64748b', values: vals });
            }
        });
        if (legend) {
            legend.innerHTML = chartSeries.map(s =>
                `<span class="legend-item"><i style="background:${s.color}"></i>${s.key}</span>`
            ).join('');
        }
        drawProgress = 0;
        const start = performance.now();
        function frame(now) {
            drawProgress = Math.min(1, (now - start) / 700);
            drawChart();
            if (drawProgress < 1) requestAnimationFrame(frame);
        }
        requestAnimationFrame(frame);
    }

    async function update() {
        const data = await fetchRouterPage();
        if (!data) return;
        updateSamplingChip(data.sampling);

        const exportLink = document.getElementById('export-csv');
        if (exportLink) exportLink.href = exportURL('router', 'csv');

        renderAlerts(data);

        const sub = document.getElementById('router-subtitle');
        if (sub) {
            sub.textContent = data.ok
                ? `SNMP v2c · ${data.host || '—'}:${data.port || '—'} · auth ${data.auth_ok ? 'ok' : 'fail'}`
                : `SNMP offline · ${data.host || '—'}`;
        }
        document.getElementById('router-age').textContent =
            data.age_sec != null ? `última coleta há ${data.age_sec}s` : 'última coleta —';

        pollMs = data.ok ? 5000 : 15000;

        document.getElementById('snmp-sysname').textContent = data.sys_name || (data.ok ? '—' : 'SNMP offline');
        document.getElementById('snmp-uptime').textContent = data.uptime_human || '—';
        document.getElementById('snmp-cpu').textContent = (data.cpu_pct || 0).toFixed(1) + '% CPU';
        document.getElementById('snmp-mem').textContent = (data.mem_pct || 0).toFixed(1) + '% mem';
        document.getElementById('snmp-uplink').textContent =
            formatMbps(data.uplink_in_mbps) + ' / ' + formatMbps(data.uplink_out_mbps);
        document.getElementById('snmp-uplink-util').textContent =
            'utilização ' + (data.uplink_util_pct || 0).toFixed(2) + '%';
        document.getElementById('snmp-if-count').textContent =
            `${data.ifaces_up || 0} / ${data.ifaces_down || 0}`;
        document.getElementById('snmp-role-hint').textContent =
            `cache ${formatMbps(data.cache_mbps || 0)} · IX ${formatMbps(data.ix_mbps || 0)} · BRAS ${formatMbps(data.bras_mbps || 0)}`;

        lastIfaces = data.interfaces || [];
        nfRoles = data.by_iface_role_nf_mbps || {};

        const bars = document.getElementById('iface-bars');
        if (bars) {
            const list = lastIfaces
                .filter(i => i.oper_status === 1 && (i.in_mbps + i.out_mbps) > 0.5)
                .slice(0, 12);
            const max = Math.max(...list.map(i => Math.max(i.in_mbps, i.out_mbps)), 1);
            bars.innerHTML = list.map(i => {
                const label = (i.alias || i.name).substring(0, 22);
                return `<a class="iface-bar-row" href="${detailURL(i.index)}" style="display:grid;text-decoration:none;color:inherit">
                    <div class="iface-bar-label" title="${i.name}">${label}</div>
                    <div class="iface-bar-tracks">
                        <div class="iface-bar in" style="width:${(i.in_mbps / max * 100)}%"></div>
                        <div class="iface-bar out" style="width:${(i.out_mbps / max * 100)}%"></div>
                    </div>
                    <div class="iface-bar-vals">${formatMbps(i.in_mbps)} ↓ · ${formatMbps(i.out_mbps)} ↑</div>
                </a>`;
            }).join('');
        }

        renderRoleGroups(data);

        const topIn = document.getElementById('snmp-top-in');
        const topOut = document.getElementById('snmp-top-out');
        if (topIn) {
            topIn.innerHTML = (data.top_in || []).slice(0, 6).map(i =>
                `<a class="dest-card-link" href="${detailURL(i.index)}">${createIfaceCard(i)}</a>`
            ).join('');
        }
        if (topOut) {
            topOut.innerHTML = (data.top_out || []).slice(0, 6).map(i =>
                `<a class="dest-card-link" href="${detailURL(i.index)}">${createIfaceCard(i)}</a>`
            ).join('');
        }

        renderTable();
    }

    document.getElementById('router-time-filter')?.addEventListener('click', (e) => {
        const btn = e.target.closest('.tf-btn[data-h]');
        if (!btn) return;
        document.querySelectorAll('#router-time-filter .tf-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        historyHours = parseInt(btn.dataset.h, 10) || 0;
        updateHistory();
    });
    document.getElementById('router-role-filter')?.addEventListener('change', (e) => {
        roleFilter = e.target.value || '';
        renderTable();
    });
    document.getElementById('router-state-filter')?.addEventListener('change', (e) => {
        stateFilter = e.target.value || '';
        renderTable();
    });
    document.getElementById('router-search')?.addEventListener('input', (e) => {
        searchQ = e.target.value || '';
        renderTable();
    });

    let timer = setInterval(update, pollMs);
    setInterval(() => {
        clearInterval(timer);
        timer = setInterval(update, pollMs);
    }, 20000);

    update();
    updateHistory();
    window.addEventListener('resize', () => drawChart());
})();
