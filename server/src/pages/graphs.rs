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
            <p class="page-subtitle">Séries temporais com Mbps estimado (NetFlow × SNMP) · passe o mouse sobre as linhas</p>
        </div>
        <div class="header-actions">
            <div class="chart-mode-toggle" id="chart-mode-toggle">
                <button type="button" data-mode="scaled" class="tf-btn active">Mbps estimado</button>
                <button type="button" data-mode="raw" class="tf-btn">NetFlow bruto</button>
            </div>
            <div class="time-filter" id="time-filter">
                <button type="button" data-h="0" class="tf-btn active">Ao vivo</button>
                <button type="button" data-h="1" class="tf-btn">1h</button>
                <button type="button" data-h="6" class="tf-btn">6h</button>
                <button type="button" data-h="24" class="tf-btn">24h</button>
            </div>
            <div class="chart-mode-toggle" id="chart-compare-toggle">
                <button type="button" data-compare="0" class="tf-btn active">Período único</button>
                <button type="button" data-compare="1" class="tf-btn">vs período anterior</button>
            </div>
            <div class="live-indicator"><span class="pulse-dot"></span> Ao vivo</div>
        </div>
    </header>

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
                <span class="stat-label">SNMP Uplink In</span>
                <span class="stat-value" id="g-snmp-in">—</span>
                <span class="stat-rate">referência borda</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #06b6d4, #22d3ee)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12l7 7 7-7"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">SNMP Uplink Out</span>
                <span class="stat-value" id="g-snmp-out">—</span>
                <span class="stat-rate">referência borda</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #8b5cf6, #a78bfa)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">Fator amostragem</span>
                <span class="stat-value" id="g-sampling">—</span>
                <span class="stat-rate" id="g-cat-count">—</span>
            </div>
        </div>
    </section>

    <section class="chart-panel chart-panel-interactive fade-in-up" style="animation-delay: 0.1s">
        <div class="chart-panel-head">
            <div>
                <h3 class="section-title">Throughput total</h3>
                <p class="chart-panel-desc">NetFlow estimado vs SNMP uplink (in/out)</p>
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
                <p class="chart-panel-desc">Top categorias em Mbps · clique na legenda para ocultar</p>
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

    <div class="graphs-bottom-row fade-in-up" style="animation-delay: 0.2s">
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
        <section class="chart-panel chart-panel-compact">
            <h3 class="section-title">Comparativo agora</h3>
            <div class="compare-bars" id="compare-bars"></div>
            <p class="chart-panel-desc" style="margin-top:12px">NetFlow estimado vs SNMP (média in/out)</p>
        </section>
    </div>
    "##.to_string()
}
