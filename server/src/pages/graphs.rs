use super::layout::PageMeta;

pub fn meta() -> PageMeta {
    PageMeta {
        title: "Gráficos".to_string(),
        active: "graphs".to_string(),
    }
}

pub fn content() -> String {
    r##"
    <header class="page-header">
        <div>
            <h2 class="page-title">Gráficos de Consumo</h2>
            <p class="page-subtitle">Séries temporais · Mbps estimado (NetFlow × SNMP) · zoom arrastando no gráfico</p>
        </div>
        <div class="header-actions">
            <span class="sampling-chip" id="page-sampling-chip">fator —</span>
            <div class="live-indicator"><span class="pulse-dot"></span> Ao vivo</div>
        </div>
    </header>

    <div class="page-toolbar fade-in-up">
        <div class="chart-mode-toggle" id="chart-mode-toggle">
            <button type="button" data-mode="scaled" class="tf-btn active">Mbps estimado</button>
            <button type="button" data-mode="raw" class="tf-btn">NetFlow bruto</button>
        </div>
        <div class="chart-mode-toggle" id="series-view-toggle">
            <button type="button" data-view="all" class="tf-btn active">NF + SNMP</button>
            <button type="button" data-view="snmp" class="tf-btn">Só SNMP</button>
        </div>
        <div class="time-filter" id="time-filter">
            <button type="button" data-h="0" class="tf-btn active">Ao vivo</button>
            <button type="button" data-h="1" class="tf-btn">1h</button>
            <button type="button" data-h="6" class="tf-btn">6h</button>
            <button type="button" data-h="24" class="tf-btn">24h</button>
            <button type="button" data-h="72" class="tf-btn">72h</button>
            <button type="button" data-h="168" class="tf-btn">7d</button>
        </div>
        <div class="chart-mode-toggle" id="chart-compare-toggle">
            <button type="button" data-compare="0" class="tf-btn active">Período único</button>
            <button type="button" data-compare="1" class="tf-btn">vs anterior</button>
        </div>
        <button type="button" class="tf-btn" id="btn-export-csv">CSV</button>
        <button type="button" class="tf-btn" id="btn-export-png">PNG</button>
        <button type="button" class="tf-btn" id="btn-zoom-reset" title="Reset zoom">Reset zoom</button>
    </div>

    <div id="graphs-alerts" class="alerts-strip fade-in-up"></div>

    <section class="page-toolbar fade-in-up" style="animation-delay:0.03s">
        <label class="section-hint" style="margin:0">De <input type="datetime-local" id="range-from" class="login-input" style="max-width:200px;margin:0;display:inline-block" /></label>
        <label class="section-hint" style="margin:0">Até <input type="datetime-local" id="range-to" class="login-input" style="max-width:200px;margin:0;display:inline-block" /></label>
        <button type="button" class="tf-btn" id="btn-range-apply">Aplicar intervalo</button>
        <span class="section-hint" id="graphs-links" style="margin:0">
            <a href="/cdns">CDNs</a> · <a href="/streaming">Streaming</a> · <a href="/asn">ASN</a> · <a href="/peers">Peers</a>
        </span>
    </section>

    <section class="stats-grid fade-in-up" style="animation-delay: 0.05s">
        <div class="stat-card stat-card-highlight">
            <div class="stat-icon" style="background: linear-gradient(135deg, #3b82f6, #60a5fa)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">NetFlow estimado</span>
                <span class="stat-value" id="g-nf-mbps">—</span>
                <span class="stat-rate" id="g-classified">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #10b981, #34d399)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 19V5M5 12l7-7 7 7"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">SNMP Uplink In / Out</span>
                <span class="stat-value" id="g-snmp-in" style="font-size:1.15rem">—</span>
                <span class="stat-rate" id="g-snmp-out">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #f59e0b, #fbbf24)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 7l10 5 10-5-10-5z"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">Gap NF × SNMP</span>
                <span class="stat-value" id="g-gap">—</span>
                <span class="stat-rate" id="g-gap-pct">|NF − SNMP médio|</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #8b5cf6, #a78bfa)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">IPv4 / IPv6 · fator</span>
                <span class="stat-value" id="g-v4v6" style="font-size:1.1rem">—</span>
                <span class="stat-rate" id="g-sampling">—</span>
            </div>
        </div>
    </section>

    <section class="chart-panel chart-panel-interactive fade-in-up" style="animation-delay: 0.1s">
        <div class="chart-panel-head">
            <div>
                <h3 class="section-title">Throughput total</h3>
                <p class="chart-panel-desc">NetFlow · SNMP · IPv4/IPv6 · arraste para zoom</p>
            </div>
            <div class="chart-hint">Passe o mouse para ver valores</div>
        </div>
        <div class="chart-wrap" id="wrap-total">
            <canvas id="chart-total" height="300"></canvas>
            <div class="chart-tooltip" id="tooltip-total"></div>
        </div>
        <div class="chart-legend chart-legend-interactive" id="legend-total"></div>
    </section>

    <section class="chart-panel chart-panel-interactive fade-in-up" style="animation-delay: 0.15s">
        <div class="chart-panel-head">
            <div>
                <h3 class="section-title">Consumo por categoria</h3>
                <p class="chart-panel-desc">Clique na legenda para ocultar · deep-link ?cat=cdn</p>
            </div>
            <div class="chart-type-toggle" id="cat-chart-type">
                <button type="button" data-type="line" class="tf-btn active">Linhas</button>
                <button type="button" data-type="stacked" class="tf-btn">Empilhado</button>
            </div>
        </div>
        <div class="chart-wrap" id="wrap-categories">
            <canvas id="chart-categories" height="340"></canvas>
            <div class="chart-tooltip" id="tooltip-categories"></div>
        </div>
        <div class="chart-legend chart-legend-interactive" id="legend-categories"></div>
    </section>

    <section class="chart-panel chart-panel-interactive fade-in-up" style="animation-delay: 0.18s">
        <div class="chart-panel-head">
            <div>
                <h3 class="section-title">Top ASNs (histórico)</h3>
                <p class="chart-panel-desc">Mbps estimado · deep-link ?asn=AS13335</p>
            </div>
            <div class="chart-mode-toggle" id="asn-role-toggle">
                <button type="button" data-asnrole="dest" class="tf-btn active">Destino</button>
                <button type="button" data-asnrole="peer" class="tf-btn">Peer</button>
            </div>
        </div>
        <div class="chart-wrap" id="wrap-asn">
            <canvas id="chart-asn" height="280"></canvas>
            <div class="chart-tooltip" id="tooltip-asn"></div>
        </div>
        <div class="chart-legend chart-legend-interactive" id="legend-asn"></div>
    </section>

    <section class="chart-panel chart-panel-interactive fade-in-up" style="animation-delay: 0.2s">
        <div class="chart-panel-head">
            <div>
                <h3 class="section-title">Top CDN / Streaming</h3>
                <p class="chart-panel-desc">Serviços no histórico · atalhos para páginas dedicadas</p>
            </div>
            <div class="chart-mode-toggle" id="svc-toggle">
                <button type="button" data-svc="cdn" class="tf-btn active">CDN</button>
                <button type="button" data-svc="streaming" class="tf-btn">Streaming</button>
            </div>
        </div>
        <div class="chart-wrap" id="wrap-svc">
            <canvas id="chart-svc" height="260"></canvas>
            <div class="chart-tooltip" id="tooltip-svc"></div>
        </div>
        <div class="chart-legend chart-legend-interactive" id="legend-svc"></div>
    </section>

    <div class="graphs-bottom-row fade-in-up" style="animation-delay: 0.22s">
        <section class="chart-panel chart-panel-compact">
            <h3 class="section-title">Distribuição atual</h3>
            <div class="chart-wrap chart-wrap-pie">
                <canvas id="chart-pie" height="280"></canvas>
                <div class="chart-tooltip" id="tooltip-pie"></div>
            </div>
        </section>
        <section class="card-section">
            <h3 class="section-title">Ranking por Mbps estimado</h3>
            <div class="category-bars" id="graph-bars"></div>
        </section>
    </div>
    "##.to_string()
}
