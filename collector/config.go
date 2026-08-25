package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

// AppConfig — configuração externa (YAML simples via JSON ou env).
type AppConfig struct {
	SourceIP       string  `json:"source_ip"`
	APIPort        string  `json:"api_port"`
	NetFlowPort    string  `json:"netflow_port"`
	SNMPHost       string  `json:"snmp_host"`
	SNMPPort       int     `json:"snmp_port"`
	SNMPCommunity  string  `json:"snmp_community"`
	SamplingRate   float64 `json:"sampling_rate"`
	APIToken       string  `json:"api_token"`
	UIUser         string  `json:"ui_user"`
	UIPassword     string  `json:"-"`
	DataDir        string  `json:"data_dir"`
	AlertUtilPct   float64 `json:"alert_util_pct"`
	AlertASNPct    float64 `json:"alert_asn_pct"`   // % do uplink SNMP por ASN (0=desliga)
	AlertASNMbps   float64 `json:"alert_asn_mbps"`  // Mbps absoluto por ASN (0=só %)
	AlertASNIgnore []string `json:"alert_asn_ignore"` // ASNs ignorados em asn_high_* (ex.: AS15169)
	AlertGapPct    float64  `json:"alert_gap_pct"`    // |NF×fator − SNMP| / SNMP % (0=desliga; padrão 35)
	AlertSilentSec int      `json:"alert_silent_sec"`  // NetFlow silencioso (padrão 90)
	AlertUDPQueue  int64    `json:"alert_udp_queue"`   // bytes na fila UDP kernel (padrão 8MB)
	UDPRcvBufMB    int      `json:"udp_rcvbuf_mb"`     // SO_RCVBUF pedido (padrão 32)
	IngestWorkers  int      `json:"ingest_workers"`    // workers de classificação (0=auto)
	ASNHistoryTop  int      `json:"asn_history_top"`  // top-N por amostra no histórico (padrão 30)
	ASNWatched     []string `json:"asn_watched"`      // ASNs sempre mantidos no histórico
	CDNWatched     []string `json:"cdn_watched"`      // CDNs sempre mantidos no histórico
	ASNDigestHour  int      `json:"asn_digest_hour"`  // hora local do digest diário (0-23; -1=off)
	IXASN          uint32   `json:"ix_asn"`           // ASN do IX (padrão 26162 IX.br)
	FeedIntervalM  int     `json:"feed_interval_min"`
	HistoryRetainH int     `json:"history_retain_hours"` // legado: local
	HistoryLocalH  int     `json:"history_local_hours"`  // VM: padrão 72 (3d)
	HistoryS3Days  int     `json:"history_s3_days"`      // S3: padrão 30d
	WebhookURL     string  `json:"webhook_url"`
	TelegramToken  string  `json:"telegram_bot_token"`
	TelegramChat   string  `json:"telegram_chat_id"`
	S3Endpoint     string  `json:"s3_endpoint"`
	S3Bucket       string  `json:"s3_bucket"`
	S3Region       string  `json:"s3_region"`
	S3AccessKey    string  `json:"-"`
	S3SecretKey    string  `json:"-"`
	Exporters      []string `json:"exporters"`
	EnableHTTPS    bool    `json:"enable_https"`
}

var (
	cfgMu sync.RWMutex
	cfg   = AppConfig{
		SourceIP:       "170.245.127.191",
		APIPort:        "127.0.0.1:9090",
		NetFlowPort:    ":2055",
		SNMPHost:       "170.245.127.191",
		SNMPPort:       15161,
		SNMPCommunity:  "",
		SamplingRate:   0,
		APIToken:       "",
		DataDir:        "/var/www/html/inforflow/data",
		AlertUtilPct:   80,
		AlertASNPct:    25,
		AlertASNMbps:   500,
		AlertASNIgnore: []string{
			"AS15169", // Google
			"AS32934", // Meta
			"AS13335", // Cloudflare
			"AS16509", // Amazon
			"AS8075",  // Microsoft
			"AS2906",  // Netflix
			"AS20940", // Akamai
		},
		AlertGapPct:    35,
		AlertSilentSec: 90,
		AlertUDPQueue:  8 * 1024 * 1024,
		UDPRcvBufMB:    32,
		IngestWorkers:  0,
		ASNHistoryTop: 30,
		ASNWatched:    nil,
		CDNWatched:    []string{"Cloudflare", "Akamai", "Google Cache", "AWS CloudFront", "Fastly"},
		ASNDigestHour: 8,
		IXASN:         26162,
		FeedIntervalM:  360,
		HistoryRetainH: 72,
		HistoryLocalH:  72,
		HistoryS3Days:  30,
		S3Endpoint:     "https://s3.eu-amsterdam.megas4.com",
		S3Bucket:       "inforflow",
		S3Region:       "eu-amsterdam",
	}
)

func GetConfig() AppConfig {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	return cfg
}

func LoadConfig(path string) {
	cfgMu.Lock()
	defer cfgMu.Unlock()

	if path == "" {
		path = os.Getenv("INFORFLOW_CONFIG")
	}
	if path == "" {
		path = "/var/www/html/inforflow/config.json"
	}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &cfg)
		log.Printf("config carregada de %s", path)
	} else {
		log.Printf("config padrão (sem arquivo %s): %v", path, err)
	}

	// Env overrides
	if v := os.Getenv("INFORFLOW_SOURCE_IP"); v != "" {
		cfg.SourceIP = v
	}
	if v := os.Getenv("INFORFLOW_API_PORT"); v != "" {
		if !strings.HasPrefix(v, ":") {
			v = ":" + v
		}
		cfg.APIPort = v
	}
	if v := os.Getenv("INFORFLOW_NETFLOW_PORT"); v != "" {
		if !strings.HasPrefix(v, ":") {
			v = ":" + v
		}
		cfg.NetFlowPort = v
	}
	if v := os.Getenv("INFORFLOW_SNMP_HOST"); v != "" {
		cfg.SNMPHost = v
	}
	if v := os.Getenv("INFORFLOW_SNMP_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.SNMPPort = n
		}
	}
	if v := os.Getenv("INFORFLOW_SNMP_COMMUNITY"); v != "" {
		cfg.SNMPCommunity = v
	}
	if v := os.Getenv("INFORFLOW_SAMPLING"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.SamplingRate = f
		}
	}
	if v := os.Getenv("INFORFLOW_API_TOKEN"); v != "" {
		cfg.APIToken = v
	}
	if v := os.Getenv("INFORFLOW_UI_USER"); v != "" {
		cfg.UIUser = v
	}
	if v := os.Getenv("INFORFLOW_UI_PASSWORD"); v != "" {
		cfg.UIPassword = v
	}
	if v := os.Getenv("INFORFLOW_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
	if v := os.Getenv("INFORFLOW_WEBHOOK_URL"); v != "" {
		cfg.WebhookURL = v
	}
	if v := os.Getenv("INFORFLOW_TELEGRAM_TOKEN"); v != "" {
		cfg.TelegramToken = v
	}
	if v := os.Getenv("INFORFLOW_TELEGRAM_CHAT"); v != "" {
		cfg.TelegramChat = v
	}
	if v := os.Getenv("INFORFLOW_S3_ENDPOINT"); v != "" {
		cfg.S3Endpoint = v
	}
	if v := os.Getenv("INFORFLOW_S3_BUCKET"); v != "" {
		cfg.S3Bucket = v
	}
	if v := os.Getenv("INFORFLOW_S3_REGION"); v != "" {
		cfg.S3Region = v
	}
	if v := os.Getenv("INFORFLOW_S3_ACCESS_KEY"); v != "" {
		cfg.S3AccessKey = v
	}
	if v := os.Getenv("INFORFLOW_S3_SECRET_KEY"); v != "" {
		cfg.S3SecretKey = v
	}
	if v := os.Getenv("INFORFLOW_HISTORY_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.HistoryLocalH = n
			cfg.HistoryRetainH = n
		}
	}
	if v := os.Getenv("INFORFLOW_HISTORY_S3_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.HistoryS3Days = n
		}
	}
	if cfg.HistoryLocalH <= 0 && cfg.HistoryRetainH > 0 {
		cfg.HistoryLocalH = cfg.HistoryRetainH
	}
	if cfg.HistoryLocalH <= 0 {
		cfg.HistoryLocalH = 72
	}
	if cfg.HistoryS3Days <= 0 {
		cfg.HistoryS3Days = 30
	}
	if cfg.ASNHistoryTop <= 0 {
		cfg.ASNHistoryTop = 30
	}
	if cfg.ASNDigestHour < -1 || cfg.ASNDigestHour > 23 {
		cfg.ASNDigestHour = 8
	}
	if cfg.AlertASNIgnore == nil {
		cfg.AlertASNIgnore = []string{
			"AS15169", "AS32934", "AS13335", "AS16509", "AS8075", "AS2906", "AS20940",
		}
	}
	if cfg.IXASN == 0 {
		cfg.IXASN = 26162
	}
	if cfg.AlertGapPct < 0 {
		cfg.AlertGapPct = 0
	}
	if cfg.AlertSilentSec <= 0 {
		cfg.AlertSilentSec = 90
	}
	if cfg.AlertUDPQueue <= 0 {
		cfg.AlertUDPQueue = 8 * 1024 * 1024
	}
	if cfg.UDPRcvBufMB <= 0 {
		cfg.UDPRcvBufMB = 32
	}
	_ = os.MkdirAll(cfg.DataDir, 0755)

	// Sync legacy package-level constants used elsewhere
	SourceIP = cfg.SourceIP
	SNMPHost = cfg.SNMPHost
	SNMPPort = cfg.SNMPPort
	SNMPCommunity = cfg.SNMPCommunity
}
