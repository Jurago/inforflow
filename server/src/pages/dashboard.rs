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
            <p class="page-subtitle">SNMP = taxa real de uplink · categorias = NetFlow × amostragem (estimado)</p>
        </div>
        <div class="header-actions">
            <div class="live-indicator"><span class="pulse-dot"></span> Ao vivo</div>
            <a class="export-link" id="export-csv" href="#">Export CSV</a>
            <span id="alert-badge" class="alert-badge" style="display:none"></span>
        </div>
    </header>

    <div id="alerts-list" class="alerts-strip fade-in-up"></div>

    <section class="fade-in-up" style="animation-delay: 0.08s">
        <h3 class="section-title">BGP <span id="bgp-summary-hint" style="font-weight:400;font-size:0.85rem;color:var(--text-muted)"></span></h3>
        <div class="cards-grid" id="bgp-summary"></div>
    </section>

    <section class="stats-grid fade-in-up" style="animation-delay: 0.05s">
        <div class="stat-card">
            <div class="stat-icon bytes-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">NetFlow</span>
                <span class="stat-value" id="nf-mbps">—</span>
                <span class="stat-rate" id="bytes-rate">—</span>
            </div>
        </div>
        <div class="stat-card">
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
            <div class="stat-icon" style="background:linear-gradient(135deg,#f59e0b,#fbbf24)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 7l10 5 10-5-10-5z"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">NetFlow × amostragem</span>
                <span class="stat-value" id="nf-scaled-mbps" style="font-size:1.15rem">—</span>
                <span class="stat-rate" id="sampling-info">fator —</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon category-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">CPU / Mem</span>
                <span class="stat-value" id="snmp-cpu-mem" style="font-size:1.15rem">—</span>
                <span class="stat-rate" id="snmp-uptime">—</span>
            </div>
        </div>
    </section>

    <section class="traffic-visual fade-in-up" style="animation-delay: 0.12s">
        <div class="traffic-pipeline">
            <div class="pipeline-node origin-node">
                <div class="node-label">Borda</div>
                <div class="node-ip" data-source-ip>—</div>
            </div>
            <div class="pipeline-track" id="pipeline-track"></div>
            <div class="pipeline-node dest-node">
                <div class="node-label">Serviços</div>
                <div class="node-count" id="dest-count">—</div>
            </div>
        </div>
    </section>

    <section class="fade-in-up" style="animation-delay: 0.18s">
        <h3 class="section-title">Consumo por categoria <span style="font-weight:400;font-size:0.85rem;color:var(--text-muted)">(Mbps estimado ≈ SNMP)</span></h3>
        <div class="consumption-grid" id="consumption-grid"></div>
        <div class="category-bars" id="category-bars" style="margin-top:16px"></div>
    </section>

    <section class="fade-in-up" style="animation-delay: 0.22s">
        <h3 class="section-title">Interfaces críticas (SNMP)</h3>
        <div class="cards-grid" id="snmp-critical"></div>
    </section>

    <section class="fade-in-up" style="animation-delay: 0.25s">
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

    <div class="cards-row fade-in-up" style="animation-delay: 0.28s">
        <section class="card-section">
            <h3 class="section-title">Top Destinos / Serviços</h3>
            <div class="cards-grid" id="top-destinations"></div>
        </section>
        <section class="card-section">
            <h3 class="section-title">Top Origens / Serviços</h3>
            <div class="cards-grid" id="top-origins"></div>
        </section>
    </div>

    <section class="flow-table-section fade-in-up" style="animation-delay: 0.35s">
        <h3 class="section-title">Flows recentes</h3>
        <div class="flow-table-wrapper">
            <table class="flow-table">
                <thead>
                    <tr>
                        <th>Hora</th>
                        <th>Origem</th>
                        <th>Destino</th>
                        <th>Proto</th>
                        <th>Bytes</th>
                        <th>Categoria</th>
                        <th>Direção</th>
                    </tr>
                </thead>
                <tbody id="flow-table-body"></tbody>
            </table>
        </div>
    </section>
    "##.to_string()
}
