(function () {
    const IF = window.Inforflow || {};
    const {
        formatBytes, formatMbps, formatNumber, createCard, createIfaceCard,
        renderCategoryBars, spawnFlowParticle, renderAlerts,
        categoryBadge, directionBadge,
        tweenMbps, prefersReducedMotion, updateSourceIP, exportURL, updateSamplingChip
    } = IF;
    const fetchDash = typeof IF.fetchDashboardPage === 'function'
        ? IF.fetchDashboardPage
        : (typeof IF.fetchStats === 'function' ? IF.fetchStats : async () => null);

    let prevFlows = [];
    let lastMbpsScaled = 0;
    let pollMs = 2000;
    let timer = null;
    let updating = false;

    const catHref = {
        cdn: '/cdns', netflix: '/streaming', globo: '/streaming',
        streaming: '/streaming', apple: '/streaming', peer: '/peers',
        social: '/asn', gaming: '/asn', cloud: '/cdns', dns: '/flows', other: '/asn'
    };

    function setText(id, text) {
        const el = document.getElementById(id);
        if (el) el.textContent = text;
    }

    document.getElementById('export-csv')?.addEventListener('click', e => {
        const a = e.currentTarget;
        if (a && exportURL) a.href = exportURL('dashboard', 'csv');
    });

    function serviceHref(item) {
        const cat = (item.category || '').toLowerCase();
        const name = item.name || '';
        if (cat === 'cdn' || /cloudflare|akamai|fastly|cloudfront|google cache/i.test(name)) {
            return '/cdn/detail?name=' + encodeURIComponent(name);
        }
        if (['streaming', 'netflix', 'globo', 'apple'].includes(cat) ||
            /youtube|netflix|globo|disney|twitch|spotify|hbo/i.test(name)) {
            return '/streaming/detail?name=' + encodeURIComponent(name);
        }
        if (item.asn) return '/asn/detail?asn=' + encodeURIComponent(item.asn);
        return catHref[cat] || '/flows';
    }

    function renderStatus(data) {
        const el = document.getElementById('dash-status');
        if (!el) return;
        const snmp = data.snmp || {};
        const bgp = data.bgp || {};
        const samp = data.sampling || {};
        const snmpTxt = snmp.ok
            ? `SNMP ok · há ${snmp.age_sec != null ? snmp.age_sec + 's' : '—'}`
            : `SNMP down`;
        const nfTxt = `NetFlow ${(data.packets_per_sec || 0).toFixed(0)} pps`;
        const bgpTxt = bgp.ok
            ? `BGP ${bgp.established || 0}/${bgp.total || 0}` + ((bgp.down || 0) > 0 ? ` · ${bgp.down} down` : '')
            : 'BGP offline';
        const sampTxt = samp.effective
            ? `fator ~${Number(samp.effective).toFixed(0)}× (${samp.mode || 'auto'})`
            : 'fator —';
        el.innerHTML = `<div class="alert-item alert-ok" style="flex-wrap:wrap;gap:12px">
            <span>${snmpTxt}</span><span>${nfTxt}</span><span>${bgpTxt}</span><span>${sampTxt}</span>
        </div>`;
    }

    function renderOpsShortcuts(data) {
        const el = document.getElementById('ops-shortcuts');
        if (!el) return;
        const bgp = data.bgp || {};
        const snmp = data.snmp || {};
        const down = (bgp.top_down || []).length || bgp.down || 0;
        const critical = (snmp.critical || []).filter(i =>
            Math.max(i.in_util_pct || 0, i.out_util_pct || 0) >= 70
        ).length;
        const talkers = (data.top_talkers || []).length;
        const alertsN = (data.alerts || []).length;
        const gap = data.gap_pct || 0;
        const chips = [
            { href: '/peers?state=down', label: `BGP down · ${down}`, hot: down > 0 },
            { href: '/router', label: `Ifaces críticas · ${critical}`, hot: critical > 0 },
            { href: '/flows', label: `Talkers CGNAT · ${talkers}`, hot: false },
            { href: '/#alerts-list', label: `Alertas · ${alertsN}`, hot: alertsN > 0 },
            { href: '/sampling', label: `Gap NF×SNMP · ${gap.toFixed(0)}%`, hot: gap >= 35 }
        ];
        el.innerHTML = chips.map(c =>
            `<a class="ops-chip${c.hot ? ' ops-chip-hot' : ''}" href="${c.href}">${c.label}</a>`
        ).join('');
    }

    function renderBlocks(blocks) {
        const el = document.getElementById('dash-blocks');
        if (!el) return;
        el.innerHTML = (blocks || []).map(b => `
            <a class="tf-btn" href="${b.href}" style="text-decoration:none">
                ${b.label} · ${formatMbps(b.mbps_scaled)} · ${(b.share_pct || 0).toFixed(0)}%
            </a>`).join('');
    }

    function renderBGP(bgp) {
        const el = document.getElementById('bgp-summary');
        const hint = document.getElementById('bgp-summary-hint');
        if (!el || !bgp) return;
        if (hint) {
            hint.textContent = bgp.ok
                ? `${bgp.established}/${bgp.total} estabelecidas · AS${bgp.local_as || ''}`
                : (bgp.error || 'indisponível');
        }
        const up = bgp.top_up || [];
        const down = bgp.top_down || [];
        el.innerHTML = [
            ...up.map(p => `<a class="dest-card-link" href="/peers/detail?asn=${encodeURIComponent(p.asn || '')}">
                <div class="dest-card cat-${p.role === 'ix' ? 'peer' : 'cdn'}">
                    <div class="card-name">${p.name || p.asn}</div>
                    <div class="card-mbps">${formatMbps(p.mbps_scaled != null ? p.mbps_scaled : p.mbps)}</div>
                    <div class="card-pct">${p.asn || ''} · ${p.state_name || 'up'}</div>
                </div></a>`),
            ...down.map(p => `<a class="dest-card-link" href="/peers/detail?asn=${encodeURIComponent(p.asn || '')}">
                <div class="dest-card" style="opacity:0.65;border-color:#f87171">
                    <div class="card-name">${p.name || p.asn} ↓</div>
                    <div class="card-pct">${p.state_name || 'down'}</div>
                </div></a>`)
        ].join('') || '<div class="dest-card"><div class="card-name">Aguardando BGP…</div></div>';
    }

    function renderConsumptionLinked(list) {
        const el = document.getElementById('consumption-grid');
        if (!el || !list) return;
        const topCat = list.length ? list.slice().sort((a, b) =>
            (b.mbps_scaled || b.mbps || 0) - (a.mbps_scaled || a.mbps || 0)
        )[0] : null;
        el.innerHTML = list.map(c => {
            const real = c.mbps_scaled != null && c.mbps_scaled > 0 ? c.mbps_scaled : c.mbps;
            const href = catHref[c.category] || '/asn';
            const breathe = topCat && topCat.category === c.category ? ' cons-breathe' : '';
            return `
            <a class="dest-card-link" href="${href}">
            <div class="consumption-card cat-${c.category}${breathe}">
                <div class="cons-label">${c.label || c.category}</div>
                <div class="cons-mbps">${formatMbps(real)}</div>
                <div class="cons-bytes">${formatBytes(c.bytes)} · ${(c.percentage || 0).toFixed(1)}%</div>
                <div class="card-bar"><div class="card-bar-fill cat-fill-${c.category}" style="width:${Math.min(c.percentage || 0, 100)}%"></div></div>
            </div></a>`;
        }).join('');
    }

    function drawSpark(pts) {
        const canvas = document.getElementById('dash-spark');
        if (!canvas) return;
        const rect = canvas.getBoundingClientRect();
        const w = rect.width || 800;
        const h = 160;
        const dpr = window.devicePixelRatio || 1;
        canvas.width = w * dpr;
        canvas.height = h * dpr;
        canvas.style.width = w + 'px';
        canvas.style.height = h + 'px';
        const ctx = canvas.getContext('2d');
        ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
        ctx.clearRect(0, 0, w, h);
        const pad = { l: 44, r: 12, t: 12, b: 20 };
        const plotW = w - pad.l - pad.r;
        const plotH = h - pad.t - pad.b;
        const data = pts || [];
        if (!data.length) {
            ctx.fillStyle = '#64748b';
            ctx.font = '12px DM Sans, sans-serif';
            ctx.fillText('Aguardando histórico…', pad.l, pad.t + 30);
            return;
        }
        let maxV = 1;
        data.forEach(p => {
            maxV = Math.max(maxV, p.mbps_scaled || 0, p.snmp_in_mbps || 0, p.snmp_out_mbps || 0);
        });
        const n = data.length;
        ctx.strokeStyle = 'rgba(148,163,184,0.2)';
        for (let i = 0; i <= 3; i++) {
            const y = pad.t + (plotH * i) / 3;
            ctx.beginPath();
            ctx.moveTo(pad.l, y);
            ctx.lineTo(pad.l + plotW, y);
            ctx.stroke();
            ctx.fillStyle = '#94a3b8';
            ctx.font = '10px IBM Plex Mono, monospace';
            ctx.fillText(formatMbps(maxV * (1 - i / 3)).replace(' Mbps', ''), 2, y + 3);
        }
        function series(key, color) {
            ctx.strokeStyle = color;
            ctx.lineWidth = 2;
            ctx.beginPath();
            data.forEach((p, i) => {
                const x = pad.l + (i / Math.max(n - 1, 1)) * plotW;
                const y = pad.t + plotH - ((p[key] || 0) / maxV) * plotH;
                if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
            });
            ctx.stroke();
        }
        series('mbps_scaled', '#3b82f6');
        series('snmp_in_mbps', '#10b981');
        series('snmp_out_mbps', '#06b6d4');
    }

    function updateAlertBadge(count) {
        const b = document.getElementById('alert-badge');
        if (!b) return;
        if (count > 0) {
            b.style.display = 'inline-flex';
            b.textContent = count + ' alerta' + (count > 1 ? 's' : '');
        } else {
            b.style.display = 'none';
        }
    }

    function schedulePoll() {
        if (timer) clearInterval(timer);
        pollMs = document.hidden ? 8000 : 2000;
        timer = setInterval(updateDashboard, pollMs);
    }

    async function updateDashboard() {
        if (updating) return;
        updating = true;
        try {
            const data = await fetchDash();
            if (!data) {
                const st = document.getElementById('dash-status');
                if (st) st.innerHTML = '<div class="alert-item alert-warning">Aguardando dados do coletor… Verifique login e /api/health.</div>';
                return;
            }

            if (updateSamplingChip) updateSamplingChip(data.sampling);
            if (updateSourceIP) updateSourceIP(data.exporter || data.source);

            const exportLink = document.getElementById('export-csv');
            if (exportLink && exportURL) exportLink.href = exportURL('dashboard', 'csv');

            const sub = document.getElementById('dash-subtitle');
            if (sub) sub.textContent = data.window_hint || sub.textContent;

            const cmp = document.getElementById('dash-compare');
            if (cmp) {
                const d = data.compare_24h_delta_mbps || 0;
                const p = data.compare_24h_pct || 0;
                cmp.textContent = `vs ~24h: ${d >= 0 ? '+' : ''}${formatMbps(d)} (${p >= 0 ? '+' : ''}${p.toFixed(1)}%)`;
            }

            if (renderAlerts) renderAlerts(data.alerts || [], 'alerts-list');
            updateAlertBadge((data.alerts || []).length);
            renderStatus(data);
            renderOpsShortcuts(data);
            renderBlocks(data.block_shares);

            const snmp = data.snmp || {};
            if (snmp.ok) {
                setText('snmp-uplink-mbps',
                    formatMbps(snmp.uplink_in_mbps) + ' / ' + formatMbps(snmp.uplink_out_mbps));
                setText('snmp-router-name',
                    (snmp.sys_name || 'roteador') + ' · util ' + (snmp.uplink_util_pct || 0).toFixed(1) + '%' +
                    (snmp.deduped ? ' · dedupe' : ''));
                setText('snmp-cpu-mem',
                    (snmp.cpu_pct || 0).toFixed(0) + '% / ' + (snmp.mem_pct || 0).toFixed(0) + '%');
            } else {
                setText('snmp-uplink-mbps', 'SNMP offline');
                setText('snmp-router-name', snmp.error || '—');
                setText('snmp-cpu-mem', '—');
            }

            if (tweenMbps) tweenMbps(document.getElementById('nf-scaled-mbps'), data.mbps_scaled || data.mbps);
            else setText('nf-scaled-mbps', formatMbps(data.mbps_scaled || data.mbps));
            lastMbpsScaled = data.mbps_scaled || data.mbps || 0;
            setText('nf-raw-hint',
                `bruto ${formatMbps(data.mbps || 0)} · ${(data.classified_pct || 0).toFixed(0)}% classif.`);

            setText('dash-gap', formatMbps(data.gap_mbps || 0));
            const gapPct = data.gap_pct || 0;
            const gapEl = document.getElementById('dash-gap-pct');
            if (gapEl) {
                gapEl.textContent = gapPct.toFixed(1) + '% do SNMP médio';
                gapEl.style.color = gapPct > 30 ? '#fbbf24' : '';
            }

            setText('dash-v4v6',
                formatMbps(data.ipv4_mbps || 0).replace(' Mbps', '') + ' / ' +
                formatMbps(data.ipv6_mbps || 0).replace(' Mbps', '') + ' · ' + (snmp.uptime_human || '—'));

            drawSpark(data.sparkline || []);
            renderConsumptionLinked(data.consumption || []);
            if (renderCategoryBars) {
                renderCategoryBars(
                    data.by_category || {},
                    data.total_bytes,
                    data.by_category_mbps_scaled
                );
            }
            document.querySelectorAll('#category-bars .category-bar-item').forEach(row => {
                const cat = row.querySelector('.category-bar-label')?.textContent?.trim();
                if (!cat) return;
                const href = catHref[cat] || '/asn';
                row.style.cursor = 'pointer';
                row.onclick = () => { window.location.href = href; };
            });

            setText('dash-exporter', data.exporter || '—');

            const pipeCats = document.getElementById('pipeline-cats');
            if (pipeCats) {
                const top = (data.consumption || [])
                    .slice()
                    .sort((a, b) => (b.mbps_scaled || 0) - (a.mbps_scaled || 0))
                    .slice(0, 5);
                pipeCats.innerHTML = top.map(c =>
                    `<a class="cdn-node" href="${catHref[c.category] || '/asn'}" style="position:relative;border-color:var(--accent)">
                        <span>${(c.label || c.category).slice(0, 6)}</span>
                        <small class="cdn-node-mbps">${formatMbps(c.mbps_scaled || c.mbps)}</small>
                    </a>`
                ).join('') || `<span class="section-hint">${data.destination_count || 0} serviços</span>`;
            }

            renderBGP(data.bgp);

            const crit = document.getElementById('snmp-critical');
            if (crit && createIfaceCard) {
                const list = snmp.critical || [];
                crit.innerHTML = list.length
                    ? list.map(i => {
                        const util = Math.max(i.in_util_pct || 0, i.out_util_pct || 0);
                        const card = createIfaceCard(i);
                        const wrap = util >= 80
                            ? `<div style="outline:1px solid #f59e0b;border-radius:8px">${card}</div>`
                            : card;
                        return `<a class="dest-card-link" href="/router/detail?ifindex=${i.index}">${wrap}</a>`;
                    }).join('')
                    : '<div class="dest-card"><div class="card-name">Aguardando SNMP…</div></div>';
            }

            const talkersBody = document.getElementById('talkers-body');
            if (talkersBody) {
                const list = data.top_talkers || [];
                talkersBody.innerHTML = list.length
                    ? list.map(t => `<tr>
                        <td><a href="/flows?ip=${encodeURIComponent(t.ip)}"><code>${t.ip}</code></a></td>
                        <td>${formatMbps(t.mbps_scaled != null ? t.mbps_scaled : t.mbps)}</td>
                        <td>${categoryBadge ? categoryBadge(t.top_category) : (t.top_category || '')}</td>
                        <td>${formatBytes(t.bytes)}</td>
                        <td>${formatNumber ? formatNumber(t.flows) : t.flows}</td>
                    </tr>`).join('')
                    : '<tr><td colspan="5">Aguardando clientes CGNAT…</td></tr>';
            }

            const destContainer = document.getElementById('top-destinations');
            if (destContainer && createCard) {
                destContainer.innerHTML = (data.top_destinations || []).map(d =>
                    `<a class="dest-card-link" href="${serviceHref(d)}">${createCard(d)}</a>`
                ).join('') || '<div class="dest-card"><div class="card-name">—</div></div>';
            }
            const origContainer = document.getElementById('top-origins');
            if (origContainer && createCard) {
                origContainer.innerHTML = (data.top_origins || []).map(d =>
                    `<a class="dest-card-link" href="${serviceHref(d)}">${createCard(d)}</a>`
                ).join('') || '<div class="dest-card"><div class="card-name">—</div></div>';
            }

            const flows = data.flows || [];
            const tbody = document.getElementById('flow-table-body');
            if (tbody) {
                tbody.innerHTML = flows.length
                    ? flows.slice(0, 15).map(f => {
                        const time = f.timestamp ? new Date(f.timestamp * 1000).toLocaleTimeString('pt-BR') : '—';
                        const asn = f.peer_asn || f.asn || f.dst_asn || '';
                        return `<tr>
                            <td>${time}</td>
                            <td><code>${f.src_ip || '—'}</code> → <code>${f.dst_ip || '—'}</code></td>
                            <td>${asn ? `<a href="/asn/detail?asn=${encodeURIComponent(asn)}">${asn}</a>` : '—'}</td>
                            <td>${categoryBadge ? categoryBadge(f.category) : (f.category || '')}</td>
                            <td>${formatBytes(f.bytes)}</td>
                            <td>${directionBadge ? directionBadge(f.direction) : (f.direction || '')}</td>
                        </tr>`;
                    }).join('')
                    : '<tr><td colspan="6">Sem flows recentes</td></tr>';
            }

            if (spawnFlowParticle) {
                flows.slice(0, 3).forEach(f => {
                    if (!prevFlows.find(p => p.id === f.id)) {
                        spawnFlowParticle('pipeline-track', f.category, lastMbpsScaled);
                    }
                });
            }
            prevFlows = flows;
        } catch (e) {
            console.error('dashboard update failed', e);
        } finally {
            updating = false;
        }
    }

    document.addEventListener('visibilitychange', schedulePoll);
    schedulePoll();
    updateDashboard();
    setInterval(() => {
        if ((prefersReducedMotion && prefersReducedMotion()) || document.hidden) return;
        if (!spawnFlowParticle) return;
        const cats = ['cdn', 'netflix', 'globo', 'streaming', 'peer'];
        spawnFlowParticle('pipeline-track', cats[Math.floor(Math.random() * cats.length)], lastMbpsScaled);
    }, 2000);
    let resizeTimer = null;
    window.addEventListener('resize', () => {
        clearTimeout(resizeTimer);
        resizeTimer = setTimeout(updateDashboard, 250);
    });
})();
