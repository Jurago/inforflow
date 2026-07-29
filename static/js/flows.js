(function () {
    const { formatBytes, categoryBadge, directionBadge, apiGet } = window.Inforflow;

    function paramsFromURL() {
        const u = new URLSearchParams(window.location.search);
        return {
            q: u.get('q') || '',
            ip: u.get('ip') || '',
            category: u.get('category') || '',
            asn: u.get('asn') || ''
        };
    }

    function applyParamsToForm(p) {
        const q = document.getElementById('flow-q');
        const ip = document.getElementById('flow-ip');
        const cat = document.getElementById('flow-cat');
        const asn = document.getElementById('flow-asn');
        if (q && p.q) q.value = p.q;
        if (ip && p.ip) ip.value = p.ip;
        if (cat && p.category) cat.value = p.category;
        if (asn && p.asn) asn.value = p.asn;
        else if (q && p.asn && !p.q) q.value = p.asn;
    }

    async function search() {
        const q = document.getElementById('flow-q').value.trim();
        const ip = document.getElementById('flow-ip').value.trim();
        const cat = document.getElementById('flow-cat').value;
        const asnEl = document.getElementById('flow-asn');
        const asn = asnEl ? asnEl.value.trim() : '';
        const params = new URLSearchParams({ limit: '100' });
        if (q) params.set('q', q);
        if (ip) params.set('ip', ip);
        if (cat) params.set('category', cat);
        if (asn) params.set('asn', asn);
        const flows = await apiGet('/flows?' + params).catch(() => []);
        const tbody = document.getElementById('flows-body');
        if (!tbody) return;
        const filterHint = document.getElementById('flow-filter-hint');
        if (filterHint) {
            filterHint.textContent = asn
                ? `Filtrando ASN ${asn}` + (flows.length ? ` · ${flows.length} flows` : '')
                : '';
        }
        tbody.innerHTML = (flows || []).map(f => {
            const t = new Date(f.timestamp * 1000).toLocaleTimeString('pt-BR');
            const svc = f.destination !== f.src_ip ? f.destination : f.origin;
            return `<tr>
                <td>${t}</td>
                <td>${categoryBadge(f.category)}</td>
                <td><code>${f.src_ip}</code> → <code>${f.dst_ip}</code></td>
                <td>${svc || '—'}</td>
                <td>${f.dst_asn || f.asn || '—'}</td>
                <td>${formatBytes(f.bytes)}</td>
                <td>${directionBadge(f.direction)}</td>
                <td>${f.ip_version || '4'}</td>
            </tr>`;
        }).join('') || '<tr><td colspan="8">Nenhum flow encontrado</td></tr>';
    }

    applyParamsToForm(paramsFromURL());
    document.getElementById('flow-search').addEventListener('click', search);
    search();
})();
