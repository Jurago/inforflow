const API_BASE = '/api';

function getApiToken() {
    return localStorage.getItem('inforflow_api_token') || '';
}

function apiHeaders() {
    const h = {};
    const tok = getApiToken();
    if (tok) h['X-API-Token'] = tok;
    return h;
}

async function apiGet(path) {
    const res = await fetch(`${API_BASE}${path}`, { headers: apiHeaders() });
    if (res.status === 401) {
        if (window.location.pathname !== '/login') {
            localStorage.removeItem('inforflow_api_token');
            window.location.href = '/login';
        }
        throw new Error('HTTP 401');
    }
    if (!res.ok) throw new Error('HTTP ' + res.status);
    return res.json();
}

async function exportDownload(kind, format) {
    const q = new URLSearchParams({ kind: kind || 'stats', format: format || 'csv' });
    const res = await fetch(`${API_BASE}/export?${q}`, { headers: apiHeaders() });
    if (res.status === 401) {
        window.location.href = '/login';
        return;
    }
    if (!res.ok) return;
    const blob = await res.blob();
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = `inforflow-${kind}.${format || 'csv'}`;
    a.click();
    URL.revokeObjectURL(a.href);
}

function formatBytes(bytes) {
    if (!bytes || bytes < 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.min(Math.floor(Math.log(bytes) / Math.log(k)), sizes.length - 1);
    return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i];
}

function formatMbps(mbps) {
    if (mbps == null || isNaN(mbps)) return '—';
    if (mbps >= 1000) return (mbps / 1000).toFixed(2) + ' Gbps';
    return mbps.toFixed(2) + ' Mbps';
}

function formatRate(bytesPerSec) {
    return formatBytes(bytesPerSec) + '/s';
}

function formatNumber(n) {
    return Number(n || 0).toLocaleString('pt-BR');
}

function categoryBadge(cat) {
    const labels = {
        cdn: 'CDN', netflix: 'Netflix', globo: 'Globo',
        streaming: 'Streaming', peer: 'Peer', other: 'Outro',
        social: 'Social', gaming: 'Games', dns: 'DNS',
        cloud: 'Cloud', apple: 'Apple'
    };
    return `<span class="badge badge-${cat || 'other'}">${labels[cat] || cat}</span>`;
}

function directionBadge(dir) {
    return `<span class="badge badge-${dir}">${dir === 'inbound' ? 'Entrada' : 'Saída'}</span>`;
}

function createCard(item) {
    const real = item.mbps_scaled != null && item.mbps_scaled > 0 ? item.mbps_scaled : item.mbps;
    const mbpsLine = real != null && real > 0
        ? `<div class="card-mbps card-mbps-primary">${formatMbps(real)}</div>`
        : '';
    const nfHint = item.mbps_scaled != null && item.mbps != null && item.mbps_scaled > item.mbps * 1.5
        ? `<div class="card-mbps-hint">NF amostra: ${formatMbps(item.mbps)}</div>` : '';
    const io = (item.in_mbps > 0 || item.out_mbps > 0)
        ? `<div class="card-io">${formatMbps(item.in_mbps || 0)} ↓ · ${formatMbps(item.out_mbps || 0)} ↑</div>` : '';
    return `
        <div class="dest-card cat-${item.category || 'other'}">
            <div class="card-name">${item.name}</div>
            ${mbpsLine}
            ${io}
            <div class="card-bytes">${formatBytes(item.bytes)} acumulado</div>
            ${nfHint}
            <div class="card-pct">${(item.percentage || 0).toFixed(1)}% do total</div>
            <div class="card-bar"><div class="card-bar-fill" style="width:${Math.min(item.percentage || 0, 100)}%"></div></div>
        </div>`;
}

function createTrafficRow(item, opts) {
    const o = opts || {};
    const real = item.mbps_scaled != null && item.mbps_scaled > 0 ? item.mbps_scaled : item.mbps;
    const max = o.maxMbps || real || 1;
    const pct = Math.min(100, (real / max) * 100);
    const color = o.color || 'var(--accent)';
    const rank = o.rank != null ? `<span class="traffic-rank">#${o.rank}</span>` : '';
    const io = (item.in_mbps > 0 || item.out_mbps > 0)
        ? `<span class="traffic-io">${formatMbps(item.in_mbps || 0)} ↓ · ${formatMbps(item.out_mbps || 0)} ↑</span>` : '';
    const flipKey = o.flipKey || item.name || '';
    return `
        <div class="traffic-row cat-${item.category || 'other'}" data-name="${item.name}" data-flip-key="${flipKey}">
            ${rank}
            <div class="traffic-row-head">
                <span class="traffic-name">${item.name}</span>
                <span class="traffic-mbps" style="color:${color}">${formatMbps(real)}</span>
            </div>
            <div class="traffic-bar-track">
                <div class="traffic-bar-fill" style="width:${pct}%;background:${color}"></div>
            </div>
            <div class="traffic-row-meta">
                ${io}
                <span class="traffic-bytes">${formatBytes(item.bytes)}</span>
                ${item.percentage != null ? `<span class="traffic-pct">${item.percentage.toFixed(1)}%</span>` : ''}
            </div>
        </div>`;
}

function samplingFactor(stats) {
    return (stats && stats.sampling && stats.sampling.effective > 0)
        ? stats.sampling.effective : 1;
}

function scaleMbps(mbps, stats) {
    return (mbps || 0) * samplingFactor(stats);
}

function createIfaceCard(iface) {
    const label = iface.alias || iface.name;
    return `
        <div class="dest-card cat-${iface.role === 'ix' || iface.role === 'peer' ? 'peer' : iface.role === 'cache' || iface.role === 'cdn' ? 'cdn' : 'other'}">
            <div class="card-name">${label}</div>
            <div class="card-bytes">${formatMbps(iface.in_mbps)} ↓</div>
            <div class="card-mbps">${formatMbps(iface.out_mbps)} ↑</div>
            <div class="card-pct">${iface.name} · ${iface.speed_mbps || 0} Mbps</div>
            <div class="card-bar"><div class="card-bar-fill" style="width:${Math.min(Math.max(iface.in_util_pct || 0, iface.out_util_pct || 0), 100)}%"></div></div>
        </div>`;
}

function renderCategoryBars(byCategory, total, byMbps) {
    const container = document.getElementById('category-bars');
    if (!container) return;
    const cats = Object.entries(byCategory || {}).sort((a, b) => b[1] - a[1]);
    container.innerHTML = cats.map(([cat, bytes]) => {
        const pct = total > 0 ? (bytes / total * 100) : 0;
        const mbps = byMbps && byMbps[cat] != null ? ` · ${formatMbps(byMbps[cat])}` : '';
        return `
            <div class="category-bar-item">
                <span class="category-bar-label">${cat}</span>
                <div class="category-bar-track">
                    <div class="category-bar-fill cat-fill-${cat}" style="width:${pct}%">${pct.toFixed(1)}%</div>
                </div>
                <span class="category-bar-value">${formatBytes(bytes)}${mbps}</span>
            </div>`;
    }).join('');
}

function renderConsumption(list) {
    const el = document.getElementById('consumption-grid');
    if (!el || !list) return;
    const topCat = list.length ? list.slice().sort((a, b) =>
        (b.mbps_scaled || b.mbps || 0) - (a.mbps_scaled || a.mbps || 0)
    )[0] : null;
    el.innerHTML = list.map(c => {
        const real = c.mbps_scaled != null && c.mbps_scaled > 0 ? c.mbps_scaled : c.mbps;
        const nf = c.mbps != null ? formatMbps(c.mbps) : '';
        const breathe = topCat && topCat.category === c.category ? ' cons-breathe' : '';
        return `
        <div class="consumption-card cat-${c.category}${breathe}">
            <div class="cons-label">${c.label || c.category}</div>
            <div class="cons-mbps">${formatMbps(real)}</div>
            <div class="cons-bytes">${formatBytes(c.bytes)} · ${(c.percentage || 0).toFixed(1)}%${c.mbps_scaled ? ' · NF ' + nf : ''}</div>
            <div class="card-bar"><div class="card-bar-fill cat-fill-${c.category}" style="width:${Math.min(c.percentage || 0, 100)}%"></div></div>
        </div>`;
    }).join('');
}

function prefersReducedMotion() {
    try {
        return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    } catch (e) {
        return false;
    }
}

const _tweenState = new WeakMap();

function tweenMbps(el, targetMbps, opts) {
    if (!el) return;
    const o = opts || {};
    const duration = prefersReducedMotion() ? 0 : (o.duration || 600);
    const prev = _tweenState.get(el) || { value: targetMbps || 0 };
    const from = prev.value;
    const to = targetMbps == null || isNaN(targetMbps) ? 0 : targetMbps;
    if (duration <= 0 || Math.abs(to - from) < 0.01) {
        el.textContent = formatMbps(to);
        _tweenState.set(el, { value: to });
        return;
    }
    if (prev.raf) cancelAnimationFrame(prev.raf);
    const start = performance.now();
    function frame(now) {
        const t = Math.min(1, (now - start) / duration);
        const eased = 1 - Math.pow(1 - t, 3);
        const cur = from + (to - from) * eased;
        el.textContent = formatMbps(cur);
        if (t < 1) {
            prev.raf = requestAnimationFrame(frame);
            prev.value = cur;
            _tweenState.set(el, prev);
        } else {
            prev.value = to;
            prev.raf = null;
            _tweenState.set(el, prev);
        }
    }
    prev.raf = requestAnimationFrame(frame);
    _tweenState.set(el, prev);
}

function setBarWidth(el, pct) {
    if (!el) return;
    const w = Math.max(0, Math.min(100, pct || 0));
    if (prefersReducedMotion()) {
        el.style.transition = 'none';
    }
    el.style.width = w + '%';
}

function spawnFlowParticle(trackId, category, speedMbps) {
    if (prefersReducedMotion()) return;
    const track = document.getElementById(trackId);
    if (!track) return;
    const particle = document.createElement('div');
    particle.className = `flow-particle ${category || 'other'}`;
    // Maior Mbps → partícula mais rápida (duração menor)
    const spd = Math.max(0, speedMbps || 0);
    const base = spd > 500 ? 0.9 : spd > 100 ? 1.4 : spd > 20 ? 2.2 : 3.2;
    particle.style.animationDuration = (base + Math.random() * 0.8) + 's';
    track.appendChild(particle);
    particle.addEventListener('animationend', () => particle.remove());
}

/** Atualiza lista com FLIP suave quando o ranking muda. */
function renderFlipList(container, items, keyFn, htmlFn) {
    if (!container) return;
    const reduce = prefersReducedMotion();
    const prev = new Map();
    Array.from(container.children).forEach(ch => {
        const k = ch.dataset.flipKey;
        if (k) prev.set(k, ch.getBoundingClientRect());
    });
    container.innerHTML = (items || []).map(htmlFn).join('');
    if (reduce || !prev.size) return;
    Array.from(container.children).forEach(ch => {
        const k = ch.dataset.flipKey || (keyFn && keyFn(ch));
        if (!k || !prev.has(k)) return;
        const first = prev.get(k);
        const last = ch.getBoundingClientRect();
        const dx = first.left - last.left;
        const dy = first.top - last.top;
        if (Math.abs(dx) < 1 && Math.abs(dy) < 1) return;
        ch.style.transform = `translate(${dx}px, ${dy}px)`;
        ch.style.transition = 'none';
        requestAnimationFrame(() => {
            ch.style.transition = 'transform 0.45s cubic-bezier(0.22, 1, 0.36, 1)';
            ch.style.transform = '';
        });
    });
}

async function fetchStats() {
    try { return await apiGet('/stats'); } catch (e) { return null; }
}

async function fetchFlows() {
    try { return await apiGet('/flows'); } catch (e) { return []; }
}

async function fetchSNMP() {
    try { return await apiGet('/snmp'); } catch (e) { return null; }
}

async function fetchBGP() {
    try { return await apiGet('/bgp'); } catch (e) { return null; }
}

async function fetchHistory(hours) {
    try {
        const q = hours != null && hours > 0 ? `?hours=${hours}` : '';
        return await apiGet('/history' + q);
    } catch (e) { return []; }
}

async function fetchHistoryCompare(hours) {
    try {
        return await apiGet('/history/compare?hours=' + (hours || 24));
    } catch (e) { return { current: [], previous: [] }; }
}

async function ensureAuth() {
    if (window.location.pathname === '/login') return true;
    try {
        const res = await fetch('/api/auth/check', { headers: apiHeaders() });
        const data = await res.json();
        if (data.auth_required && !data.ok) {
            window.location.href = '/login';
            return false;
        }
        return true;
    } catch (e) {
        return true;
    }
}

function logout() {
    localStorage.removeItem('inforflow_api_token');
    window.location.href = '/login';
}

document.addEventListener('DOMContentLoaded', () => { ensureAuth(); });

async function fetchAlerts() {
    try { return await apiGet('/alerts'); } catch (e) { return { active: [], recent: [] }; }
}

async function fetchSampling() {
    try { return await apiGet('/sampling'); } catch (e) { return null; }
}

function exportURL(kind, format) {
    const q = new URLSearchParams({ kind: kind || 'stats', format: format || 'json' });
    return `${API_BASE}/export?${q}`;
}

function renderAlerts(list, elId) {
    const el = document.getElementById(elId || 'alerts-list');
    if (!el) return;
    const items = list || [];
    if (!items.length) {
        el.innerHTML = '<div class="alert-item alert-ok">Nenhum alerta ativo</div>';
        return;
    }
    el.innerHTML = items.map(a => `
        <div class="alert-item alert-${a.severity || 'info'}">
            <strong>${a.title}</strong>
            <span>${a.detail || ''}</span>
        </div>`).join('');
}

function timeFilterBar(onChange) {
    return `
        <div class="time-filter" id="time-filter">
            <button data-h="0" class="tf-btn active">Ao vivo</button>
            <button data-h="1" class="tf-btn">1h</button>
            <button data-h="6" class="tf-btn">6h</button>
            <button data-h="24" class="tf-btn">24h</button>
            <a class="tf-btn" href="${exportURL('stats','csv')}" download>Export CSV</a>
        </div>`;
}

function updateSourceIP(ip) {
    document.querySelectorAll('[data-source-ip]').forEach(el => {
        if (ip) el.textContent = ip;
    });
}

window.Inforflow = {
    API_BASE, getApiToken, apiHeaders, apiGet, exportDownload,
    formatBytes, formatMbps, formatRate, formatNumber,
    categoryBadge, directionBadge, createCard, createTrafficRow, createIfaceCard,
    renderCategoryBars, renderConsumption, spawnFlowParticle, renderAlerts,
    samplingFactor, scaleMbps, prefersReducedMotion, tweenMbps, setBarWidth, renderFlipList,
    fetchStats, fetchFlows, fetchSNMP, fetchBGP, fetchHistory, fetchHistoryCompare, fetchAlerts, fetchSampling,
    ensureAuth,
    exportURL, timeFilterBar, logout
};
