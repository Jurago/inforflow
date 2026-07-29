(function() {
    const {
        formatBytes, formatMbps, createTrafficRow, createIfaceCard,
        fetchStats, fetchFlows, fetchSNMP, samplingFactor,
        tweenMbps, renderFlipList, spawnFlowParticle, prefersReducedMotion
    } = window.Inforflow;

    const streamColors = {
        'Netflix': '#e50914', 'Globo': '#0066cc', 'YouTube': '#ff0000',
        'Spotify': '#1db954', 'Twitch': '#9146ff', 'Disney+': '#0063e5',
        'HBO Max': '#b535f6', 'RTMP': '#f97316', 'Amazon Prime': '#00a8e1',
        'Paramount+': '#0064ff', 'Apple TV+': '#a2aaad'
    };

    const heroStyles = {
        'Netflix': 'netflix-card',
        'Globo': 'globo-card',
        'YouTube': 'youtube-card',
        'Spotify': 'spotify-card'
    };

    const streamPositions = [
        { top: '10%', left: '70%' }, { top: '30%', left: '85%' },
        { top: '60%', left: '80%' }, { top: '80%', left: '65%' },
        { top: '75%', left: '30%' }, { top: '50%', left: '15%' },
        { top: '20%', left: '20%' }, { top: '10%', left: '45%' },
        { top: '45%', left: '50%' }, { top: '85%', left: '40%' }
    ];

    function isCacheIface(iface) {
        const t = ((iface.alias || '') + ' ' + (iface.name || '')).toUpperCase();
        return /CACHE|NETFLIX|GOOGLE|GLOBO|YOUTUBE|CDN|FACEBOOK|META|STREAM/.test(t);
    }

    function streamingTotalScaled(stats) {
        const scaled = stats.by_category_mbps_scaled || {};
        return (scaled.streaming || 0) + (scaled.netflix || 0) + (scaled.globo || 0);
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
                category: r.category || 'streaming'
            }));
        }
        const eff = samplingFactor(stats);
        const breakdown = stats.streaming_breakdown || {};
        return Object.entries(breakdown)
            .filter(([, b]) => b > 0)
            .map(([name, bytes]) => {
                const dest = (stats.top_destinations || []).find(d => d.name === name);
                const mbps = dest ? dest.mbps : 0;
                return {
                    name, bytes, mbps, mbps_scaled: mbps * eff,
                    category: dest ? dest.category : 'streaming'
                };
            })
            .sort((a, b) => (b.mbps_scaled || 0) - (a.mbps_scaled || 0));
    }

    function initStreamTargets(rates) {
        const container = document.getElementById('stream-targets');
        if (!container) return;
        const active = rates.filter(r => r.mbps_scaled > 0 || r.bytes > 0);
        container.innerHTML = '';
        active.slice(0, 10).forEach((r, i) => {
            const pos = streamPositions[i % streamPositions.length];
            const el = document.createElement('div');
            el.className = 'stream-target';
            el.dataset.service = r.name;
            el.textContent = r.name;
            el.style.top = pos.top;
            el.style.left = pos.left;
            el.style.borderColor = streamColors[r.name] || '#8b5cf6';
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
            const color = streamColors[r.name] || '#8b5cf6';
            const pct = totalScaled > 0 ? Math.min(100, (r.mbps_scaled / totalScaled) * 100) : 0;
            return `
                <div class="hero-card ${cls}" style="--stream-color:${color}">
                    <div class="hero-card-bg"></div>
                    <div class="hero-card-content">
                        <span class="hero-label">${r.name}</span>
                        <span class="hero-value">${formatMbps(r.mbps_scaled)}</span>
                        <span class="hero-rate">${formatBytes(r.bytes)} acumulado · ${pct.toFixed(1)}% do streaming</span>
                        <div class="hero-bar"><div class="hero-bar-fill" style="width:${pct}%;background:${color}"></div></div>
                    </div>
                </div>`;
        }).join('');
    }

    function renderIoSummary(stats, totalScaled) {
        const el = document.getElementById('stream-io-summary');
        if (!el) return;
        const inM = ((stats.by_category_in_mbps || {}).streaming || 0)
            + ((stats.by_category_in_mbps || {}).netflix || 0)
            + ((stats.by_category_in_mbps || {}).globo || 0);
        const outM = ((stats.by_category_out_mbps || {}).streaming || 0)
            + ((stats.by_category_out_mbps || {}).netflix || 0)
            + ((stats.by_category_out_mbps || {}).globo || 0);
        el.innerHTML = `
            <div class="io-stat"><span class="io-label">Entrada</span><span class="io-value">${formatMbps(inM)}</span></div>
            <div class="io-stat"><span class="io-label">Saída</span><span class="io-value">${formatMbps(outM)}</span></div>
            <div class="io-stat"><span class="io-label">Total cat.</span><span class="io-value">${formatMbps(totalScaled)}</span></div>`;
    }

    async function updateStreaming() {
        const [stats, snmp, flows] = await Promise.all([
            fetchStats(), fetchSNMP(), fetchFlows()
        ]);
        if (!stats) return;

        const rates = buildRates(stats);
        const totalScaled = streamingTotalScaled(stats);
        const totalBytes = rates.reduce((s, r) => s + (r.bytes || 0), 0) || 1;
        const eff = samplingFactor(stats);
        const maxMbps = Math.max(...rates.map(r => r.mbps_scaled || 0), 1);

        tweenMbps(document.getElementById('stream-total-mbps'), totalScaled);
        document.getElementById('stream-total-hint').textContent =
            `NF bruto: ${formatMbps((stats.by_category_mbps?.streaming || 0) + (stats.by_category_mbps?.netflix || 0) + (stats.by_category_mbps?.globo || 0))} · fator ~${eff.toFixed(0)}×`;
        document.getElementById('stream-count').textContent = String(rates.filter(r => r.bytes > 0).length);
        document.getElementById('stream-sampling-hint').textContent =
            stats.sampling ? `${stats.sampling.mode} · SNMP ${formatMbps(stats.sampling.snmp_mbps)}` : '—';

        const sn = snmp || stats.snmp;
        if (sn && sn.ok) {
            document.getElementById('stream-snmp-uplink').textContent =
                formatMbps(sn.uplink_in_mbps) + ' / ' + formatMbps(sn.uplink_out_mbps);
            const caches = (sn.interfaces || [])
                .filter(isCacheIface)
                .sort((a, b) => (b.in_mbps + b.out_mbps) - (a.in_mbps + a.out_mbps));
            const cacheIn = caches.reduce((s, i) => s + (i.in_mbps || 0), 0);
            const cacheOut = caches.reduce((s, i) => s + (i.out_mbps || 0), 0);
            document.getElementById('stream-snmp-cache').textContent = formatMbps(cacheIn + cacheOut);
            document.getElementById('stream-snmp-cache-label').textContent =
                caches.length ? `${caches.length} ifaces · ↓${formatMbps(cacheIn)} ↑${formatMbps(cacheOut)}` : 'sem iface cache';
            const snmpCards = document.getElementById('streaming-snmp-ifaces');
            if (snmpCards) {
                snmpCards.innerHTML = caches.length
                    ? caches.slice(0, 6).map(createIfaceCard).join('')
                    : '<div class="dest-card"><div class="card-name">Nenhuma interface CACHE no SNMP</div></div>';
            }
        } else {
            document.getElementById('stream-snmp-uplink').textContent = 'SNMP offline';
            document.getElementById('stream-snmp-cache').textContent = '—';
        }

        renderHero(rates, totalScaled);
        renderIoSummary(stats, totalScaled);

        const list = document.getElementById('streaming-traffic-list');
        if (list) {
            if (!rates.length) {
                list.innerHTML = '<div class="dest-card"><div class="card-name">Aguardando NetFlow de streaming…</div></div>';
            } else {
                renderFlipList(list, rates, null, (r, i) => createTrafficRow({
                    ...r,
                    percentage: (r.bytes / totalBytes) * 100
                }, {
                    rank: i + 1,
                    maxMbps,
                    color: streamColors[r.name] || '#8b5cf6',
                    flipKey: r.name
                }));
            }
        }

        initStreamTargets(rates);
        document.querySelectorAll('.stream-target').forEach(el => {
            const svc = el.dataset.service;
            const r = rates.find(x => x.name === svc);
            if (r && (r.mbps_scaled > 0 || r.bytes > 0)) {
                el.classList.add('active');
                setTimeout(() => el.classList.remove('active'), 1500);
                if (!prefersReducedMotion()) {
                    spawnFlowParticle('stream-rays', r.category || 'streaming', r.mbps_scaled);
                }
            }
        });

        const timeline = document.getElementById('streaming-timeline');
        if (timeline && flows) {
            const streamCats = ['netflix', 'globo', 'streaming'];
            const streamFlows = flows.filter(f => streamCats.includes(f.category)).slice(0, 20);
            timeline.innerHTML = streamFlows.length
                ? streamFlows.map(f => {
                    const time = new Date(f.timestamp * 1000).toLocaleTimeString('pt-BR');
                    const label = f.direction === 'outbound' ? f.destination : f.origin;
                    const color = streamColors[label] || '#8b5cf6';
                    const flowMbps = (f.bytes * 8 / 10 / 1e6) * eff;
                    return `<div class="timeline-item">
                        <div class="timeline-dot" style="background: ${color}"></div>
                        <span class="timeline-time">${time}</span>
                        <span class="timeline-service">${label || f.category}</span>
                        <span class="timeline-mbps">${formatMbps(flowMbps)}</span>
                        <span class="timeline-bytes">${formatBytes(f.bytes)}</span>
                    </div>`;
                }).join('')
                : '<div class="timeline-item"><span class="timeline-service">Sem flows de streaming no buffer</span></div>';
        }
    }

    setInterval(updateStreaming, 2000);
    updateStreaming();
})();
