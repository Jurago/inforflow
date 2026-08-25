pub struct PageMeta {
    pub title: String,
    pub active: String,
}

pub struct Layout;

impl Layout {
    pub fn render(meta: &PageMeta, content: &str) -> String {
        format!(
            r##"<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{title} — Inforflow</title>
    <link rel="stylesheet" href="/static/css/inforflow.css?v=20260811b">
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link href="https://fonts.googleapis.com/css2?family=DM+Sans:wght@400;500;600;700&family=IBM+Plex+Mono:wght@400;500&display=swap" rel="stylesheet">
</head>
<body>
    <div class="app-shell">
        <header class="topbar" id="sidebar">
            <a href="/" class="topbar-brand">
                <div class="brand-icon">
                    <svg viewBox="0 0 32 32" fill="none"><circle cx="16" cy="16" r="14" stroke="currentColor" stroke-width="2"/><path d="M8 16h16M16 8v16" stroke="currentColor" stroke-width="2"/><circle cx="16" cy="16" r="4" fill="currentColor"/></svg>
                </div>
                <div class="brand-text">
                    <h1>Inforflow</h1>
                    <span class="brand-sub">ISP Flow + SNMP</span>
                </div>
            </a>

            <button type="button" class="nav-toggle" id="nav-toggle" aria-label="Abrir menu" aria-expanded="false" aria-controls="topbar-nav">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 6h16M4 12h16M4 18h16"/></svg>
            </button>

            <nav class="topbar-nav" id="topbar-nav">
                <div class="nav-group" data-label="Visão">
                    <a href="/" class="nav-item {dash_active}" title="Dashboard">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>
                        <span>Dashboard</span>
                    </a>
                    <a href="/graphs" class="nav-item {graphs_active}" title="Gráficos">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 20V10M12 20V4M6 20v-6"/></svg>
                        <span>Gráficos</span>
                    </a>
                </div>
                <div class="nav-sep" aria-hidden="true"></div>
                <div class="nav-group" data-label="Infra">
                    <a href="/router" class="nav-item {router_active}" title="Roteador">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/></svg>
                        <span>Roteador</span>
                    </a>
                    <a href="/peers" class="nav-item {peer_active}" title="Peers BGP">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v4M12 19v4M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83M1 12h4M19 12h4M4.22 19.78l2.83-2.83M16.95 7.05l2.83-2.83"/></svg>
                        <span>Peers</span>
                    </a>
                    <a href="/asn" class="nav-item {asn_active}" title="ASN">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M2 12h20M12 2a15 15 0 010 20M12 2a15 15 0 000 20"/></svg>
                        <span>ASN</span>
                    </a>
                </div>
                <div class="nav-sep" aria-hidden="true"></div>
                <div class="nav-group" data-label="Tráfego">
                    <a href="/cdns" class="nav-item {cdn_active}" title="CDNs">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/></svg>
                        <span>CDNs</span>
                    </a>
                    <a href="/streaming" class="nav-item {stream_active}" title="Streaming">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>
                        <span>Streaming</span>
                    </a>
                    <a href="/flows" class="nav-item {flows_active}" title="Flows">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 6h16M4 12h16M4 18h10"/></svg>
                        <span>Flows</span>
                    </a>
                </div>
                <div class="nav-sep" aria-hidden="true"></div>
                <div class="nav-group" data-label="Sistema">
                    <a href="/cache" class="nav-item {cache_active}" title="Cache">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 4h16v6H4zM4 14h16v6H4z"/></svg>
                        <span>Cache</span>
                    </a>
                    <a href="/sampling" class="nav-item {samp_active}" title="Amostragem">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/></svg>
                        <span>Amostragem</span>
                    </a>
                </div>
            </nav>

            <div class="topbar-meta">
                <div class="source-ip-badge">
                    <span class="pulse-dot"></span>
                    <span data-source-ip>—</span>
                </div>
                <div class="sampling-chip" id="sidebar-sampling-chip" title="Amostragem NetFlow×SNMP">fator —</div>
                <button type="button" class="logout-btn" onclick="window.Inforflow && window.Inforflow.logout()">Sair</button>
            </div>
        </header>
        <main class="main-content page-enter">
            <div class="page-shell">
            {content}
            </div>
        </main>
    </div>
    <div id="flow-particles" class="flow-particles"></div>
    <script src="/static/js/inforflow.js?v=20260811b"></script>
    <script src="/static/js/{page_script}?v=20260811b"></script>
</body>
</html>"##,
            title = meta.title,
            dash_active = if meta.active == "dashboard" { "active" } else { "" },
            graphs_active = if meta.active == "graphs" { "active" } else { "" },
            router_active = if meta.active == "router" || meta.active == "routerdetail" { "active" } else { "" },
            cdn_active = if meta.active == "cdns" || meta.active == "cdndetail" { "active" } else { "" },
            stream_active = if meta.active == "streaming" || meta.active == "streamingdetail" { "active" } else { "" },
            peer_active = if meta.active == "peers" || meta.active == "peersdetail" { "active" } else { "" },
            asn_active = if meta.active == "asn" || meta.active == "asndetail" { "active" } else { "" },
            cache_active = if meta.active == "cache" { "active" } else { "" },
            flows_active = if meta.active == "flows" { "active" } else { "" },
            samp_active = if meta.active == "sampling" { "active" } else { "" },
            content = content,
            page_script = match meta.active.as_str() {
                "cdns" => "cdns.js",
                "cdndetail" => "cdndetail.js",
                "streaming" => "streaming.js",
                "streamingdetail" => "streamingdetail.js",
                "peers" => "peers.js",
                "peersdetail" => "peersdetail.js",
                "asn" => "asn.js",
                "asndetail" => "asndetail.js",
                "router" => "router.js",
                "routerdetail" => "routerdetail.js",
                "graphs" => "graphs.js",
                "sampling" => "sampling.js",
                "cache" => "cache.js",
                "flows" => "flows.js",
                _ => "dashboard.js",
            },
        )
    }
}
