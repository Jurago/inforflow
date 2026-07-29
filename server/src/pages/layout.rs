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
    <link rel="stylesheet" href="/static/css/inforflow.css">
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link href="https://fonts.googleapis.com/css2?family=DM+Sans:wght@400;500;600;700&family=IBM+Plex+Mono:wght@400;500&display=swap" rel="stylesheet">
</head>
<body>
    <div class="app-shell">
        <aside class="sidebar" id="sidebar">
            <div class="sidebar-brand">
                <div class="brand-icon">
                    <svg viewBox="0 0 32 32" fill="none"><circle cx="16" cy="16" r="14" stroke="currentColor" stroke-width="2"/><path d="M8 16h16M16 8v16" stroke="currentColor" stroke-width="2"/><circle cx="16" cy="16" r="4" fill="currentColor"/></svg>
                </div>
                <div>
                    <h1>Inforflow</h1>
                    <span class="brand-sub">ISP Flow + SNMP</span>
                </div>
            </div>
            <nav class="sidebar-nav">
                <a href="/" class="nav-item {dash_active}">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>
                    Dashboard
                </a>
                <a href="/graphs" class="nav-item {graphs_active}">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 20V10M12 20V4M6 20v-6"/></svg>
                    Gráficos
                </a>
                <a href="/router" class="nav-item {router_active}">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/></svg>
                    Roteador
                </a>
                <a href="/cdns" class="nav-item {cdn_active}">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/></svg>
                    CDNs
                </a>
                <a href="/streaming" class="nav-item {stream_active}">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>
                    Streaming
                </a>
                <a href="/peers" class="nav-item {peer_active}">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v4M12 19v4M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83M1 12h4M19 12h4M4.22 19.78l2.83-2.83M16.95 7.05l2.83-2.83"/></svg>
                    Peers
                </a>
                <a href="/asn" class="nav-item {asn_active}">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M2 12h20M12 2a15 15 0 010 20M12 2a15 15 0 000 20"/></svg>
                    ASN
                </a>
                <a href="/cache" class="nav-item {cache_active}">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 4h16v6H4zM4 14h16v6H4z"/></svg>
                    Cache
                </a>
                <a href="/flows" class="nav-item {flows_active}">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 6h16M4 12h16M4 18h10"/></svg>
                    Flows
                </a>
                <a href="/sampling" class="nav-item {samp_active}">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/></svg>
                    Amostragem
                </a>
            </nav>
            <div class="sidebar-footer">
                <div class="source-ip-badge">
                    <span class="pulse-dot"></span>
                    <span>170.245.127.191</span>
                </div>
                <button type="button" class="logout-btn" onclick="window.Inforflow && window.Inforflow.logout()">Sair</button>
            </div>
        </aside>
        <main class="main-content page-enter">
            {content}
        </main>
    </div>
    <div id="flow-particles" class="flow-particles"></div>
    <script src="/static/js/inforflow.js"></script>
    <script src="/static/js/{page_script}"></script>
</body>
</html>"##,
            title = meta.title,
            dash_active = if meta.active == "dashboard" { "active" } else { "" },
            graphs_active = if meta.active == "graphs" { "active" } else { "" },
            router_active = if meta.active == "router" { "active" } else { "" },
            cdn_active = if meta.active == "cdns" { "active" } else { "" },
            stream_active = if meta.active == "streaming" { "active" } else { "" },
            peer_active = if meta.active == "peers" { "active" } else { "" },
            asn_active = if meta.active == "asn" { "active" } else { "" },
            cache_active = if meta.active == "cache" { "active" } else { "" },
            flows_active = if meta.active == "flows" { "active" } else { "" },
            samp_active = if meta.active == "sampling" { "active" } else { "" },
            content = content,
            page_script = match meta.active.as_str() {
                "cdns" => "cdns.js",
                "streaming" => "streaming.js",
                "peers" => "peers.js",
                "asn" => "asn.js",
                "router" => "router.js",
                "graphs" => "graphs.js",
                "sampling" => "sampling.js",
                "cache" => "cache.js",
                "flows" => "flows.js",
                _ => "dashboard.js",
            },
        )
    }
}
