use super::layout::PageMeta;

pub fn meta() -> PageMeta {
    PageMeta {
        title: "Streaming".to_string(),
        active: "streaming".to_string(),
    }
}

pub fn content() -> String {
    r##"
    <header class="page-header">
        <div>
            <h2 class="page-title">Streaming</h2>
            <p class="page-subtitle">Taxa real estimada (NetFlow × amostragem SNMP) · interfaces cache via SNMP — 170.245.127.191</p>
        </div>
        <div class="header-actions">
            <div class="live-indicator"><span class="pulse-dot"></span> Ao vivo</div>
        </div>
    </header>

    <section class="stats-grid fade-in-up" style="animation-delay: 0.05s">
        <div class="stat-card stat-card-highlight">
            <div class="stat-icon" style="background: linear-gradient(135deg, #8b5cf6, #a78bfa)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">Streaming total (estimado)</span>
                <span class="stat-value" id="stream-total-mbps">—</span>
                <span class="stat-rate" id="stream-total-hint">NetFlow × fator SNMP</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #10b981, #34d399)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 7l10 5 10-5-10-5z"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">SNMP Uplink</span>
                <span class="stat-value" id="stream-snmp-uplink">—</span>
                <span class="stat-rate">entrada / saída</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #f59e0b, #fbbf24)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">SNMP Caches</span>
                <span class="stat-value" id="stream-snmp-cache">—</span>
                <span class="stat-rate" id="stream-snmp-cache-label">interfaces cache</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg, #06b6d4, #22d3ee)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 3l14 9-14 9V3z"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">Serviços detectados</span>
                <span class="stat-value" id="stream-count">—</span>
                <span class="stat-rate" id="stream-sampling-hint">—</span>
            </div>
        </div>
    </section>

    <section class="streaming-hero fade-in-up" style="animation-delay: 0.1s">
        <h3 class="section-title">Top serviços (Mbps estimado)</h3>
        <div class="hero-cards" id="stream-hero-cards"></div>
    </section>

    <div class="cards-row fade-in-up" style="animation-delay: 0.15s">
        <section class="card-section card-section-wide">
            <h3 class="section-title">Tráfego por serviço</h3>
            <p class="section-hint">Lista dinâmica — inclui todos os streamings detectados no NetFlow (YouTube, Netflix, RTMP, HBO Max, etc.)</p>
            <div class="traffic-list" id="streaming-traffic-list"></div>
        </section>
        <section class="card-section">
            <h3 class="section-title">Entrada / Saída (categoria)</h3>
            <div class="traffic-io-summary" id="stream-io-summary"></div>
            <h3 class="section-title" style="margin-top:24px">Interfaces SNMP (cache)</h3>
            <div class="cards-grid" id="streaming-snmp-ifaces"></div>
        </section>
    </div>

    <section class="traffic-visual fade-in-up" style="animation-delay: 0.25s">
        <div class="streaming-flow" id="streaming-flow">
            <div class="stream-source">
                <div class="source-ring"></div>
                <span>170.245.127.191</span>
            </div>
            <div class="stream-rays" id="stream-rays"></div>
            <div class="stream-targets" id="stream-targets"></div>
        </div>
    </section>

    <section class="fade-in-up" style="animation-delay: 0.35s">
        <h3 class="section-title">Flows recentes de streaming</h3>
        <div class="streaming-timeline" id="streaming-timeline"></div>
    </section>
    "##.to_string()
}
