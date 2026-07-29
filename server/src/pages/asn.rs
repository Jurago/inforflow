use super::layout::PageMeta;

pub fn meta() -> PageMeta {
    PageMeta {
        title: "ASN".to_string(),
        active: "asn".to_string(),
    }
}

pub fn content() -> String {
    r##"
    <header class="page-header">
        <div>
            <h2 class="page-title">ASN</h2>
            <p class="page-subtitle">Tráfego real estimado por ASN de destino (NetFlow × amostragem SNMP) — 170.245.127.191</p>
        </div>
        <div class="header-actions">
            <div class="live-indicator"><span class="pulse-dot"></span> Ao vivo</div>
            <a class="export-link" id="export-csv" href="/api/export?kind=asn&format=csv">Export CSV</a>
        </div>
    </header>

    <section class="stats-grid fade-in-up" style="animation-delay: 0.05s">
        <div class="stat-card stat-card-highlight">
            <div class="stat-icon" style="background: linear-gradient(135deg, #0ea5e9, #38bdf8)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M2 12h20M12 2a15 15 0 010 20M12 2a15 15 0 000 20"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">Tráfego ASN (estimado)</span>
                <span class="stat-value" id="asn-total-mbps">—</span>
                <span class="stat-rate" id="asn-total-hint">NetFlow × fator SNMP</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #10b981, #34d399)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">SNMP Uplink</span>
                <span class="stat-value" id="asn-snmp-uplink">—</span>
                <span class="stat-rate">referência total roteador</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #6366f1, #818cf8)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v4M12 19v4M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83M1 12h4M19 12h4"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">ASNs de destino</span>
                <span class="stat-value" id="asn-count">—</span>
                <span class="stat-rate" id="asn-sampling-hint">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #f59e0b, #fbbf24)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 7l10 5 10-5-10-5z"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">Top ASN agora</span>
                <span class="stat-value" id="asn-top-mbps">—</span>
                <span class="stat-rate" id="asn-top-name">—</span>
            </div>
        </div>
    </section>

    <section class="stats-grid fade-in-up" style="animation-delay: 0.08s">
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">IPv4 (ASN dest.)</span>
                <span class="stat-value" id="asn-ipv4-mbps">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">IPv6 (ASN dest.)</span>
                <span class="stat-value" id="asn-ipv6-mbps">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">vs período anterior</span>
                <span class="stat-value" id="asn-compare-delta">—</span>
                <span class="stat-rate" id="asn-compare-hint">ative o filtro abaixo</span>
            </div>
        </div>
    </section>

    <section class="fade-in-up" style="animation-delay: 0.1s">
        <div class="time-filter" id="asn-time-filter">
            <button type="button" data-h="0" class="tf-btn active">Ao vivo</button>
            <button type="button" data-h="1" class="tf-btn">1h</button>
            <button type="button" data-h="6" class="tf-btn">6h</button>
            <button type="button" data-h="24" class="tf-btn">24h</button>
            <button type="button" id="asn-compare-btn" class="tf-btn">Comparar</button>
        </div>
        <h3 class="section-title">Histórico top ASNs</h3>
        <div class="chart-wrap">
            <canvas id="asn-history-chart" height="260"></canvas>
            <div class="chart-tooltip" id="asn-chart-tooltip"></div>
        </div>
        <div class="chart-legend" id="asn-chart-legend"></div>
    </section>

    <section class="fade-in-up" style="animation-delay: 0.12s">
        <h3 class="section-title">Destinos ASN ativos</h3>
        <div class="cdn-nodes" id="asn-nodes"></div>
    </section>

    <div class="cards-row fade-in-up" style="animation-delay: 0.18s">
        <section class="card-section card-section-wide">
            <h3 class="section-title">Tráfego real por ASN de destino</h3>
            <p class="section-hint">Mbps estimado · clique no ASN para ver flows</p>
            <div class="traffic-list" id="asn-traffic-list"></div>
        </section>
        <section class="card-section">
            <h3 class="section-title">Cards por ASN</h3>
            <div class="cards-grid" id="asn-cards"></div>
        </section>
    </div>

    <section class="flow-table-section fade-in-up" style="animation-delay: 0.28s">
        <h3 class="section-title">Detalhe por ASN de destino</h3>
        <div class="flow-table-wrapper">
            <table class="flow-table">
                <thead>
                    <tr>
                        <th>#</th>
                        <th>ASN</th>
                        <th>Nome</th>
                        <th>Mbps est.</th>
                        <th>IPv4</th>
                        <th>IPv6</th>
                        <th>↓ Entrada</th>
                        <th>↑ Saída</th>
                        <th>Bytes</th>
                        <th>Flows</th>
                        <th>%</th>
                        <th>Categoria</th>
                    </tr>
                </thead>
                <tbody id="asn-table-body"></tbody>
            </table>
        </div>
    </section>

    <section class="flow-table-section fade-in-up" style="animation-delay: 0.35s">
        <h3 class="section-title">Flows recentes com ASN de destino</h3>
        <div class="flow-table-wrapper">
            <table class="flow-table">
                <thead>
                    <tr>
                        <th>Hora</th>
                        <th>Origem → Destino</th>
                        <th>ASN dest.</th>
                        <th>IP</th>
                        <th>Categoria</th>
                        <th>Bytes</th>
                        <th>Direção</th>
                    </tr>
                </thead>
                <tbody id="asn-flows-body"></tbody>
            </table>
        </div>
    </section>
    "##.to_string()
}
