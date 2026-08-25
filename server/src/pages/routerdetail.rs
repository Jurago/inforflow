use super::layout::PageMeta;

pub fn meta() -> PageMeta {
    PageMeta {
        title: "Iface Detalhe".to_string(),
        active: "routerdetail".to_string(),
    }
}

pub fn content() -> String {
    r##"
    <header class="page-header">
        <div>
            <p class="section-hint" style="margin:0 0 4px"><a href="/router">← Roteador</a></p>
            <h2 class="page-title" id="rd-title">Interface</h2>
            <p class="page-subtitle" id="rd-sub">SNMP · NetFlow correlacionado</p>
        </div>
        <div class="header-actions">
            <span class="sampling-chip" id="page-sampling-chip">fator —</span>
            <a class="export-link" id="export-csv" href="/api/export?kind=router&amp;format=csv">Export CSV</a>
        </div>
    </header>

    <div id="rd-alerts" class="alerts-strip fade-in-up"></div>

    <section class="stats-grid fade-in-up">
        <div class="stat-card stat-card-highlight">
            <div class="stat-info">
                <span class="stat-label">SNMP In / Out</span>
                <span class="stat-value" id="rd-snmp" style="font-size:1.15rem">—</span>
                <span class="stat-rate" id="rd-util">util —</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">NetFlow × if (buffer)</span>
                <span class="stat-value" id="rd-nf">—</span>
                <span class="stat-rate">estimativa amostrada</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">Speed / Role</span>
                <span class="stat-value" id="rd-speed" style="font-size:1.1rem">—</span>
                <span class="stat-rate" id="rd-role">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">Status</span>
                <span class="stat-value" id="rd-status" style="font-size:1.1rem">—</span>
                <span class="stat-rate" id="rd-host">—</span>
            </div>
        </div>
    </section>

    <section class="fade-in-up" style="animation-delay:0.08s">
        <div class="time-filter" id="rd-time-filter">
            <button type="button" data-h="1" class="tf-btn active">1h</button>
            <button type="button" data-h="6" class="tf-btn">6h</button>
            <button type="button" data-h="24" class="tf-btn">24h</button>
        </div>
        <h3 class="section-title">Histórico desta interface (memória SNMP)</h3>
        <div class="chart-wrap">
            <canvas id="rd-chart" height="240"></canvas>
        </div>
    </section>

    <section class="flow-table-section fade-in-up" style="animation-delay:0.15s">
        <h3 class="section-title">Flows recentes (in_if / out_if)</h3>
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
                <tbody id="rd-flows"></tbody>
            </table>
        </div>
    </section>
    "##.to_string()
}
