use super::layout::PageMeta;

pub fn meta() -> PageMeta {
    PageMeta {
        title: "Peer Detalhe".to_string(),
        active: "peersdetail".to_string(),
    }
}

pub fn content() -> String {
    r##"
    <header class="page-header">
        <div>
            <p class="section-hint" style="margin:0 0 4px"><a href="/peers">← Peers</a></p>
            <h2 class="page-title" id="pd-title">Peer</h2>
            <p class="page-subtitle" id="pd-sub">Sessões BGP, série e flows deste ASN</p>
        </div>
        <div class="header-actions">
            <span class="sampling-chip" id="page-sampling-chip">fator —</span>
            <a class="export-link" id="pd-asn-link" href="/asn">Ver em ASN</a>
        </div>
    </header>

    <section class="stats-grid fade-in-up">
        <div class="stat-card stat-card-highlight">
            <div class="stat-info">
                <span class="stat-label">Mbps peer (10s est.)</span>
                <span class="stat-value" id="pd-mbps">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">Sessões up / total</span>
                <span class="stat-value" id="pd-sessions">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">Bytes</span>
                <span class="stat-value" id="pd-bytes">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">Papel</span>
                <span class="stat-value" id="pd-role" style="font-size:1.1rem">—</span>
            </div>
        </div>
    </section>

    <section class="fade-in-up" style="animation-delay:0.08s">
        <div class="time-filter" id="pd-time-filter">
            <button type="button" data-h="1" class="tf-btn">1h</button>
            <button type="button" data-h="6" class="tf-btn active">6h</button>
            <button type="button" data-h="24" class="tf-btn">24h</button>
        </div>
        <h3 class="section-title">Histórico deste peer ASN</h3>
        <div class="chart-wrap">
            <canvas id="pd-chart" height="240"></canvas>
        </div>
    </section>

    <section class="flow-table-section fade-in-up" style="animation-delay:0.12s">
        <h3 class="section-title">Sessões BGP deste ASN</h3>
        <div class="flow-table-wrapper">
            <table class="flow-table">
                <thead>
                    <tr>
                        <th>Peer IP</th>
                        <th>Estado</th>
                        <th>Mbps est.</th>
                        <th>Uptime</th>
                        <th>Flaps</th>
                        <th>Updates</th>
                    </tr>
                </thead>
                <tbody id="pd-sessions-body"></tbody>
            </table>
        </div>
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
                <tbody id="pd-flows"></tbody>
            </table>
        </div>
    </section>
    "##.to_string()
}
