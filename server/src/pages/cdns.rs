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
            <p class="page-subtitle" id="cdn-subtitle">Taxa estimada por CDN (NetFlow × amostragem SNMP)</p>
        </div>
        <div class="header-actions">
            <div class="live-indicator"><span class="pulse-dot"></span> Ao vivo</div>
            <span class="sampling-chip" id="page-sampling-chip" title="Fator de amostragem">fator —</span>
            <a class="export-link" id="export-csv" href="/api/export?kind=cdn&amp;format=csv">Export CSV</a>
            <span id="cdn-compare-hint" class="section-hint" style="margin:0">Comparativo 24h…</span>
        </div>
    </header>

    <div id="cdn-alerts" class="alerts-strip fade-in-up"></div>

    <section class="stats-grid fade-in-up" style="animation-delay: 0.05s">
        <div class="stat-card stat-card-highlight">
            <div class="stat-icon" style="background: linear-gradient(135deg, #f59e0b, #fbbf24)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 7l10 5 10-5-10-5z"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">CDN total (Mbps est.)</span>
                <span class="stat-value" id="cdn-total-mbps">—</span>
                <span class="stat-rate" id="cdn-total-hint">janela ~10s × amostragem</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #10b981, #34d399)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">% do uplink SNMP</span>
                <span class="stat-value" id="cdn-uplink-share">—</span>
                <span class="stat-rate" id="cdn-snmp-uplink">entrada / saída</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #6366f1, #818cf8)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">Cache hit (est.)</span>
                <span class="stat-value" id="cdn-cache-hit">—</span>
                <span class="stat-rate" id="cdn-cache-hint">interfaces cache/CDN</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #f48120, #faad3f)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 3l14 9-14 9V3z"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">IPv4 / IPv6 · Top</span>
                <span class="stat-value" id="cdn-v4v6" style="font-size:1.1rem">—</span>
                <span class="stat-rate" id="cdn-top-name">—</span>
            </div>
        </div>
    </section>

    <section class="traffic-visual fade-in-up" style="animation-delay: 0.12s">
        <div class="cdn-pipeline">
            <div class="pipeline-node origin-node">
                <div class="node-label" id="cdn-exporter-label">exporter</div>
            </div>
            <div class="pipeline-track cdn-track" id="cdn-pipeline"></div>
            <div class="cdn-nodes" id="cdn-nodes"></div>
        </div>
    </section>

    <section class="fade-in-up" style="animation-delay: 0.15s">
        <div class="time-filter" id="cdn-time-filter">
            <button type="button" data-h="0" class="tf-btn active">Ao vivo</button>
            <button type="button" data-h="1" class="tf-btn">1h</button>
            <button type="button" data-h="6" class="tf-btn">6h</button>
            <button type="button" data-h="24" class="tf-btn">24h</button>
        </div>
        <h3 class="section-title">Histórico por CDN</h3>
        <div class="chart-wrap">
            <canvas id="cdn-history-chart" height="240"></canvas>
        </div>
        <div class="chart-legend" id="cdn-chart-legend"></div>
    </section>

    <div class="cards-row fade-in-up" style="animation-delay: 0.2s">
        <section class="card-section card-section-wide">
            <div class="section-title-row" style="display:flex;flex-wrap:wrap;gap:12px;align-items:center;justify-content:space-between;margin-bottom:12px">
                <div>
                    <h3 class="section-title" style="margin:0">Taxa por CDN (Mbps estimado)</h3>
                    <p class="section-hint" id="cdn-bytes-hint">Mbps = janela ~10s · Bytes / % = acumulado</p>
                </div>
                <div style="display:flex;flex-wrap:wrap;gap:8px;align-items:center">
                    <div class="time-filter" id="cdn-chip-filter" style="margin:0">
                        <button type="button" data-chip="" class="tf-btn active">Todos</button>
                        <button type="button" data-chip="Cloudflare" class="tf-btn">Cloudflare</button>
                        <button type="button" data-chip="Akamai" class="tf-btn">Akamai</button>
                        <button type="button" data-chip="Google" class="tf-btn">Google</button>
                        <button type="button" data-chip="AWS" class="tf-btn">AWS</button>
                        <button type="button" data-chip="other" class="tf-btn">Outros</button>
                    </div>
                    <input type="search" id="cdn-search" class="login-input" placeholder="Filtrar CDN…" style="max-width:180px;margin:0" />
                </div>
            </div>
            <div class="traffic-list" id="cdn-traffic-list"></div>
        </section>
        <section class="card-section">
            <h3 class="section-title">Interfaces SNMP (cache/CDN)</h3>
            <div class="cards-grid" id="cdn-snmp-ifaces"></div>
            <h3 class="section-title" style="margin-top:24px">Feeds de prefixos</h3>
            <div id="cdn-feeds" class="section-hint">—</div>
        </section>
    </div>

    <section class="flow-table-section fade-in-up" style="animation-delay: 0.3s">
        <h3 class="section-title">Flows CDN recentes</h3>
        <div class="flow-table-wrapper">
            <table class="flow-table">
                <thead>
                    <tr>
                        <th>CDN</th>
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
