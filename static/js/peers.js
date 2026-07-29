(function() {
    const {
        formatBytes, formatMbps, createCard, fetchStats, fetchFlows,
        fetchBGP, directionBadge, categoryBadge
    } = window.Inforflow;

    const roleLabel = {
        ix: 'IX', content: 'Conteúdo', transit: 'Trânsito',
        regional: 'Regional', local: 'Local', private: 'Privado'
    };

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

    function renderOrbit(peers) {
        const orbit = document.getElementById('peer-orbit');
        if (!orbit) return;

        const established = (peers || []).filter(p => p.established);
        const uniq = [];
        const seen = new Set();
        for (const p of established) {
            const key = p.remote_as || p.asn;
            if (seen.has(key)) continue;
            seen.add(key);
            uniq.push(p);
            if (uniq.length >= 12) break;
        }

        const maxMbps = Math.max(...uniq.map(p => p.mbps || 0), 0.01);
        orbit.innerHTML = '';

        let svg = orbit.querySelector('svg.peer-rays');
        if (!svg) {
            svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
            svg.classList.add('peer-rays');
            svg.setAttribute('viewBox', '0 0 400 400');
            svg.style.cssText = 'position:absolute;inset:0;width:100%;height:100%;pointer-events:none;overflow:visible';
            orbit.appendChild(svg);
        } else {
            svg.innerHTML = '';
        }

        uniq.forEach((p, i) => {
            const angle = (i / Math.max(uniq.length, 1)) * 360 - 90;
            const radius = 130 + (i % 3) * 36;
            const x = Math.cos(angle * Math.PI / 180) * radius;
            const y = Math.sin(angle * Math.PI / 180) * radius;
            const active = (p.mbps || 0) > 0.05;
            const thickness = Math.max(1, Math.min(6, ((p.mbps || 0) / maxMbps) * 6));

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

            const node = document.createElement('div');
            node.className = 'peer-node-orbit' + (active ? ' active' : ' idle');
            node.textContent = shortLabel(p.name, p.asn);
            node.style.left = `calc(50% + ${x}px - 24px)`;
            node.style.top = `calc(50% + ${y}px - 24px)`;
            if (!active) node.style.opacity = '0.4';
            node.title = `${p.name} ${p.asn}\n${p.remote_addr}\n${formatMbps(p.mbps)}`;
            orbit.appendChild(node);
        });
    }

    function renderBgpTable(peers) {
        const tbody = document.getElementById('bgp-table-body');
        if (!tbody) return;
        const list = (peers || []).slice().sort((a, b) => {
            if (a.established !== b.established) return a.established ? -1 : 1;
            return (b.mbps || 0) - (a.mbps || 0);
        });
        tbody.innerHTML = list.map(p => `<tr class="${p.established ? '' : 'row-dim'}">
            <td><code>${p.remote_addr}</code></td>
            <td>${p.asn || '—'}</td>
            <td>${p.name || '—'}</td>
            <td>${stateBadge(p.state_name, p.established)}</td>
            <td><span class="badge badge-peer">${roleLabel[p.role] || p.role || '—'}</span></td>
            <td>${formatMbps(p.mbps)}</td>
            <td>${formatBytes(p.bytes)}</td>
            <td>${(p.in_updates || 0).toLocaleString('pt-BR')} / ${(p.out_updates || 0).toLocaleString('pt-BR')}</td>
        </tr>`).join('') || '<tr><td colspan="8">Aguardando SNMP BGP…</td></tr>';
    }

    async function updatePeers() {
        const [stats, bgp, flows] = await Promise.all([
            fetchStats(),
            fetchBGP(),
            fetchFlows()
        ]);

        const snap = bgp || stats?.bgp;
        if (snap) {
            document.getElementById('peer-established').textContent =
                `${snap.established || 0} / ${snap.total || 0}`;
            document.getElementById('peer-total').textContent = String(snap.total || 0);
            const localEl = document.getElementById('bgp-local-as');
            if (localEl) localEl.textContent = snap.local_asn || 'AS—';

            renderOrbit(snap.peers);
            renderBgpTable(snap.peers);

            const ix = (snap.peers || []).filter(p => p.remote_as === 26162);
            const ixMbps = ix.reduce((s, p) => s + (p.mbps || 0), 0);
            const ixBytes = ix.reduce((s, p) => s + (p.bytes || 0), 0);
            document.getElementById('peer-ixbr').textContent =
                ixMbps > 0.01 ? formatMbps(ixMbps) : formatBytes(ixBytes);
        }

        if (stats) {
            const peerBytes = (stats.peer_breakdown || []).reduce((s, p) => s + (p.bytes || 0), 0)
                || stats.by_category?.peer || 0;
            document.getElementById('peer-traffic').textContent = formatBytes(peerBytes);

            const breakdown = document.getElementById('peer-breakdown');
            if (breakdown) {
                const items = stats.peer_breakdown || [];
                breakdown.innerHTML = items.length
                    ? items.map(o => createCard(o)).join('')
                    : '<div class="dest-card"><div class="card-name">Sem tráfego atribuído a ASN BGP ainda</div></div>';
            }

            const dests = (stats.top_destinations || []).filter(d =>
                d.category === 'peer' || (d.name || '').includes('Peer')
            );
            const destContainer = document.getElementById('peer-destinations');
            if (destContainer) {
                destContainer.innerHTML = dests.length
                    ? dests.map(d => createCard(d)).join('')
                    : '<div class="dest-card"><div class="card-name">Aguardando classificação peer…</div></div>';
            }
        }

        const tbody = document.getElementById('peer-table-body');
        if (tbody && flows) {
            const peerFlows = flows.filter(f =>
                f.peer_asn || f.category === 'peer' || (f.asn && snap?.peers?.some(p => p.asn === f.asn))
            ).slice(0, 20);
            tbody.innerHTML = peerFlows.map(f => {
                const peer = f.peer_name
                    ? `${f.peer_name} (${f.peer_asn || f.asn || '—'})`
                    : (f.direction === 'outbound' ? f.destination : f.origin);
                return `<tr>
                    <td>${peer}</td>
                    <td>${f.peer_asn || f.asn || '—'}</td>
                    <td><code>${f.src_ip}</code> → <code>${f.dst_ip}</code></td>
                    <td>${categoryBadge(f.category)}</td>
                    <td>${formatBytes(f.bytes)}</td>
                    <td>${directionBadge(f.direction)}</td>
                </tr>`;
            }).join('') || '<tr><td colspan="6">Nenhum flow associado a peer BGP no momento</td></tr>';
        }
    }

    setInterval(updatePeers, 2500);
    updatePeers();
})();
