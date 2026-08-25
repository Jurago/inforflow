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
            <p class="page-subtitle" id="router-subtitle">SNMP v2c · host dinâmico</p>
        </div>
        <div class="header-actions">
            <div class="live-indicator"><span class="pulse-dot"></span> SNMP ao vivo</div>
            <span class="sampling-chip" id="page-sampling-chip">fator —</span>
            <span class="section-hint" id="router-age" style="margin:0">última coleta —</span>
            <a class="export-link" id="export-csv" href="/api/export?kind=router&amp;format=csv">Export CSV</a>
        </div>
    </header>

    <div id="router-alerts" class="alerts-strip fade-in-up"></div>

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
                <span class="stat-label">CPU / Mem</span>
                <span class="stat-value" id="snmp-cpu" style="font-size:1.15rem">—</span>
                <span class="stat-rate" id="snmp-mem">—</span>
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
        <div class="stat-card">
            <div class="stat-icon" style="background: linear-gradient(135deg,#8b5cf6,#a78bfa)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/></svg>
            </div>
            <div class="stat-info">
                <span class="stat-label">Ifaces up / down</span>
                <span class="stat-value" id="snmp-if-count">—</span>
                <span class="stat-rate" id="snmp-role-hint">cache · IX · BRAS</span>
            </div>
        </div>
    </section>

    <section class="fade-in-up" style="animation-delay:0.1s">
        <div class="time-filter" id="router-time-filter">
            <button type="button" data-h="0" class="tf-btn active">Ao vivo</button>
            <button type="button" data-h="1" class="tf-btn">1h</button>
            <button type="button" data-h="6" class="tf-btn">6h</button>
            <button type="button" data-h="24" class="tf-btn">24h</button>
        </div>
        <h3 class="section-title">Histórico uplink / roles SNMP</h3>
        <div class="chart-wrap">
            <canvas id="router-history-chart" height="240"></canvas>
        </div>
        <div class="chart-legend" id="router-chart-legend"></div>
    </section>

    <section class="traffic-visual fade-in-up" style="animation-delay: 0.15s">
        <h3 class="section-title">Throughput das interfaces (Mbps)</h3>
        <div class="iface-bars" id="iface-bars"></div>
    </section>

    <div class="cards-row fade-in-up" style="animation-delay: 0.2s" id="router-role-groups"></div>

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

    <section class="flow-table-section fade-in-up" style="animation-delay: 0.3s">
        <div class="section-title-row" style="display:flex;flex-wrap:wrap;gap:12px;align-items:center;justify-content:space-between;margin-bottom:12px">
            <h3 class="section-title" style="margin:0">Interfaces do roteador</h3>
            <div style="display:flex;flex-wrap:wrap;gap:8px;align-items:center">
                <select id="router-role-filter" class="login-input" style="max-width:140px;margin:0">
                    <option value="">Todos roles</option>
                    <option value="uplink">Uplink</option>
                    <option value="ix">IX</option>
                    <option value="cache">Cache</option>
                    <option value="bras">BRAS</option>
                    <option value="cgnat">CGNAT</option>
                    <option value="other">Other</option>
                </select>
                <select id="router-state-filter" class="login-input" style="max-width:120px;margin:0">
                    <option value="">Up+Down</option>
                    <option value="up">Up</option>
                    <option value="down">Down</option>
                </select>
                <input type="search" id="router-search" class="login-input" placeholder="Nome / alias…" style="max-width:200px;margin:0" />
            </div>
        </div>
        <p class="section-hint">Util ≥80% destacada · clique na linha para detalhe</p>
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
                        <th>NF role</th>
                    </tr>
                </thead>
                <tbody id="snmp-iface-body"></tbody>
            </table>
        </div>
    </section>
    "##.to_string()
}
