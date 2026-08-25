use super::layout::PageMeta;

pub fn meta() -> PageMeta {
    PageMeta {
        title: "Streaming".to_string(),
        active: "streaming".to_string(),
    }
}

pub fn content() -> String {
    r##"
    <header class="page-header">
        <div>
            <h2 class="page-title">Streaming</h2>
            <p class="page-subtitle" id="stream-subtitle">Taxa estimada (NetFlow × amostragem SNMP) · caches via SNMP</p>
        </div>
        <div class="header-actions">
            <div class="live-indicator"><span class="pulse-dot"></span> Ao vivo</div>
            <span class="sampling-chip" id="page-sampling-chip" title="Fator de amostragem">fator —</span>
            <a class="export-link" id="export-csv" href="/api/export?kind=streaming&amp;format=csv">Export CSV</a>
        </div>
    </header>

    <div id="stream-alerts" class="alerts-strip fade-in-up"></div>

    <section class="stats-grid fade-in-up" style="animation-delay: 0.05s">
        <div class="stat-card stat-card-highlight">
            <div class="stat-icon" style="background: linear-gradient(135deg, #8b5cf6, #a78bfa)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">Streaming total (Mbps est.)</span>
                <span class="stat-value" id="stream-total-mbps">—</span>
                <span class="stat-rate" id="stream-total-hint">janela ~10s × amostragem</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #10b981, #34d399)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 7l10 5 10-5-10-5z"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">% do uplink SNMP</span>
                <span class="stat-value" id="stream-uplink-share">—</span>
                <span class="stat-rate" id="stream-snmp-uplink">entrada / saída</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #f59e0b, #fbbf24)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">Cache hit (est.)</span>
                <span class="stat-value" id="stream-cache-hit">—</span>
                <span class="stat-rate" id="stream-snmp-cache-label">interfaces cache</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #06b6d4, #22d3ee)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 3l14 9-14 9V3z"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">IPv4 / IPv6</span>
                <span class="stat-value" id="stream-v4v6" style="font-size:1.15rem">—</span>
                <span class="stat-rate" id="stream-count">serviços</span>
            </div>
        </div>
    </section>

    <section class="streaming-hero fade-in-up" style="animation-delay: 0.1s">
        <h3 class="section-title">Top serviços (Mbps estimado)</h3>
        <div class="hero-cards" id="stream-hero-cards"></div>
    </section>

    <section class="fade-in-up" style="animation-delay: 0.12s">
        <div class="time-filter" id="stream-time-filter">
            <button type="button" data-h="0" class="tf-btn active">Ao vivo</button>
            <button type="button" data-h="1" class="tf-btn">1h</button>
            <button type="button" data-h="6" class="tf-btn">6h</button>
            <button type="button" data-h="24" class="tf-btn">24h</button>
        </div>
        <h3 class="section-title">Histórico por serviço</h3>
        <div class="chart-wrap">
            <canvas id="stream-history-chart" height="240"></canvas>
        </div>
        <div class="chart-legend" id="stream-chart-legend"></div>
    </section>

    <div class="cards-row fade-in-up" style="animation-delay: 0.15s">
        <section class="card-section card-section-wide">
            <div class="section-title-row" style="display:flex;flex-wrap:wrap;gap:12px;align-items:center;justify-content:space-between;margin-bottom:12px">
                <div>
                    <h3 class="section-title" style="margin:0">Tráfego por serviço</h3>
                    <p class="section-hint" id="stream-bytes-hint">Mbps = janela ~10s · Bytes / % = acumulado do dia</p>
                </div>
                <div style="display:flex;flex-wrap:wrap;gap:8px;align-items:center">
                    <div class="time-filter" id="stream-cat-filter" style="margin:0">
                        <button type="button" data-cat="" class="tf-btn active">Todos</button>
                        <button type="button" data-cat="streaming" class="tf-btn">Streaming</button>
                        <button type="button" data-cat="netflix" class="tf-btn">Netflix</button>
                        <button type="button" data-cat="globo" class="tf-btn">Globo</button>
                        <button type="button" data-cat="apple" class="tf-btn">Apple</button>
                        <button type="button" data-cat="social" class="tf-btn">Social</button>
                    </div>
                    <input type="search" id="stream-search" class="login-input" placeholder="Filtrar serviço…" style="max-width:200px;margin:0" />
                </div>
            </div>
            <div class="traffic-list" id="streaming-traffic-list"></div>
        </section>
        <section class="card-section">
            <h3 class="section-title">Entrada / Saída</h3>
            <div class="traffic-io-summary" id="stream-io-summary"></div>
            <h3 class="section-title" style="margin-top:24px">Interfaces SNMP (cache)</h3>
            <div class="cards-grid" id="streaming-snmp-ifaces"></div>
        </section>
    </div>

    <section class="traffic-visual fade-in-up" style="animation-delay: 0.25s">
        <div class="streaming-flow" id="streaming-flow">
            <div class="stream-source">
                <div class="source-ring"></div>
                <span id="stream-exporter-label">exporter</span>
            </div>
            <div class="stream-rays" id="stream-rays"></div>
            <div class="stream-targets" id="stream-targets"></div>
        </div>
    </section>

    <section class="flow-table-section fade-in-up" style="animation-delay: 0.35s">
        <h3 class="section-title">Flows recentes de streaming</h3>
        <div class="flow-table-wrapper">
            <table class="flow-table">
                <thead>
                    <tr>
                        <th>Hora</th>
                        <th>Serviço</th>
                        <th>Origem → Destino</th>
                        <th>Cat.</th>
                        <th>Bytes</th>
                        <th>Dir.</th>
                    </tr>
                </thead>
                <tbody id="streaming-flows-body"></tbody>
            </table>
        </div>
    </section>
    "##.to_string()
}
