(function() {
    const {
        formatBytes, formatMbps, createCard, createTrafficRow, createIfaceCard,
        spawnFlowParticle, fetchStats, fetchFlows, fetchSNMP, directionBadge, samplingFactor,
        tweenMbps, renderFlipList, fetchHistoryCompare, prefersReducedMotion
    } = window.Inforflow;

    const cdnColors = {
        'Cloudflare': '#f48120',
        'Akamai': '#0099cc',
        'Fastly': '#ff0744',
        'AWS CloudFront': '#ff9900',
        'Google Cache': '#4285f4',
        'CDN77': '#00b67a',
        'BunnyCDN': '#ff6600',
        'Edgecast': '#6b7280',
        'Limelight': '#8b5cf6',
        'Imperva': '#ef4444',
        'Cachefly': '#14b8a6',
        'G-Core': '#f43f5e',
        'QUIC.cloud': '#f59e0b',
        'Azure CDN': '#0078d4'
    };

    const cdnShort = {
        'Cloudflare': 'CF', 'Akamai': 'AK', 'Fastly': 'FS', 'AWS CloudFront': 'CFNT',
        'Google Cache': 'GGC', 'CDN77': '77', 'BunnyCDN': 'BN', 'Edgecast': 'ED',
        'Limelight': 'LL', 'Imperva': 'IM', 'Cachefly': 'CY', 'G-Core': 'GC',
        'QUIC.cloud': 'QC', 'Azure CDN': 'AZ'
    };

    function isCacheIface(iface) {
        const t = ((iface.alias || '') + ' ' + (iface.name || '')).toUpperCase();
        return /CACHE|CDN|GOOGLE|GGC|CLOUDFLARE|AKAMAI|FASTLY|NETFLIX/.test(t);
    }

    function buildRates(stats) {
        if (stats.cdn_rates && stats.cdn_rates.length) {
            return stats.cdn_rates.map(r => ({
                name: r.name,
                bytes: r.bytes,
                mbps: r.mbps,
                mbps_scaled: r.mbps_scaled,
                in_mbps: r.in_mbps,
                out_mbps: r.out_mbps,
                category: 'cdn'
            }));
        }
        const eff = samplingFactor(stats);
        const breakdown = stats.cdn_breakdown || {};
        const destMap = {};
        (stats.top_destinations || []).forEach(d => { destMap[d.name] = d; });
        return Object.entries(breakdown)
            .filter(([, b]) => b > 0)
            .map(([name, bytes]) => {
                const dest = destMap[name];
                const mbps = dest ? dest.mbps : 0;
                return { name, bytes, mbps, mbps_scaled: mbps * eff, category: 'cdn' };
            })
            .sort((a, b) => (b.mbps_scaled || 0) - (a.mbps_scaled || 0));
    }

    function renderNodes(rates) {
        const el = document.getElementById('cdn-nodes');
        if (!el) return;
        const top = rates.filter(r => r.mbps_scaled > 0 || r.bytes > 0).slice(0, 10);
        el.innerHTML = top.map(r => {
            const short = cdnShort[r.name] || r.name.slice(0, 3).toUpperCase();
            const color = cdnColors[r.name] || '#64748b';
            return `<div class="cdn-node" data-cdn="${r.name}" title="${r.name}: ${formatMbps(r.mbps_scaled)}" style="border-color:${color}">
                <span>${short}</span>
                <small class="cdn-node-mbps">${formatMbps(r.mbps_scaled)}</small>
            </div>`;
        }).join('') || '<div class="cdn-node"><span>—</span></div>';
    }

    async function updateCDNs() {
        const [stats, flows, snmp] = await Promise.all([
            fetchStats(), fetchFlows(), fetchSNMP()
        ]);
        if (!stats) return;

        const rates = buildRates(stats);
        const totalBytes = rates.reduce((s, r) => s + (r.bytes || 0), 0) || 1;
        const cdnTotalScaled = (stats.by_category_mbps_scaled || {}).cdn || 0;
        const eff = samplingFactor(stats);
        const maxMbps = Math.max(...rates.map(r => r.mbps_scaled || 0), 1);
        const top = rates[0];

        tweenMbps(document.getElementById('cdn-total-mbps'), cdnTotalScaled);
        document.getElementById('cdn-total-hint').textContent =
            `NF bruto: ${formatMbps((stats.by_category_mbps || {}).cdn || 0)} · fator ~${eff.toFixed(0)}× · ${formatBytes(totalBytes)} acumulado`;
        document.getElementById('cdn-count').textContent = String(rates.filter(r => r.bytes > 0).length);
        document.getElementById('cdn-sampling-hint').textContent =
            stats.sampling ? `${(cdnTotalScaled / Math.max(stats.sampling.snmp_mbps || 1, 1) * 100).toFixed(1)}% do uplink SNMP` : '—';
        tweenMbps(document.getElementById('cdn-top-mbps'), top ? top.mbps_scaled : 0);
        document.getElementById('cdn-top-name').textContent = top ? top.name : '—';

        const sn = snmp || stats.snmp;
        if (sn && sn.ok) {
            document.getElementById('cdn-snmp-uplink').textContent =
                formatMbps(sn.uplink_in_mbps) + ' / ' + formatMbps(sn.uplink_out_mbps);
            const caches = (sn.interfaces || [])
                .filter(isCacheIface)
                .sort((a, b) => (b.in_mbps + b.out_mbps) - (a.in_mbps + a.out_mbps));
            const snmpCards = document.getElementById('cdn-snmp-ifaces');
            if (snmpCards) {
                snmpCards.innerHTML = caches.length
                    ? caches.slice(0, 8).map(createIfaceCard).join('')
                    : '<div class="dest-card"><div class="card-name">Sem interfaces cache/CDN no SNMP</div></div>';
            }
        } else {
            document.getElementById('cdn-snmp-uplink').textContent = 'SNMP offline';
        }

        renderNodes(rates);

        const list = document.getElementById('cdn-traffic-list');
        if (list) {
            if (!rates.length) {
                list.innerHTML = '<div class="dest-card"><div class="card-name">Aguardando tráfego CDN classificado…</div></div>';
            } else {
                renderFlipList(list, rates, null, (r, i) => createTrafficRow({
                    ...r,
                    percentage: (r.bytes / totalBytes) * 100
                }, {
                    rank: i + 1,
                    maxMbps,
                    color: cdnColors[r.name] || '#f59e0b',
                    flipKey: r.name
                }));
            }
        }

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
                compareEl.textContent = `vs 24h anterior: ${d >= 0 ? '+' : ''}${formatMbps(d)} (${pct >= 0 ? '+' : ''}${pct.toFixed(1)}%)`;
            }).catch(() => {});
        }

        const chart = document.getElementById('cdn-chart');
        if (chart) {
            const active = rates.filter(r => r.mbps_scaled > 0 || r.bytes > 0);
            const maxVal = Math.max(...active.map(r => r.mbps_scaled), 1);
            chart.innerHTML = active.length
                ? active.map(r => {
                    const h = Math.max(12, (r.mbps_scaled / maxVal) * 180);
                    const color = cdnColors[r.name] || '#64748b';
                    return `<div class="cdn-bar-col" title="${r.name}: ${formatMbps(r.mbps_scaled)}">
                        <div class="cdn-bar-mbps">${formatMbps(r.mbps_scaled)}</div>
                        <div class="cdn-bar" style="height: ${h}px; background: ${color}"></div>
                        <span class="cdn-bar-label">${cdnShort[r.name] || r.name.split(' ')[0]}</span>
                        <span class="cdn-bar-bytes">${formatBytes(r.bytes)}</span>
                    </div>`;
                }).join('')
                : '<div class="dest-card"><div class="card-name">Aguardando tráfego CDN…</div></div>';
        }

        const cards = document.getElementById('cdn-detail-cards');
        if (cards) {
            cards.innerHTML = rates.length
                ? rates.map(r => createCard({
                    name: r.name,
                    bytes: r.bytes,
                    percentage: (r.bytes / totalBytes) * 100,
                    category: 'cdn',
                    mbps: r.mbps,
                    mbps_scaled: r.mbps_scaled,
                    in_mbps: r.in_mbps,
                    out_mbps: r.out_mbps
                })).join('')
                : '<div class="dest-card"><div class="card-name">Nenhum CDN detectado ainda</div></div>';
        }

        document.querySelectorAll('.cdn-node').forEach(node => {
            const name = node.dataset.cdn;
            const r = rates.find(x => x.name === name);
            if (r && (r.mbps_scaled > 0 || r.bytes > 0)) {
                node.classList.add('active');
                setTimeout(() => node.classList.remove('active'), 900);
            }
        });

        const tbody = document.getElementById('cdn-table-body');
        if (tbody && flows) {
            const cdnFlows = flows.filter(f => f.category === 'cdn').slice(0, 25);
            tbody.innerHTML = cdnFlows.map(f => {
                const name = f.direction === 'outbound' ? f.destination : f.origin;
                const flowMbps = (f.bytes * 8 / 10 / 1e6) * eff;
                return `<tr>
                    <td>${name}</td>
                    <td class="mono">${formatMbps(flowMbps)}</td>
                    <td>${f.asn || '—'}</td>
                    <td><code>${f.src_ip}</code> → <code>${f.dst_ip}</code></td>
                    <td>${formatBytes(f.bytes)}</td>
                    <td>${directionBadge(f.direction)}</td>
                </tr>`;
            }).join('') || '<tr><td colspan="6">Sem flows CDN no buffer recente</td></tr>';
        }
    }

    setInterval(updateCDNs, 2000);
    updateCDNs();
    setInterval(() => {
        if (prefersReducedMotion()) return;
        spawnFlowParticle('cdn-pipeline', 'cdn', 80);
    }, 1200);
})();
