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
            <p class="page-subtitle">Sessões BGP (SNMP) + tráfego NetFlow × amostragem por ASN de peer</p>
        </div>
        <div class="header-actions">
            <div class="live-indicator"><span class="pulse-dot"></span> Ao vivo</div>
            <span class="sampling-chip" id="page-sampling-chip" title="Fator de amostragem">fator —</span>
            <a class="export-link" id="export-csv" href="/api/export?kind=peers&format=csv">Export CSV</a>
        </div>
    </header>

    <div id="peers-alerts" class="alerts-strip fade-in-up"></div>

    <section class="stats-grid fade-in-up" style="animation-delay: 0.08s">
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #6366f1, #818cf8)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v4M12 19v4"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">BGP Established</span>
                <span class="stat-value" id="peer-established">—</span>
                <span class="stat-rate" id="peer-down-hint">—</span>
            </div>
        </div>
        <div class="stat-card stat-card-highlight">
            <div class="stat-icon" style="background: linear-gradient(135deg, #10b981, #34d399)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">Tráfego Peer (Mbps est. · 10s)</span>
                <span class="stat-value" id="peer-traffic">—</span>
                <span class="stat-rate" id="peer-traffic-hint">vs SNMP IX/transit</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #f59e0b, #fbbf24)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 7l10 5 10-5-10-5z"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label" id="peer-ix-label">IX</span>
                <span class="stat-value" id="peer-ixbr">—</span>
                <span class="stat-rate" id="peer-ix-hint">Mbps estimado</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #06b6d4, #22d3ee)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">Sessões BGP</span>
                <span class="stat-value" id="peer-total">—</span>
                <span class="stat-rate" id="bgp-local-hint">AS local</span>
            </div>
        </div>
    </section>

    <section class="traffic-visual fade-in-up" style="animation-delay: 0.12s">
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

    <section class="fade-in-up" style="animation-delay: 0.15s">
        <div class="time-filter" id="peer-time-filter">
            <button type="button" data-h="0" class="tf-btn active">Ao vivo</button>
            <button type="button" data-h="1" class="tf-btn">1h</button>
            <button type="button" data-h="6" class="tf-btn">6h</button>
            <button type="button" data-h="24" class="tf-btn">24h</button>
        </div>
        <h3 class="section-title">Histórico top peers (ASN)</h3>
        <div class="chart-wrap">
            <canvas id="peer-history-chart" height="240"></canvas>
        </div>
        <div class="chart-legend" id="peer-chart-legend"></div>
    </section>

    <div class="cards-row fade-in-up" style="animation-delay: 0.2s">
        <section class="card-section" style="flex: 1.2">
            <h3 class="section-title">Tráfego por Peer BGP (ASN)</h3>
            <p class="section-hint">Mbps estimado (10s) · clique para detalhe</p>
            <div class="cards-grid" id="peer-breakdown"></div>
        </section>
        <section class="card-section">
            <h3 class="section-title">Interfaces SNMP (IX / transit)</h3>
            <p class="section-hint">Correlação com NetFlow × amostragem</p>
            <div class="cards-grid" id="peer-snmp-ifaces"></div>
        </section>
    </div>

    <section class="flow-table-section fade-in-up" style="animation-delay: 0.28s">
        <div class="section-title-row" style="display:flex;flex-wrap:wrap;gap:12px;align-items:center;justify-content:space-between;margin-bottom:12px">
            <h3 class="section-title" style="margin:0">Sessões BGP</h3>
            <div style="display:flex;flex-wrap:wrap;gap:8px;align-items:center">
                <button type="button" class="tf-btn active" id="peer-view-session" data-view="session">Por sessão</button>
                <button type="button" class="tf-btn" id="peer-view-asn" data-view="asn">Por ASN</button>
                <select id="peer-role-filter" class="login-input" style="max-width:140px;margin:0">
                    <option value="">Todos papéis</option>
                    <option value="ix">IX</option>
                    <option value="content">Conteúdo</option>
                    <option value="transit">Trânsito</option>
                    <option value="regional">Regional</option>
                    <option value="local">Local</option>
                    <option value="private">Privado</option>
                </select>
                <select id="peer-state-filter" class="login-input" style="max-width:140px;margin:0">
                    <option value="">Todos estados</option>
                    <option value="up">Established</option>
                    <option value="down">Down</option>
                </select>
                <input type="search" id="peer-search" class="login-input" placeholder="Filtrar ASN, nome, IP…" style="max-width:220px;margin:0" />
            </div>
        </div>
        <p class="section-hint">Mbps est. = NetFlow × amostragem · Uptime / flaps inferidos pelo poller</p>
        <div class="flow-table-wrapper">
            <table class="flow-table">
                <thead>
                    <tr>
                        <th>Peer / ASN</th>
                        <th>Nome</th>
                        <th>Estado</th>
                        <th>Papel</th>
                        <th>Mbps est.</th>
                        <th>Bytes</th>
                        <th>Uptime</th>
                        <th>Flaps</th>
                        <th>Updates in/out</th>
                    </tr>
                </thead>
                <tbody id="bgp-table-body"></tbody>
            </table>
        </div>
    </section>

    <section class="flow-table-section fade-in-up" style="animation-delay: 0.35s">
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
