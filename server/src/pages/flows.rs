use super::layout::PageMeta;

pub fn meta() -> PageMeta {
    PageMeta {
        title: "Flows".to_string(),
        active: "flows".to_string(),
    }
}

pub fn content() -> String {
    r##"
    <header class="page-header">
        <div>
            <h2 class="page-title">Explorador de Flows</h2>
            <p class="page-subtitle">Busca por IP, categoria ou texto · drill-down em tempo real</p>
        </div>
    </header>
    <section class="flow-search-bar fade-in-up">
        <input type="text" id="flow-q" class="login-input" placeholder="Busca livre (IP, ASN, serviço…)" />
        <input type="text" id="flow-ip" class="login-input" placeholder="IP" />
        <input type="text" id="flow-asn" class="login-input" placeholder="ASN (ex: AS15169)" />
        <select id="flow-cat" class="login-input">
            <option value="">Todas categorias</option>
            <option value="cdn">CDN</option>
            <option value="streaming">Streaming</option>
            <option value="netflix">Netflix</option>
            <option value="globo">Globo</option>
            <option value="peer">Peer</option>
            <option value="social">Social</option>
            <option value="gaming">Gaming</option>
            <option value="other">Outros</option>
        </select>
        <button type="button" id="flow-search" class="tf-btn">Buscar</button>
        <button type="button" id="flow-export" class="tf-btn">Export CSV</button>
        <span id="flow-filter-hint" class="section-hint" style="margin:0"></span>
    </section>
    <section class="flow-table-section fade-in-up">
        <table class="flow-table">
            <thead><tr>
                <th>Hora</th><th>Cat.</th><th>Origem → Destino</th><th>Serviço</th><th>Dst ASN</th>
                <th>Peer ASN</th><th>Interface</th><th>Next-hop</th><th>Portas</th><th>Bytes</th><th>Dir</th><th>IP</th>
            </tr></thead>
            <tbody id="flows-body"></tbody>
        </table>
    </section>
    "##.to_string()
}
