pub mod dashboard;
pub mod cdns;
pub mod peers;
pub mod asn;
pub mod streaming;
pub mod router;
pub mod graphs;
pub mod sampling;
pub mod cache;
pub mod flows;
pub mod login;
pub mod layout;

use layout::PageMeta;

pub fn render_page(meta: &PageMeta, content: &str) -> String {
    layout::Layout::render(meta, content)
}

pub fn render_dashboard() -> String {
    render_page(&dashboard::meta(), &dashboard::content())
}

pub fn render_cdns() -> String {
    render_page(&cdns::meta(), &cdns::content())
}

pub fn render_peers() -> String {
    render_page(&peers::meta(), &peers::content())
}

pub fn render_asn() -> String {
    render_page(&asn::meta(), &asn::content())
}

pub fn render_streaming() -> String {
    render_page(&streaming::meta(), &streaming::content())
}

pub fn render_router() -> String {
    render_page(&router::meta(), &router::content())
}

pub fn render_graphs() -> String {
    render_page(&graphs::meta(), &graphs::content())
}

pub fn render_sampling() -> String {
    render_page(&sampling::meta(), &sampling::content())
}

pub fn render_cache() -> String {
    render_page(&cache::meta(), &cache::content())
}

pub fn render_flows() -> String {
    render_page(&flows::meta(), &flows::content())
}
