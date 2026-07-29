package main

import (
	"fmt"
	"net"
	"strings"
)

type cidrRule struct {
	net      *net.IPNet
	name     string
	category FlowCategory
	icon     string
	asn      string
}

var classifyRules []cidrRule

func init() {
	add := func(cidr, name string, cat FlowCategory, icon, asn string) {
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			return
		}
		classifyRules = append(classifyRules, cidrRule{n, name, cat, icon, asn})
	}

	// Netflix
	add("45.57.0.0/16", "Netflix", CategoryNetflix, "netflix", "AS2906")
	add("198.38.96.0/19", "Netflix", CategoryNetflix, "netflix", "AS2906")
	add("198.45.48.0/20", "Netflix", CategoryNetflix, "netflix", "AS2906")
	add("23.246.0.0/18", "Netflix", CategoryNetflix, "netflix", "AS2906")
	add("37.77.184.0/21", "Netflix", CategoryNetflix, "netflix", "AS2906")
	add("108.175.32.0/20", "Netflix", CategoryNetflix, "netflix", "AS2906")
	add("185.2.220.0/22", "Netflix", CategoryNetflix, "netflix", "AS2906")
	add("185.9.188.0/22", "Netflix", CategoryNetflix, "netflix", "AS2906")
	add("192.173.64.0/18", "Netflix", CategoryNetflix, "netflix", "AS2906")
	add("207.45.72.0/22", "Netflix", CategoryNetflix, "netflix", "AS2906")
	add("208.75.76.0/22", "Netflix", CategoryNetflix, "netflix", "AS2906")
	add("69.53.224.0/19", "Netflix", CategoryNetflix, "netflix", "AS2906")
	add("64.120.128.0/17", "Netflix", CategoryNetflix, "netflix", "AS2906")

	// Globo / Globosat
	add("200.221.0.0/16", "Globo", CategoryGlobo, "globo", "AS28604")
	add("189.14.0.0/16", "Globo", CategoryGlobo, "globo", "AS28604")
	add("186.192.0.0/16", "Globo", CategoryGlobo, "globo", "AS28604")
	add("201.0.0.0/16", "Globo", CategoryGlobo, "globo", "AS7738")
	add("177.54.144.0/20", "Globo", CategoryGlobo, "globo", "AS28604")
	add("200.143.192.0/19", "Globo", CategoryGlobo, "globo", "AS7738")

	// Cloudflare
	add("1.1.1.0/24", "Cloudflare", CategoryCDN, "cloudflare", "AS13335")
	add("1.0.0.0/24", "Cloudflare", CategoryCDN, "cloudflare", "AS13335")
	add("104.16.0.0/12", "Cloudflare", CategoryCDN, "cloudflare", "AS13335")
	add("172.64.0.0/13", "Cloudflare", CategoryCDN, "cloudflare", "AS13335")
	add("173.245.48.0/20", "Cloudflare", CategoryCDN, "cloudflare", "AS13335")
	add("103.21.244.0/22", "Cloudflare", CategoryCDN, "cloudflare", "AS13335")
	add("103.22.200.0/22", "Cloudflare", CategoryCDN, "cloudflare", "AS13335")
	add("103.31.4.0/22", "Cloudflare", CategoryCDN, "cloudflare", "AS13335")
	add("141.101.64.0/18", "Cloudflare", CategoryCDN, "cloudflare", "AS13335")
	add("108.162.192.0/18", "Cloudflare", CategoryCDN, "cloudflare", "AS13335")
	add("190.93.240.0/20", "Cloudflare", CategoryCDN, "cloudflare", "AS13335")
	add("188.114.96.0/20", "Cloudflare", CategoryCDN, "cloudflare", "AS13335")
	add("197.234.240.0/22", "Cloudflare", CategoryCDN, "cloudflare", "AS13335")
	add("198.41.128.0/17", "Cloudflare", CategoryCDN, "cloudflare", "AS13335")
	add("162.158.0.0/15", "Cloudflare", CategoryCDN, "cloudflare", "AS13335")
	add("104.234.0.0/16", "Cloudflare", CategoryCDN, "cloudflare", "AS13335")

	// Akamai (incl. 95.100/15 visto no tráfego BR)
	add("23.0.0.0/12", "Akamai", CategoryCDN, "akamai", "AS20940")
	add("23.32.0.0/11", "Akamai", CategoryCDN, "akamai", "AS20940")
	add("23.192.0.0/11", "Akamai", CategoryCDN, "akamai", "AS20940")
	add("95.100.0.0/15", "Akamai", CategoryCDN, "akamai", "AS20940")
	add("96.16.0.0/15", "Akamai", CategoryCDN, "akamai", "AS20940")
	add("104.64.0.0/10", "Akamai", CategoryCDN, "akamai", "AS20940")
	add("184.24.0.0/13", "Akamai", CategoryCDN, "akamai", "AS20940")
	add("184.50.0.0/15", "Akamai", CategoryCDN, "akamai", "AS20940")
	add("2.16.0.0/13", "Akamai", CategoryCDN, "akamai", "AS20940")
	add("72.246.0.0/15", "Akamai", CategoryCDN, "akamai", "AS20940")
	add("88.221.0.0/16", "Akamai", CategoryCDN, "akamai", "AS20940")
	add("92.122.0.0/15", "Akamai", CategoryCDN, "akamai", "AS20940")
	add("184.84.0.0/14", "Akamai", CategoryCDN, "akamai", "AS20940")

	// Fastly
	add("151.101.0.0/16", "Fastly", CategoryCDN, "fastly", "AS54113")
	add("199.232.0.0/16", "Fastly", CategoryCDN, "fastly", "AS54113")
	add("23.235.32.0/20", "Fastly", CategoryCDN, "fastly", "AS54113")
	add("167.82.0.0/17", "Fastly", CategoryCDN, "fastly", "AS54113")

	// AWS CloudFront (antes das faixas genéricas AWS cloud)
	add("13.32.0.0/15", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("13.35.0.0/16", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("13.224.0.0/12", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("18.64.0.0/14", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("18.160.0.0/15", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("18.164.0.0/15", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("18.172.0.0/15", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("52.84.0.0/15", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("52.222.128.0/17", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("54.182.0.0/16", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("54.192.0.0/12", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("54.230.0.0/16", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("54.239.128.0/18", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("99.84.0.0/16", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("108.138.0.0/15", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("108.156.0.0/14", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("143.204.0.0/16", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("204.246.164.0/22", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("205.251.192.0/19", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("216.137.32.0/19", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")

	// CDN77
	add("79.127.128.0/17", "CDN77", CategoryCDN, "cdn", "AS60068")
	add("89.187.160.0/19", "CDN77", CategoryCDN, "cdn", "AS60068")
	add("185.59.220.0/22", "CDN77", CategoryCDN, "cdn", "AS60068")
	add("185.246.180.0/22", "CDN77", CategoryCDN, "cdn", "AS60068")
	add("138.199.0.0/16", "CDN77", CategoryCDN, "cdn", "AS60068")

	// BunnyCDN
	add("143.244.66.0/23", "BunnyCDN", CategoryCDN, "cdn", "AS262254")
	add("185.93.0.0/22", "BunnyCDN", CategoryCDN, "cdn", "AS262254")
	add("45.134.212.0/22", "BunnyCDN", CategoryCDN, "cdn", "AS262254")
	add("103.72.192.0/22", "BunnyCDN", CategoryCDN, "cdn", "AS262254")

	// Edgecast / Edgio
	add("152.195.0.0/16", "Edgecast", CategoryCDN, "cdn", "AS15133")
	add("192.229.128.0/17", "Edgecast", CategoryCDN, "cdn", "AS15133")
	add("68.232.32.0/19", "Edgecast", CategoryCDN, "cdn", "AS15133")

	// Limelight / Edgio
	add("68.142.64.0/18", "Limelight", CategoryCDN, "cdn", "AS22822")
	add("208.111.128.0/18", "Limelight", CategoryCDN, "cdn", "AS22822")

	// Imperva / Incapsula
	add("199.83.128.0/21", "Imperva", CategoryCDN, "cdn", "AS19551")
	add("45.60.0.0/16", "Imperva", CategoryCDN, "cdn", "AS19551")
	add("107.154.0.0/16", "Imperva", CategoryCDN, "cdn", "AS19551")

	// Cachefly
	add("205.234.175.0/24", "Cachefly", CategoryCDN, "cdn", "AS30081")
	add("205.196.16.0/20", "Cachefly", CategoryCDN, "cdn", "AS30081")

	// G-Core Labs
	add("92.223.0.0/16", "G-Core", CategoryCDN, "cdn", "AS199524")
	add("92.38.48.0/21", "G-Core", CategoryCDN, "cdn", "AS199524")
	add("5.188.0.0/16", "G-Core", CategoryCDN, "cdn", "AS199524")

	// QUIC.cloud
	add("103.131.188.0/22", "QUIC.cloud", CategoryCDN, "cdn", "AS13335")
	add("154.83.0.0/16", "QUIC.cloud", CategoryCDN, "cdn", "AS13335")

	// Azure Front Door / CDN (antes do bloco Azure genérico)
	add("13.107.213.0/24", "Azure CDN", CategoryCDN, "cdn", "AS8075")
	add("13.107.246.0/24", "Azure CDN", CategoryCDN, "cdn", "AS8075")
	add("150.171.0.0/16", "Azure CDN", CategoryCDN, "cdn", "AS8075")
	add("204.79.197.0/24", "Azure CDN", CategoryCDN, "cdn", "AS8075")

	// Google / YouTube
	add("172.217.0.0/16", "YouTube", CategoryStreaming, "youtube", "AS15169")
	add("142.250.0.0/15", "YouTube", CategoryStreaming, "youtube", "AS15169")
	add("74.125.0.0/16", "YouTube", CategoryStreaming, "youtube", "AS15169")
	add("108.177.0.0/17", "YouTube", CategoryStreaming, "youtube", "AS15169")
	add("209.85.128.0/17", "YouTube", CategoryStreaming, "youtube", "AS15169")
	add("216.58.192.0/19", "YouTube", CategoryStreaming, "youtube", "AS15169")
	add("64.233.160.0/19", "YouTube", CategoryStreaming, "youtube", "AS15169")
	add("66.102.0.0/20", "YouTube", CategoryStreaming, "youtube", "AS15169")
	add("66.249.64.0/19", "YouTube", CategoryStreaming, "youtube", "AS15169")

	// Spotify
	add("35.186.224.0/19", "Spotify", CategoryStreaming, "spotify", "AS15169")
	add("104.199.64.0/18", "Spotify", CategoryStreaming, "spotify", "AS15169")
	add("78.31.8.0/22", "Spotify", CategoryStreaming, "spotify", "AS200147")
	add("193.182.8.0/21", "Spotify", CategoryStreaming, "spotify", "AS200147")

	// Twitch
	add("52.223.192.0/18", "Twitch", CategoryStreaming, "twitch", "AS46489")
	add("185.42.204.0/22", "Twitch", CategoryStreaming, "twitch", "AS46489")

	// Disney+
	add("99.86.0.0/16", "Disney+", CategoryStreaming, "disney", "AS16509")

	// HBO / Max
	add("3.160.0.0/14", "HBO Max", CategoryStreaming, "hbo", "AS16509")

	// Facebook / Instagram / Meta → Social
	add("31.13.64.0/18", "Meta / WhatsApp", CategorySocial, "social", "AS32934")
	add("157.240.0.0/16", "Meta / WhatsApp", CategorySocial, "social", "AS32934")
	add("69.171.224.0/19", "Meta / WhatsApp", CategorySocial, "social", "AS32934")
	add("57.144.0.0/14", "Meta / WhatsApp", CategorySocial, "social", "AS32934")
	add("129.134.0.0/16", "Meta / WhatsApp", CategorySocial, "social", "AS32934")
	add("179.60.192.0/22", "Meta / WhatsApp", CategorySocial, "social", "AS32934")
	add("185.60.216.0/22", "Meta / WhatsApp", CategorySocial, "social", "AS32934")

	// TikTok / ByteDance
	add("103.115.0.0/16", "TikTok", CategorySocial, "social", "AS138699")
	add("110.238.0.0/16", "TikTok", CategorySocial, "social", "AS138699")
	add("119.28.0.0/16", "TikTok", CategorySocial, "social", "AS132203")

	// Twitter / X
	add("104.244.42.0/24", "Twitter / X", CategorySocial, "social", "AS13414")
	add("192.133.76.0/22", "Twitter / X", CategorySocial, "social", "AS13414")

	// Microsoft / Azure CDN / Xbox — split CDN vs cloud vs gaming below
	add("13.64.0.0/11", "Azure", CategoryCloud, "cloud", "AS8075")
	add("20.33.0.0/16", "Azure", CategoryCloud, "cloud", "AS8075")
	add("40.64.0.0/10", "Azure", CategoryCloud, "cloud", "AS8075")
	add("52.112.0.0/14", "Xbox Live", CategoryGaming, "gaming", "AS8075")
	add("52.120.0.0/14", "Xbox Live", CategoryGaming, "gaming", "AS8075")

	// IX.br / peers comuns BR (LAN PTT + prefixes)
	add("200.160.0.0/20", "Peer IX.br", CategoryPeer, "peer", "AS26162")
	add("200.219.144.0/20", "Peer IX.br", CategoryPeer, "peer", "AS26162")
	add("200.219.139.0/24", "Peer IX.br", CategoryPeer, "peer", "AS26162")
	add("187.16.216.0/21", "Peer IX.br", CategoryPeer, "peer", "AS26162")
	add("45.68.80.0/23", "Peer IX.br", CategoryPeer, "peer", "AS26162")

	// Telia
	add("80.239.0.0/16", "Peer Telia", CategoryPeer, "peer", "AS1299")
	add("62.115.0.0/16", "Peer Telia", CategoryPeer, "peer", "AS1299")

	// Cogent
	add("38.0.0.0/8", "Peer Cogent", CategoryPeer, "peer", "AS174")

	// Hurricane Electric
	add("184.104.0.0/15", "Peer HE", CategoryPeer, "peer", "AS6939")
	add("216.218.128.0/17", "Peer HE", CategoryPeer, "peer", "AS6939")

	// Google Cache / GGC common
	add("35.190.0.0/16", "Google Cache", CategoryCDN, "cdn", "AS15169")
	add("35.201.0.0/16", "Google Cache", CategoryCDN, "cdn", "AS15169")
	add("35.227.0.0/16", "Google Cache", CategoryCDN, "cdn", "AS15169")
	add("45.90.28.0/24", "Google Cache", CategoryCDN, "cdn", "AS15169")
	// note: 34.x handled as Google Cloud below when not CDN-specific

	// --- IPv6 major services ---
	add("2001:4860::/32", "YouTube", CategoryStreaming, "youtube", "AS15169")
	add("2607:f8b0::/32", "YouTube", CategoryStreaming, "youtube", "AS15169")
	add("2404:6800::/32", "YouTube", CategoryStreaming, "youtube", "AS15169")
	add("2a00:1450::/32", "YouTube", CategoryStreaming, "youtube", "AS15169")

	add("2606:4700::/32", "Cloudflare", CategoryCDN, "cloudflare", "AS13335")
	add("2803:f800::/32", "Cloudflare", CategoryCDN, "cloudflare", "AS13335")
	add("2a06:98c1::/32", "Cloudflare", CategoryCDN, "cloudflare", "AS13335")

	add("2600:9000::/28", "AWS CloudFront", CategoryCDN, "cloudfront", "AS16509")
	add("2a03:2880::/32", "Meta / WhatsApp", CategorySocial, "social", "AS32934")
	add("2a04:4e40::/32", "Fastly", CategoryCDN, "fastly", "AS54113")
	add("2a04:4e42::/32", "Fastly", CategoryCDN, "fastly", "AS54113")
	add("2600:1400::/24", "Akamai", CategoryCDN, "akamai", "AS20940")
	add("2a02:26f0::/32", "Akamai", CategoryCDN, "akamai", "AS20940")

	add("2a00:86c0::/32", "Netflix", CategoryNetflix, "netflix", "AS2906")
	add("2607:fb10::/32", "Netflix", CategoryNetflix, "netflix", "AS2906")

	add("2603:1000::/25", "Azure", CategoryCloud, "cloud", "AS8075")
	add("2a01:111::/32", "Azure", CategoryCloud, "cloud", "AS8075")

	// Google LATAM / Cloud
	add("2800:3f0::/32", "YouTube", CategoryStreaming, "youtube", "AS15169")
	add("35.184.0.0/13", "Google Cloud", CategoryCloud, "cloud", "AS15169")
	add("34.64.0.0/11", "Google Cloud", CategoryCloud, "cloud", "AS15169")
	add("34.96.0.0/12", "Google Cloud", CategoryCloud, "cloud", "AS15169")

	// AWS (non-CloudFront compute)
	add("3.0.0.0/8", "AWS", CategoryCloud, "cloud", "AS16509")
	add("18.0.0.0/8", "AWS", CategoryCloud, "cloud", "AS16509")
	add("52.0.0.0/10", "AWS", CategoryCloud, "cloud", "AS16509")
	add("54.0.0.0/9", "AWS", CategoryCloud, "cloud", "AS16509")
	add("2600:1f00::/24", "AWS", CategoryCloud, "cloud", "AS16509")

	// Apple
	add("17.0.0.0/8", "Apple", CategoryApple, "apple", "AS714")
	add("2620:149::/32", "Apple", CategoryApple, "apple", "AS714")
	add("2a01:b740::/32", "Apple", CategoryApple, "apple", "AS714")
	add("2603:3000::/24", "Apple", CategoryApple, "apple", "AS714")

	// Gaming
	add("162.254.192.0/21", "Steam", CategoryGaming, "gaming", "AS32590")
	add("155.133.224.0/19", "Steam", CategoryGaming, "gaming", "AS32590")
	add("185.25.180.0/22", "Riot Games", CategoryGaming, "gaming", "AS6507")
	add("104.160.128.0/19", "Riot Games", CategoryGaming, "gaming", "AS6507")
	add("146.66.152.0/21", "Steam", CategoryGaming, "gaming", "AS32590")
	add("192.81.240.0/21", "PlayStation", CategoryGaming, "gaming", "AS33351")
	add("209.200.144.0/20", "PlayStation", CategoryGaming, "gaming", "AS33351")
	add("3.22.0.0/15", "Epic Games", CategoryGaming, "gaming", "AS16509")
}

type Classified struct {
	Name     string
	Category FlowCategory
	Icon     string
	ASN      string
}

func classifyIP(ip net.IP) Classified {
	if ip == nil {
		return Classified{Name: "unknown", Category: CategoryOther, Icon: "default", ASN: ""}
	}
	ip4 := ip.To4()
	check := ip
	if ip4 != nil {
		check = ip4
	}
	for _, r := range classifyRules {
		if r.net.Contains(check) {
			return Classified{Name: r.name, Category: r.category, Icon: r.icon, ASN: r.asn}
		}
	}
	return Classified{
		Name:     check.String(),
		Category: CategoryOther,
		Icon:     "default",
		ASN:      "",
	}
}

func classifyByASN(as uint32) Classified {
	switch as {
	case 2906:
		return Classified{"Netflix", CategoryNetflix, "netflix", "AS2906"}
	case 28604, 7738:
		return Classified{"Globo", CategoryGlobo, "globo", "AS28604"}
	case 13335:
		return Classified{"Cloudflare", CategoryCDN, "cloudflare", "AS13335"}
	case 20940:
		return Classified{"Akamai", CategoryCDN, "akamai", "AS20940"}
	case 54113:
		return Classified{"Fastly", CategoryCDN, "fastly", "AS54113"}
	case 16509:
		return Classified{"AWS CloudFront", CategoryCDN, "cloudfront", "AS16509"}
	case 60068:
		return Classified{"CDN77", CategoryCDN, "cdn", "AS60068"}
	case 15133:
		return Classified{"Edgecast", CategoryCDN, "cdn", "AS15133"}
	case 22822:
		return Classified{"Limelight", CategoryCDN, "cdn", "AS22822"}
	case 19551:
		return Classified{"Imperva", CategoryCDN, "cdn", "AS19551"}
	case 30081:
		return Classified{"Cachefly", CategoryCDN, "cdn", "AS30081"}
	case 199524:
		return Classified{"G-Core", CategoryCDN, "cdn", "AS199524"}
	case 15169:
		return Classified{"YouTube", CategoryStreaming, "youtube", "AS15169"}
	case 46489:
		return Classified{"Twitch", CategoryStreaming, "twitch", "AS46489"}
	case 32934:
		return Classified{"Meta / WhatsApp", CategorySocial, "social", "AS32934"}
	case 714:
		return Classified{"Apple", CategoryApple, "apple", "AS714"}
	case 32590:
		return Classified{"Steam", CategoryGaming, "gaming", "AS32590"}
	case 6507:
		return Classified{"Riot Games", CategoryGaming, "gaming", "AS6507"}
	case 13414:
		return Classified{"Twitter / X", CategorySocial, "social", "AS13414"}
	case 26162:
		return Classified{"Peer IX.br", CategoryPeer, "peer", "AS26162"}
	case 1299:
		return Classified{"Peer Telia", CategoryPeer, "peer", "AS1299"}
	case 174:
		return Classified{"Peer Cogent", CategoryPeer, "peer", "AS174"}
	case 6939:
		return Classified{"Peer HE", CategoryPeer, "peer", "AS6939"}
	case 8075:
		return Classified{"Azure", CategoryCloud, "cloud", "AS8075"}
	default:
		return Classified{}
	}
}

// classifyBGPSession matches exact BGP peer remote address (session IP).
func classifyBGPSession(ip net.IP) Classified {
	if ip == nil {
		return Classified{}
	}
	p, ok := bgpStore.LookupIP(ip.String())
	if !ok {
		return Classified{}
	}
	name := "Peer " + p.Name
	if p.Role == "ix" {
		name = "Peer IX.br"
	}
	return Classified{
		Name:     name,
		Category: CategoryPeer,
		Icon:     "peer",
		ASN:      p.ASN,
	}
}

// classifyBGPByAS uses live BGP sessions for non-content peers.
func classifyBGPByAS(as uint32) Classified {
	if as == 0 {
		return Classified{}
	}
	p, ok := bgpStore.LookupAS(as)
	if !ok {
		return Classified{}
	}
	// Content ASNs keep service classification (YouTube/Meta/CDN); only
	// session IPs become CategoryPeer via classifyBGPSession.
	if p.Role == "content" {
		return Classified{}
	}
	name := "Peer " + p.Name
	if p.Role == "ix" {
		name = "Peer IX.br"
	}
	return Classified{
		Name:     name,
		Category: CategoryPeer,
		Icon:     "peer",
		ASN:      p.ASN,
	}
}

func parseASNNum(asn string) uint32 {
	if asn == "" {
		return 0
	}
	s := asn
	if len(s) > 2 && (s[0] == 'A' || s[0] == 'a') {
		s = s[2:]
	}
	var n uint32
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + uint32(c-'0')
	}
	return n
}

// isTrackableASN filtra lixo comum (campo NetFlow DstAS lido como IP/contador).
func isTrackableASN(as uint32) bool {
	return plausibleAS(as)
}

// isVerifiedASN — só ASN com nome/origem confiável (evita offsets NetFlow errados).
func isVerifiedASN(as uint32) bool {
	if !isTrackableASN(as) {
		return false
	}
	if classifyByASN(as).Name != "" {
		return true
	}
	if _, ok := knownASNNames[as]; ok {
		return true
	}
	if _, ok := bgpStore.LookupAS(as); ok {
		return true
	}
	if ipapi.LookupASN(as) != nil {
		return true
	}
	return false
}

func enrichClassification(ip net.IP, as uint32) Classified {
	// 1) Exact BGP session IP → always peer interconnection
	if b := classifyBGPSession(ip); b.Name != "" {
		return b
	}

	// 2) Static CIDR rules
	c := classifyIP(ip)
	if c.Category != CategoryOther {
		return c
	}

	// 3) Dynamic feeds (Cloudflare etc.)
	if f := feedClassify(ip); f.Name != "" {
		return f
	}

	// 4) Cache ip-api.com (IP → ASN/nome) — fonte principal de nomes
	if ip != nil && !isPrivateOrSpecial(ip) {
		if info := ipapi.LookupIP(ip.String()); info != nil && info.ASN > 0 {
			if a := classifyByASN(info.ASN); a.Name != "" {
				return a
			}
			if b := classifyBGPByAS(info.ASN); b.Name != "" {
				return b
			}
			name := info.Name
			if name == "" {
				name = asnDisplayName(info.ASN)
			}
			return Classified{
				Name:     name,
				Category: CategoryOther,
				Icon:     "peer",
				ASN:      fmt.Sprintf("AS%d", info.ASN),
			}
		}
		ipapi.Enqueue(ip)
	}

	if as > 0 && isVerifiedASN(as) {
		a := classifyByASN(as)
		if a.Name != "" {
			return a
		}
		if b := classifyBGPByAS(as); b.Name != "" {
			return b
		}
		if n := ipapi.NameForASN(as); n != "" {
			return Classified{
				Name:     n,
				Category: CategoryOther,
				Icon:     "peer",
				ASN:      "AS" + itoa(int(as)),
			}
		}
		if n, ok := knownASNNames[as]; ok {
			return Classified{
				Name:     n,
				Category: CategoryOther,
				Icon:     "peer",
				ASN:      "AS" + itoa(int(as)),
			}
		}
	}

	return c
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func isPrivateOrSpecial(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	ip4 := ip.To4()
	if ip4 != nil {
		if ip4[0] == 10 {
			return true
		}
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		// CGNAT 100.64.0.0/10
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
		// Rede monitorada / exporter
		if ip4[0] == 170 && ip4[1] == 245 {
			return true
		}
		return false
	}
	// IPv6 ULA
	if len(ip) == net.IPv6len && ip[0] == 0xfd {
		return true
	}
	// Prefixo local observado no NetFlow (clientes 2804:3944::/32)
	if len(ip) == net.IPv6len && ip[0] == 0x28 && ip[1] == 0x04 && ip[2] == 0x39 && ip[3] == 0x44 {
		return true
	}
	return false
}

func isAggregateLabel(name string) bool {
	if name == "" || name == SourceIP {
		return false
	}
	ip := net.ParseIP(name)
	if ip == nil {
		return true // named service
	}
	return !isPrivateOrSpecial(ip)
}

func labelForEndpoint(ip net.IP, as uint32) (name string, cat FlowCategory, icon, asn string) {
	c := enrichClassification(ip, as)
	if strings.Contains(c.Name, ".") && c.Category == CategoryOther {
		return c.Name, c.Category, c.Icon, c.ASN
	}
	return c.Name, c.Category, c.Icon, c.ASN
}

func classifyByPort(srcPort, dstPort uint16) Classified {
	p1, p2 := srcPort, dstPort
	if p1 == 53 || p2 == 53 || p1 == 853 || p2 == 853 {
		return Classified{"DNS", CategoryDNS, "dns", ""}
	}
	if p1 == 853 || p2 == 853 || p1 == 5353 || p2 == 5353 {
		return Classified{"DNS", CategoryDNS, "dns", ""}
	}
	// QUIC / HTTP3 often streaming or CDN — leave to IP rules
	gamePorts := map[uint16]string{
		27015: "Steam", 27016: "Steam", 27017: "Steam", 27018: "Steam",
		27019: "Steam", 27020: "Steam", 27031: "Steam", 27036: "Steam",
		3074: "Xbox Live", 3075: "Xbox Live", 7500: "Xbox Live",
		3478: "PlayStation", 3479: "PlayStation", 3658: "PlayStation",
		5000: "Riot Games", 7000: "Riot Games",
		7777: "Epic Games", 7778: "Epic Games",
		1119: "Battle.net", 1120: "Battle.net", 6112: "Battle.net",
		3724: "WoW", 6113: "Battle.net",
		1935: "RTMP", // often streaming ingest
	}
	if name, ok := gamePorts[p1]; ok {
		if name == "RTMP" {
			return Classified{name, CategoryStreaming, "streaming", ""}
		}
		return Classified{name, CategoryGaming, "gaming", ""}
	}
	if name, ok := gamePorts[p2]; ok {
		if name == "RTMP" {
			return Classified{name, CategoryStreaming, "streaming", ""}
		}
		return Classified{name, CategoryGaming, "gaming", ""}
	}
	// WhatsApp voice/video call ports (heuristic)
	if p1 == 3478 || p2 == 3478 || p1 == 4244 || p2 == 4244 {
		return Classified{"Meta / WhatsApp", CategorySocial, "social", "AS32934"}
	}
	return Classified{}
}
