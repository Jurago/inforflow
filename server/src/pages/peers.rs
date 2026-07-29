use super::layout::PageMeta;

pub fn meta() -> PageMeta {
    PageMeta {
        title: "Peers".to_string(),
        active: "peers".to_string(),
    }
}

pub fn content() -> String {
    r##"
    <header class="page-header">
        <div>
            <h2 class="page-title">Peers &amp; BGP</h2>
            <p class="page-subtitle">Sessões BGP via SNMP (BGP4-MIB) + tráfego NetFlow por ASN — 170.245.127.191</p>
        </div>
        <div class="header-actions">
            <div class="live-indicator"><span class="pulse-dot"></span> Ao vivo</div>
        </div>
    </header>

    <section class="stats-grid fade-in-up" style="animation-delay: 0.1s">
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #6366f1, #818cf8)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v4M12 19v4"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">BGP Established</span>
                <span class="stat-value" id="peer-established">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #10b981, #34d399)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">Tráfego Peer / ASN</span>
                <span class="stat-value" id="peer-traffic">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #f59e0b, #fbbf24)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 7l10 5 10-5-10-5z"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">IX.br (AS26162)</span>
                <span class="stat-value" id="peer-ixbr">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #06b6d4, #22d3ee)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">Sessões BGP</span>
                <span class="stat-value" id="peer-total">—</span>
            </div>
        </div>
    </section>

    <section class="traffic-visual fade-in-up" style="animation-delay: 0.15s">
        <div class="peer-topology" id="peer-topology">
            <div class="topology-center">
                <div class="center-node">
                    <span id="bgp-local-as">AS—</span>
                    <span class="center-sub">BGP local</span>
                </div>
                <div class="orbit-ring ring-1"></div>
                <div class="orbit-ring ring-2"></div>
                <div class="orbit-ring ring-3"></div>
            </div>
            <div class="peer-orbit" id="peer-orbit"></div>
        </div>
    </section>

    <div class="cards-row fade-in-up" style="animation-delay: 0.25s">
        <section class="card-section" style="flex: 1.2">
            <h3 class="section-title">Tráfego por Peer BGP (ASN)</h3>
            <div class="cards-grid" id="peer-breakdown"></div>
        </section>
        <section class="card-section">
            <h3 class="section-title">Top origens / destinos Peer</h3>
            <div class="cards-grid" id="peer-destinations"></div>
        </section>
    </div>

    <section class="flow-table-section fade-in-up" style="animation-delay: 0.35s">
        <h3 class="section-title">Sessões BGP (SNMP BGP4-MIB)</h3>
        <div class="flow-table-wrapper">
            <table class="flow-table">
                <thead>
                    <tr>
                        <th>Peer IP</th>
                        <th>ASN</th>
                        <th>Nome</th>
                        <th>Estado</th>
                        <th>Papel</th>
                        <th>Mbps</th>
                        <th>Bytes</th>
                        <th>Updates in/out</th>
                    </tr>
                </thead>
                <tbody id="bgp-table-body"></tbody>
            </table>
        </div>
    </section>

    <section class="flow-table-section fade-in-up" style="animation-delay: 0.45s">
        <h3 class="section-title">Flows associados a peers BGP</h3>
        <div class="flow-table-wrapper">
            <table class="flow-table">
                <thead>
                    <tr>
                        <th>Peer</th>
                        <th>ASN</th>
                        <th>Origem → Destino</th>
                        <th>Categoria</th>
                        <th>Bytes</th>
                        <th>Direção</th>
                    </tr>
                </thead>
                <tbody id="peer-table-body"></tbody>
            </table>
        </div>
    </section>
    "##.to_string()
}
