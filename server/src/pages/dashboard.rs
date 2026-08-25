use super::layout::PageMeta;

pub fn meta() -> PageMeta {
    PageMeta {
        title: "Dashboard".to_string(),
        active: "dashboard".to_string(),
    }
}

pub fn content() -> String {
    r##"
    <header class="page-header">
        <div>
            <h2 class="page-title">Dashboard ISP</h2>
            <p class="page-subtitle" id="dash-subtitle">SNMP = taxa real · NetFlow × amostragem = estimado</p>
        </div>
        <div class="header-actions">
            <div class="live-indicator"><span class="pulse-dot"></span> Ao vivo</div>
            <span class="sampling-chip" id="page-sampling-chip">fator —</span>
            <span id="dash-compare" class="section-hint" style="margin:0">vs 24h…</span>
            <a class="export-link" id="export-csv" href="/api/export?kind=dashboard&amp;format=csv">Export CSV</a>
            <span id="alert-badge" class="alert-badge" style="display:none"></span>
        </div>
    </header>

    <div id="alerts-list" class="alerts-strip fade-in-up"></div>
    <div id="dash-status" class="alerts-strip fade-in-up" style="animation-delay:0.03s"></div>
    <section class="ops-shortcuts fade-in-up" id="ops-shortcuts" style="animation-delay:0.04s" aria-label="Atalhos operacionais"></section>

    <section class="stats-grid fade-in-up" style="animation-delay: 0.05s">
        <div class="stat-card stat-card-highlight">
            <div class="stat-icon" style="background:linear-gradient(135deg,#0ea5e9,#38bdf8)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">SNMP Uplink</span>
                <span class="stat-value" id="snmp-uplink-mbps" style="font-size:1.15rem">—</span>
                <span class="stat-rate" id="snmp-router-name">roteador</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background:linear-gradient(135deg,#3b82f6,#60a5fa)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">NetFlow estimado</span>
                <span class="stat-value" id="nf-scaled-mbps">—</span>
                <span class="stat-rate" id="nf-raw-hint">bruto —</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background:linear-gradient(135deg,#f59e0b,#fbbf24)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 7l10 5 10-5-10-5z"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">Gap NF × SNMP</span>
                <span class="stat-value" id="dash-gap">—</span>
                <span class="stat-rate" id="dash-gap-pct">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon category-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">CPU / Mem · v4/v6</span>
                <span class="stat-value" id="snmp-cpu-mem" style="font-size:1.1rem">—</span>
                <span class="stat-rate" id="dash-v4v6">—</span>
            </div>
        </div>
    </section>

    <section class="page-toolbar fade-in-up" style="animation-delay:0.08s">
        <div id="dash-blocks" class="time-filter" style="flex-wrap:wrap"></div>
    </section>

    <section class="chart-panel fade-in-up" style="animation-delay:0.1s">
        <div class="chart-panel-head">
            <div>
                <h3 class="section-title">Última 1h <a href="/graphs?h=1" class="section-hint" style="font-weight:400">Abrir gráficos →</a></h3>
                <p class="chart-panel-desc">NetFlow estimado vs SNMP uplink</p>
            </div>
        </div>
        <div class="chart-wrap">
            <canvas id="dash-spark" height="160"></canvas>
        </div>
    </section>

    <section class="fade-in-up" style="animation-delay: 0.14s">
        <h3 class="section-title">Consumo por categoria <span style="font-weight:400;font-size:0.85rem;color:var(--text-muted)">(clique para detalhe)</span></h3>
        <div class="consumption-grid" id="consumption-grid"></div>
        <div class="category-bars" id="category-bars" style="margin-top:16px"></div>
    </section>

    <section class="traffic-visual fade-in-up" style="animation-delay: 0.16s">
        <div class="traffic-pipeline">
            <div class="pipeline-node origin-node">
                <div class="node-label">Borda</div>
                <div class="node-ip" id="dash-exporter" data-source-ip>—</div>
            </div>
            <div class="pipeline-track" id="pipeline-track"></div>
            <div class="pipeline-cats" id="pipeline-cats" style="display:flex;flex-wrap:wrap;gap:8px;align-items:center;min-width:140px"></div>
        </div>
    </section>

    <section class="fade-in-up" style="animation-delay: 0.18s">
        <h3 class="section-title">BGP <span id="bgp-summary-hint" style="font-weight:400;font-size:0.85rem;color:var(--text-muted)"></span>
            <a href="/peers" class="section-hint" style="font-weight:400">Peers →</a></h3>
        <div class="cards-grid" id="bgp-summary"></div>
    </section>

    <section class="fade-in-up" style="animation-delay: 0.2s">
        <h3 class="section-title">Interfaces críticas (SNMP) <a href="/router" class="section-hint" style="font-weight:400">Roteador →</a></h3>
        <div class="cards-grid" id="snmp-critical"></div>
    </section>

    <section class="fade-in-up" style="animation-delay: 0.22s">
        <h3 class="section-title">Top clientes CGNAT <span style="font-weight:400;font-size:0.85rem;color:var(--text-muted)">(100.64.x)</span></h3>
        <div class="flow-table-wrapper">
            <table class="flow-table">
                <thead>
                    <tr>
                        <th>IP</th>
                        <th>Mbps est.</th>
                        <th>Categoria</th>
                        <th>Bytes</th>
                        <th>Flows</th>
                    </tr>
                </thead>
                <tbody id="talkers-body"></tbody>
            </table>
        </div>
    </section>

    <div class="cards-row fade-in-up" style="animation-delay: 0.25s">
        <section class="card-section">
            <h3 class="section-title">Top Destinos / Serviços</h3>
            <div class="cards-grid" id="top-destinations"></div>
        </section>
        <section class="card-section">
            <h3 class="section-title">Top Origens / Serviços</h3>
            <div class="cards-grid" id="top-origins"></div>
        </section>
    </div>

    <section class="flow-table-section fade-in-up" style="animation-delay: 0.28s">
        <h3 class="section-title">Flows recentes <a href="/flows" class="section-hint" style="font-weight:400">Explorar →</a></h3>
        <div class="flow-table-wrapper">
            <table class="flow-table">
                <thead>
                    <tr>
                        <th>Hora</th>
                        <th>Origem → Destino</th>
                        <th>ASN</th>
                        <th>Cat.</th>
                        <th>Bytes</th>
                        <th>Dir.</th>
                    </tr>
                </thead>
                <tbody id="flow-table-body"></tbody>
            </table>
        </div>
    </section>
    "##.to_string()
}
