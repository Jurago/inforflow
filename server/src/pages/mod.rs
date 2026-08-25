pub mod dashboard;
pub mod cdns;
pub mod cdndetail;
pub mod peers;
pub mod peersdetail;
pub mod asn;
pub mod asndetail;
pub mod streaming;
pub mod streamingdetail;
pub mod router;
pub mod routerdetail;
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

pub fn render_cdn_detail() -> String {
    render_page(&cdndetail::meta(), &cdndetail::content())
}

pub fn render_peers() -> String {
    render_page(&peers::meta(), &peers::content())
}

pub fn render_peers_detail() -> String {
    render_page(&peersdetail::meta(), &peersdetail::content())
}

pub fn render_asn() -> String {
    render_page(&asn::meta(), &asn::content())
}

pub fn render_asn_detail() -> String {
    render_page(&asndetail::meta(), &asndetail::content())
}

pub fn render_streaming() -> String {
    render_page(&streaming::meta(), &streaming::content())
}

pub fn render_streaming_detail() -> String {
    render_page(&streamingdetail::meta(), &streamingdetail::content())
}

pub fn render_router() -> String {
    render_page(&router::meta(), &router::content())
}

pub fn render_router_detail() -> String {
    render_page(&routerdetail::meta(), &routerdetail::content())
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
