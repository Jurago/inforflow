use super::layout::PageMeta;

pub fn meta() -> PageMeta {
    PageMeta {
        title: "Streaming Detalhe".to_string(),
        active: "streamingdetail".to_string(),
    }
}

pub fn content() -> String {
    r##"
    <header class="page-header">
        <div>
            <p class="section-hint" style="margin:0 0 4px"><a href="/streaming">← Streaming</a></p>
            <h2 class="page-title" id="sd-title">Serviço</h2>
            <p class="page-subtitle" id="sd-sub">Mbps estimado · SNMP cache · flows</p>
        </div>
        <div class="header-actions">
            <span class="sampling-chip" id="page-sampling-chip">fator —</span>
            <a class="export-link" id="export-csv" href="/api/export?kind=streaming&amp;format=csv">Export CSV</a>
        </div>
    </header>

    <div id="sd-alerts" class="alerts-strip fade-in-up"></div>

    <section class="stats-grid fade-in-up">
        <div class="stat-card stat-card-highlight">
            <div class="stat-info">
                <span class="stat-label">Mbps estimado (10s)</span>
                <span class="stat-value" id="sd-mbps">—</span>
                <span class="stat-rate" id="sd-mbps-hint">NetFlow × amostragem</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">Bytes acumulados</span>
                <span class="stat-value" id="sd-bytes">—</span>
                <span class="stat-rate" id="sd-pct">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">IPv4 / IPv6</span>
                <span class="stat-value" id="sd-v4v6" style="font-size:1.1rem">—</span>
                <span class="stat-rate" id="sd-io">↓ / ↑</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">SNMP cache match</span>
                <span class="stat-value" id="sd-snmp">—</span>
                <span class="stat-rate" id="sd-snmp-hint">ifaces correlacionadas</span>
            </div>
        </div>
    </section>

    <section class="fade-in-up" style="animation-delay:0.08s">
        <div class="time-filter" id="sd-time-filter">
            <button type="button" data-h="1" class="tf-btn">1h</button>
            <button type="button" data-h="6" class="tf-btn active">6h</button>
            <button type="button" data-h="24" class="tf-btn">24h</button>
        </div>
        <h3 class="section-title">Histórico deste serviço</h3>
        <div class="chart-wrap">
            <canvas id="sd-chart" height="240"></canvas>
        </div>
    </section>

    <section class="fade-in-up" style="animation-delay:0.12s">
        <h3 class="section-title">Interfaces SNMP correlacionadas</h3>
        <div class="cards-grid" id="sd-snmp-ifaces"></div>
    </section>

    <section class="flow-table-section fade-in-up" style="animation-delay:0.18s">
        <h3 class="section-title">Flows recentes</h3>
        <div class="flow-table-wrapper">
            <table class="flow-table">
                <thead>
                    <tr>
                        <th>Hora</th>
                        <th>Origem → Destino</th>
                        <th>Cat.</th>
                        <th>Bytes</th>
                        <th>Dir.</th>
                    </tr>
                </thead>
                <tbody id="sd-flows"></tbody>
            </table>
        </div>
    </section>
    "##.to_string()
}
