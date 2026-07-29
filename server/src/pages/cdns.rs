use super::layout::PageMeta;

pub fn meta() -> PageMeta {
    PageMeta {
        title: "CDNs".to_string(),
        active: "cdns".to_string(),
    }
}

pub fn content() -> String {
    r##"
    <header class="page-header">
        <div>
            <h2 class="page-title">Análise de CDNs</h2>
            <p class="page-subtitle">Taxa real estimada por CDN (NetFlow × amostragem SNMP) · 170.245.127.191</p>
        </div>
        <div class="header-actions">
            <div class="live-indicator"><span class="pulse-dot"></span> Ao vivo</div>
            <span id="cdn-compare-hint" class="section-hint" style="margin:0">Comparativo 24h…</span>
        </div>
    </header>

    <section class="stats-grid fade-in-up" style="animation-delay: 0.05s">
        <div class="stat-card stat-card-highlight">
            <div class="stat-icon" style="background: linear-gradient(135deg, #f59e0b, #fbbf24)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 7l10 5 10-5-10-5z"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">CDN total (estimado)</span>
                <span class="stat-value" id="cdn-total-mbps">—</span>
                <span class="stat-rate" id="cdn-total-hint">NetFlow × fator SNMP</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #10b981, #34d399)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">SNMP Uplink</span>
                <span class="stat-value" id="cdn-snmp-uplink">—</span>
                <span class="stat-rate">referência total roteador</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #6366f1, #818cf8)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v4M12 19v4"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">CDNs ativos</span>
                <span class="stat-value" id="cdn-count">—</span>
                <span class="stat-rate" id="cdn-sampling-hint">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #f48120, #faad3f)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 7l10 5 10-5-10-5z"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">Top CDN agora</span>
                <span class="stat-value" id="cdn-top-mbps">—</span>
                <span class="stat-rate" id="cdn-top-name">—</span>
            </div>
        </div>
    </section>

    <section class="traffic-visual fade-in-up" style="animation-delay: 0.15s">
        <div class="cdn-pipeline">
            <div class="pipeline-node origin-node">
                <div class="node-label">170.245.127.191</div>
            </div>
            <div class="pipeline-track cdn-track" id="cdn-pipeline"></div>
            <div class="cdn-nodes" id="cdn-nodes"></div>
        </div>
    </section>

    <div class="cards-row fade-in-up" style="animation-delay: 0.2s">
        <section class="card-section card-section-wide">
            <h3 class="section-title">Taxa por CDN (Mbps estimado)</h3>
            <div class="traffic-list" id="cdn-traffic-list"></div>
        </section>
        <section class="card-section">
            <h3 class="section-title">Comparativo SNMP cache</h3>
            <div class="cards-grid" id="cdn-snmp-ifaces"></div>
        </section>
    </div>

    <section class="fade-in-up" style="animation-delay: 0.3s">
        <h3 class="section-title">Gráfico de barras — Mbps por CDN</h3>
        <div class="cdn-chart cdn-chart-mbps" id="cdn-chart"></div>
    </section>

    <section class="card-section fade-in-up" style="animation-delay: 0.35s">
        <h3 class="section-title">Detalhes por CDN</h3>
        <div class="cards-grid" id="cdn-detail-cards"></div>
    </section>

    <section class="flow-table-section fade-in-up" style="animation-delay: 0.4s">
        <h3 class="section-title">Flows CDN recentes</h3>
        <div class="flow-table-wrapper">
            <table class="flow-table">
                <thead>
                    <tr>
                        <th>CDN</th>
                        <th>Mbps est.</th>
                        <th>ASN</th>
                        <th>Origem → Destino</th>
                        <th>Bytes</th>
                        <th>Direção</th>
                    </tr>
                </thead>
                <tbody id="cdn-table-body"></tbody>
            </table>
        </div>
    </section>
    "##.to_string()
}
