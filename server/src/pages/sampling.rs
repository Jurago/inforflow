use super::layout::PageMeta;

pub fn meta() -> PageMeta {
    PageMeta {
        title: "Amostragem".to_string(),
        active: "sampling".to_string(),
    }
}

pub fn content() -> String {
    r##"
    <header class="page-header">
        <div>
            <h2 class="page-title">Amostragem NetFlow</h2>
            <p class="page-subtitle">Como o fator é obtido e como as categorias são escaladas ao SNMP</p>
        </div>
        <div class="header-actions">
            <div class="live-indicator"><span class="pulse-dot"></span> Ao vivo</div>
        </div>
    </header>

    <section class="stats-grid fade-in-up">
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">Fator efetivo</span>
                <span class="stat-value" id="samp-effective">—</span>
                <span class="stat-rate" id="samp-mode">modo</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">Nativo (options)</span>
                <span class="stat-value" id="samp-native">—</span>
                <span class="stat-rate">NetFlow sampler</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">Estimado SNMP/NF</span>
                <span class="stat-value" id="samp-estimated">—</span>
                <span class="stat-rate">razão uplink</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-info">
                <span class="stat-label">Mbps escalado</span>
                <span class="stat-value" id="samp-scaled" style="font-size:1.1rem">—</span>
                <span class="stat-rate" id="samp-compare">vs SNMP</span>
            </div>
        </div>
    </section>

    <section class="card-section fade-in-up" style="animation-delay:0.1s">
        <h3 class="section-title">Como funciona</h3>
        <div class="sampling-explain">
            <p><strong>1. SNMP</strong> mede octets reais nas interfaces (autoridade de Gbps).</p>
            <p><strong>2. NetFlow</strong> chega amostrado — por isso o Mbps “cru” é muito menor.</p>
            <p><strong>3. Fator</strong> preferido nesta ordem: config fixa → options template NetFlow (nativo) → razão SNMP÷NetFlow (auto).</p>
            <p><strong>4. Categorias</strong> no dashboard usam <em>Mbps estimado</em> = taxa NetFlow da categoria × fator, mantendo a proporção do NetFlow.</p>
            <p id="samp-source" class="stat-rate">fonte atual: —</p>
        </div>
    </section>

    <section class="card-section fade-in-up" style="animation-delay:0.2s">
        <h3 class="section-title">Categorias (in / out estimado)</h3>
        <div class="cards-grid" id="samp-cat-inout"></div>
    </section>

    <section class="card-section fade-in-up" style="animation-delay:0.25s">
        <h3 class="section-title">Tráfego por papel de interface (ifIndex→SNMP)</h3>
        <div class="cards-grid" id="samp-iface-roles"></div>
    </section>
    "##.to_string()
}
