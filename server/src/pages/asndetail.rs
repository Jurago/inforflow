use super::layout::PageMeta;

pub fn meta() -> PageMeta {
    PageMeta {
        title: "ASN Detalhe".to_string(),
        active: "asndetail".to_string(),
    }
}

pub fn content() -> String {
    r##"
    <header class="page-header">
        <div>
            <p class="section-hint" style="margin:0 0 4px"><a href="/asn">← ASN</a></p>
            <h2 class="page-title" id="asn-detail-title">ASN</h2>
            <p class="page-subtitle" id="asn-detail-sub">Série, peers e flows recentes deste ASN</p>
        </div>
        <div class="header-actions">
            <span class="sampling-chip" id="page-sampling-chip">fator —</span>
            <a class="export-link" id="asn-detail-flows-link" href="/flows">Ver em Flows</a>
        </div>
    </header>

    <section class="stats-grid fade-in-up">
        <div class="stat-card stat-card-highlight">
            <div class="stat-info">
                <span class="stat-label">Mbps dest. (10s est.)</span>
                <span class="stat-value" id="asd-mbps">—</span>
                <span class="stat-rate" id="asd-role">destino</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">Mbps peer (10s)</span>
                <span class="stat-value" id="asd-peer-mbps">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">Bytes hoje</span>
                <span class="stat-value" id="asd-bytes">—</span>
                <span class="stat-rate" id="asd-pct">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">↓ / ↑ (10s)</span>
                <span class="stat-value" id="asd-io" style="font-size:1.1rem">—</span>
            </div>
        </div>
    </section>

    <section class="fade-in-up" style="animation-delay:0.08s">
        <div class="time-filter" id="asd-time-filter">
            <button type="button" data-h="1" class="tf-btn">1h</button>
            <button type="button" data-h="6" class="tf-btn active">6h</button>
            <button type="button" data-h="24" class="tf-btn">24h</button>
        </div>
        <h3 class="section-title">Histórico deste ASN</h3>
        <div class="chart-wrap">
            <canvas id="asd-chart" height="240"></canvas>
        </div>
        <div class="chart-legend" id="asd-legend"></div>
    </section>

    <section class="flow-table-section fade-in-up" style="animation-delay:0.15s">
        <h3 class="section-title">Flows recentes (índice por ASN)</h3>
        <p class="section-hint">Até 80 flows mantidos por ASN no coletor (além do ring global)</p>
        <div class="flow-table-wrapper">
            <table class="flow-table">
                <thead>
                    <tr>
                        <th>Hora</th>
                        <th>Origem → Destino</th>
                        <th>Peer ASN</th>
                        <th>Cat.</th>
                        <th>Bytes</th>
                        <th>Dir.</th>
                    </tr>
                </thead>
                <tbody id="asd-flows"></tbody>
            </table>
        </div>
    </section>
    "##.to_string()
}
