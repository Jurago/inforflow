use super::layout::PageMeta;

pub fn meta() -> PageMeta {
    PageMeta {
        title: "Roteador".to_string(),
        active: "router".to_string(),
    }
}

pub fn content() -> String {
    r##"
    <header class="page-header">
        <div>
            <h2 class="page-title">Roteador de Borda</h2>
            <p class="page-subtitle">SNMP — 170.245.127.191:15161 · community infornetV2</p>
        </div>
        <div class="header-actions">
            <div class="live-indicator"><span class="pulse-dot"></span> SNMP ao vivo</div>
        </div>
    </header>

    <section class="stats-grid fade-in-up" style="animation-delay: 0.05s">
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg,#0ea5e9,#38bdf8)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">Equipamento</span>
                <span class="stat-value" id="snmp-sysname" style="font-size:1.1rem">—</span>
                <span class="stat-rate" id="snmp-uptime">—</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg,#f59e0b,#fbbf24)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">CPU</span>
                <span class="stat-value" id="snmp-cpu">—</span>
                <span class="stat-rate">média entidades</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg,#8b5cf6,#a78bfa)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v6c0 1.7 4 3 9 3s9-1.3 9-3V5"/><path d="M3 11v6c0 1.7 4 3 9 3s9-1.3 9-3v-6"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">Memória</span>
                <span class="stat-value" id="snmp-mem">—</span>
                <span class="stat-rate">uso</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg,#10b981,#34d399)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">Uplink In / Out</span>
                <span class="stat-value" id="snmp-uplink" style="font-size:1.1rem">—</span>
                <span class="stat-rate" id="snmp-uplink-util">utilização —</span>
            </div>
        </div>
    </section>

    <section class="traffic-visual fade-in-up" style="animation-delay: 0.15s">
        <h3 class="section-title">Throughput das interfaces (Mbps)</h3>
        <div class="iface-bars" id="iface-bars"></div>
    </section>

    <div class="cards-row fade-in-up" style="animation-delay: 0.25s">
        <section class="card-section">
            <h3 class="section-title">Top entrada (In)</h3>
            <div class="cards-grid" id="snmp-top-in"></div>
        </section>
        <section class="card-section">
            <h3 class="section-title">Top saída (Out)</h3>
            <div class="cards-grid" id="snmp-top-out"></div>
        </section>
    </div>

    <section class="flow-table-section fade-in-up" style="animation-delay: 0.35s">
        <h3 class="section-title">Interfaces do roteador</h3>
        <div class="flow-table-wrapper">
            <table class="flow-table">
                <thead>
                    <tr>
                        <th>If</th>
                        <th>Nome</th>
                        <th>Alias</th>
                        <th>Role</th>
                        <th>Status</th>
                        <th>Speed</th>
                        <th>In Mbps</th>
                        <th>Out Mbps</th>
                        <th>Util%</th>
                    </tr>
                </thead>
                <tbody id="snmp-iface-body"></tbody>
            </table>
        </div>
    </section>
    "##.to_string()
}
