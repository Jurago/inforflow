use super::layout::PageMeta;

pub fn meta() -> PageMeta {
    PageMeta {
        title: "Cache Hit Ratio".to_string(),
        active: "cache".to_string(),
    }
}

pub fn content() -> String {
    r##"
    <header class="page-header">
        <div>
            <h2 class="page-title">Cache Hit Ratio</h2>
            <p class="page-subtitle">Interfaces SNMP cache vs streaming classificado (NetFlow×SNMP)</p>
        </div>
        <div class="header-actions"><div class="live-indicator"><span class="pulse-dot"></span> Ao vivo</div></div>
    </header>
    <section class="stats-grid fade-in-up">
        <div class="stat-card stat-card-highlight">
            <div class="stat-info">
                <span class="stat-label">Hit estimado</span>
                <span class="stat-value" id="cache-hit-pct">—</span>
                <span class="stat-rate">cache SNMP / streaming</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">Cache SNMP</span>
                <span class="stat-value" id="cache-snmp-total">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">Streaming est.</span>
                <span class="stat-value" id="cache-stream-total">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">Netflix est.</span>
                <span class="stat-value" id="cache-netflix">—</span>
            </div>
        </div>
    </section>
    <div class="cards-row fade-in-up">
        <section class="card-section card-section-wide">
            <h3 class="section-title">Interfaces cache (SNMP)</h3>
            <div class="cards-grid" id="cache-ifaces"></div>
        </section>
        <section class="card-section">
            <h3 class="section-title">IPv4 / IPv6 (NetFlow est.)</h3>
            <div class="traffic-io-summary" id="cache-ip-summary"></div>
            <h3 class="section-title" style="margin-top:20px">Armazenamento 30 dias</h3>
            <div id="storage-info" class="sampling-explain">—</div>
        </section>
    </div>
    "##.to_string()
}
