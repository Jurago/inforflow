use super::layout::PageMeta;

pub fn meta() -> PageMeta {
    PageMeta {
        title: "CDN Detalhe".to_string(),
        active: "cdndetail".to_string(),
    }
}

pub fn content() -> String {
    r##"
    <header class="page-header">
        <div>
            <p class="section-hint" style="margin:0 0 4px"><a href="/cdns">← CDNs</a></p>
            <h2 class="page-title" id="cd-title">CDN</h2>
            <p class="page-subtitle" id="cd-sub">Mbps estimado · SNMP · flows</p>
        </div>
        <div class="header-actions">
            <span class="sampling-chip" id="page-sampling-chip">fator —</span>
            <a class="export-link" id="cd-asn-link" href="/asn">Ver ASN</a>
            <a class="export-link" id="export-csv" href="/api/export?kind=cdn&amp;format=csv">Export CSV</a>
        </div>
    </header>

    <div id="cd-alerts" class="alerts-strip fade-in-up"></div>

    <section class="stats-grid fade-in-up">
        <div class="stat-card stat-card-highlight">
            <div class="stat-info">
                <span class="stat-label">Mbps estimado (10s)</span>
                <span class="stat-value" id="cd-mbps">—</span>
                <span class="stat-rate" id="cd-mbps-hint">NetFlow × amostragem</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">Bytes acumulados</span>
                <span class="stat-value" id="cd-bytes">—</span>
                <span class="stat-rate" id="cd-pct">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">IPv4 / IPv6</span>
                <span class="stat-value" id="cd-v4v6" style="font-size:1.1rem">—</span>
                <span class="stat-rate" id="cd-io">↓ / ↑</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">SNMP cache match</span>
                <span class="stat-value" id="cd-snmp">—</span>
                <span class="stat-rate" id="cd-snmp-hint">ifaces correlacionadas</span>
            </div>
        </div>
    </section>

    <section class="fade-in-up" style="animation-delay:0.08s">
        <div class="time-filter" id="cd-time-filter">
            <button type="button" data-h="1" class="tf-btn">1h</button>
            <button type="button" data-h="6" class="tf-btn active">6h</button>
            <button type="button" data-h="24" class="tf-btn">24h</button>
        </div>
        <h3 class="section-title">Histórico deste CDN</h3>
        <div class="chart-wrap">
            <canvas id="cd-chart" height="240"></canvas>
        </div>
    </section>

    <section class="fade-in-up" style="animation-delay:0.12s">
        <h3 class="section-title">Interfaces SNMP correlacionadas</h3>
        <div class="cards-grid" id="cd-snmp-ifaces"></div>
    </section>

    <section class="flow-table-section fade-in-up" style="animation-delay:0.18s">
        <h3 class="section-title">Flows recentes</h3>
        <div class="flow-table-wrapper">
            <table class="flow-table">
                <thead>
                    <tr>
                        <th>Hora</th>
                        <th>Origem → Destino</th>
                        <th>ASN</th>
                        <th>Bytes</th>
                        <th>Dir.</th>
                    </tr>
                </thead>
                <tbody id="cd-flows"></tbody>
            </table>
        </div>
    </section>
    "##.to_string()
}
