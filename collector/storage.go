package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	historyDB   *sql.DB
	storageOnce sync.Once
	storageCh   chan storageJob
)

type storageJob struct {
	pt  HistoryPoint
	v4  float64
	v6  float64
}

func startStorageWriter() {
	storageCh = make(chan storageJob, 64)
	go func() {
		for job := range storageCh {
			storageInsert(job.pt, job.v4, job.v6)
		}
	}()
}

func storageEnqueue(pt HistoryPoint, v4, v6 float64) {
	if storageCh == nil {
		storageInsert(pt, v4, v6)
		return
	}
	select {
	case storageCh <- storageJob{pt: pt, v4: v4, v6: v6}:
	default:
		go storageInsert(pt, v4, v6)
	}
}

const defaultLocalHours = 72  // 3 dias na VM
const defaultS3Days     = 30  // restante no S3

func localRetentionSec() int64 {
	h := GetConfig().HistoryLocalH
	if h <= 0 {
		h = GetConfig().HistoryRetainH
	}
	if h <= 0 {
		h = defaultLocalHours
	}
	return int64(h) * 3600
}

func s3RetentionSec() int64 {
	d := GetConfig().HistoryS3Days
	if d <= 0 {
		d = defaultS3Days
	}
	return int64(d) * 86400
}

func initStorage() {
	storageOnce.Do(func() {
		path := filepath.Join(GetConfig().DataDir, "history.db")
		db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
		if err != nil {
			log.Printf("sqlite open: %v", err)
			return
		}
		schema := `
CREATE TABLE IF NOT EXISTS history_raw (
  ts INTEGER PRIMARY KEY,
  mbps REAL NOT NULL DEFAULT 0,
  mbps_scaled REAL NOT NULL DEFAULT 0,
  snmp_in REAL NOT NULL DEFAULT 0,
  snmp_out REAL NOT NULL DEFAULT 0,
  sampling_factor REAL NOT NULL DEFAULT 1,
  by_category TEXT NOT NULL DEFAULT '{}',
  by_category_scaled TEXT NOT NULL DEFAULT '{}',
  ipv4_mbps REAL NOT NULL DEFAULT 0,
  ipv6_mbps REAL NOT NULL DEFAULT 0,
  by_asn TEXT NOT NULL DEFAULT '{}',
  by_asn_scaled TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_history_raw_ts ON history_raw(ts);
CREATE TABLE IF NOT EXISTS s3_exported (
  day TEXT PRIMARY KEY,
  exported_at INTEGER NOT NULL
);
`
		if _, err := db.Exec(schema); err != nil {
			log.Printf("sqlite schema: %v", err)
			return
		}
		// Migração suave para DBs existentes
		for _, col := range []string{
			`ALTER TABLE history_raw ADD COLUMN by_asn TEXT NOT NULL DEFAULT '{}'`,
			`ALTER TABLE history_raw ADD COLUMN by_asn_scaled TEXT NOT NULL DEFAULT '{}'`,
		} {
			_, _ = db.Exec(col)
		}
		historyDB = db
		log.Printf("sqlite histórico: %s (local %dh · S3 %dd)",
			path, GetConfig().HistoryLocalH, GetConfig().HistoryS3Days)
	})
}

func storageInsert(pt HistoryPoint, ipv4Mbps, ipv6Mbps float64) {
	if historyDB == nil {
		return
	}
	catB, _ := json.Marshal(pt.ByCategory)
	scaledB, _ := json.Marshal(pt.ByCategoryScaled)
	asnB, _ := json.Marshal(pt.ByASN)
	asnScaledB, _ := json.Marshal(pt.ByASNScaled)
	_, err := historyDB.Exec(`INSERT OR REPLACE INTO history_raw
		(ts, mbps, mbps_scaled, snmp_in, snmp_out, sampling_factor, by_category, by_category_scaled, ipv4_mbps, ipv6_mbps, by_asn, by_asn_scaled)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		pt.Ts, pt.Mbps, pt.MbpsScaled, pt.SNMPIn, pt.SNMPOut, pt.SamplingFactor,
		string(catB), string(scaledB), ipv4Mbps, ipv6Mbps, string(asnB), string(asnScaledB))
	if err != nil {
		log.Printf("sqlite insert: %v", err)
	}
}

func storageQueryLocal(since int64) []HistoryPoint {
	if historyDB == nil {
		return nil
	}
	localCutoff := time.Now().Unix() - localRetentionSec()
	if since < localCutoff {
		since = localCutoff
	}
	q := `SELECT ts, mbps, mbps_scaled, snmp_in, snmp_out, sampling_factor, by_category, by_category_scaled,
		COALESCE(by_asn, '{}'), COALESCE(by_asn_scaled, '{}')
		FROM history_raw WHERE ts >= ? ORDER BY ts`
	rows, err := historyDB.Query(q, since)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []HistoryPoint
	for rows.Next() {
		var pt HistoryPoint
		var catJSON, scaledJSON, asnJSON, asnScaledJSON string
		if rows.Scan(&pt.Ts, &pt.Mbps, &pt.MbpsScaled, &pt.SNMPIn, &pt.SNMPOut, &pt.SamplingFactor,
			&catJSON, &scaledJSON, &asnJSON, &asnScaledJSON) == nil {
			_ = json.Unmarshal([]byte(catJSON), &pt.ByCategory)
			_ = json.Unmarshal([]byte(scaledJSON), &pt.ByCategoryScaled)
			_ = json.Unmarshal([]byte(asnJSON), &pt.ByASN)
			_ = json.Unmarshal([]byte(asnScaledJSON), &pt.ByASNScaled)
			out = append(out, pt)
		}
	}
	return out
}

// queryHistorySince — local (3d) + S3 (até 30d) conforme necessário.
func queryHistorySince(since int64) []HistoryPoint {
	localCutoff := time.Now().Unix() - localRetentionSec()
	s3Cutoff := time.Now().Unix() - s3RetentionSec()

	if since == 0 {
		since = s3Cutoff
	}
	if since < s3Cutoff {
		since = s3Cutoff
	}

	var out []HistoryPoint
	localSince := since
	if localSince < localCutoff {
		localSince = localCutoff
	}
	out = append(out, storageQueryLocal(localSince)...)

	if since < localCutoff && s3Enabled() {
		until := localCutoff
		if until > time.Now().Unix() {
			until = time.Now().Unix()
		}
		out = append(out, s3LoadHistoryRange(since, until)...)
	}

	if len(out) <= 1 {
		return out
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ts < out[j].Ts })
	merged := make([]HistoryPoint, 0, len(out))
	var lastTs int64 = -1
	for _, pt := range out {
		if pt.Ts == lastTs {
			continue
		}
		lastTs = pt.Ts
		merged = append(merged, pt)
	}
	return downsampleHistory(merged, 800)
}

func downsampleHistory(pts []HistoryPoint, maxPoints int) []HistoryPoint {
	if maxPoints <= 0 || len(pts) <= maxPoints {
		return pts
	}
	step := float64(len(pts)) / float64(maxPoints)
	if step < 1 {
		step = 1
	}
	out := make([]HistoryPoint, 0, maxPoints)
	for i := 0.0; i < float64(len(pts)); i += step {
		idx := int(i)
		if idx >= len(pts) {
			break
		}
		out = append(out, pts[idx])
	}
	if len(out) > 0 && out[len(out)-1].Ts != pts[len(pts)-1].Ts {
		out = append(out, pts[len(pts)-1])
	}
	return out
}

func storageExportDayToS3(day string) error {
	if historyDB == nil || !s3Enabled() {
		return nil
	}
	var exported int64
	_ = historyDB.QueryRow(`SELECT exported_at FROM s3_exported WHERE day = ?`, day).Scan(&exported)
	if exported > 0 {
		return nil
	}
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return err
	}
	start := t.Unix()
	end := start + 86400
	rows, err := historyDB.Query(`SELECT ts, mbps, mbps_scaled, snmp_in, snmp_out, sampling_factor, by_category, by_category_scaled,
		COALESCE(by_asn, '{}'), COALESCE(by_asn_scaled, '{}')
		FROM history_raw WHERE ts >= ? AND ts < ? ORDER BY ts`, start, end)
	if err != nil {
		return err
	}
	defer rows.Close()
	var pts []HistoryPoint
	for rows.Next() {
		var pt HistoryPoint
		var catJSON, scaledJSON, asnJSON, asnScaledJSON string
		if rows.Scan(&pt.Ts, &pt.Mbps, &pt.MbpsScaled, &pt.SNMPIn, &pt.SNMPOut, &pt.SamplingFactor,
			&catJSON, &scaledJSON, &asnJSON, &asnScaledJSON) == nil {
			_ = json.Unmarshal([]byte(catJSON), &pt.ByCategory)
			_ = json.Unmarshal([]byte(scaledJSON), &pt.ByCategoryScaled)
			_ = json.Unmarshal([]byte(asnJSON), &pt.ByASN)
			_ = json.Unmarshal([]byte(asnScaledJSON), &pt.ByASNScaled)
			pts = append(pts, pt)
		}
	}
	if len(pts) == 0 {
		return nil
	}
	if err := s3UploadHistoryDay(day, pts); err != nil {
		return err
	}
	_, _ = historyDB.Exec(`INSERT OR REPLACE INTO s3_exported (day, exported_at) VALUES (?, ?)`, day, time.Now().Unix())
	log.Printf("storage: dia %s arquivado no S3 (%d pontos)", day, len(pts))
	return nil
}

func storagePruneLocal() {
	if historyDB == nil {
		return
	}
	cutoff := time.Now().Unix() - localRetentionSec()

	// Exportar dias completos antes de apagar
	if s3Enabled() {
		rows, err := historyDB.Query(`
			SELECT DISTINCT strftime('%Y-%m-%d', ts, 'unixepoch') AS d
			FROM history_raw WHERE ts < ?`, cutoff)
		if err == nil {
			for rows.Next() {
				var day string
				if rows.Scan(&day) == nil && day != "" {
					_ = storageExportDayToS3(day)
				}
			}
			rows.Close()
		}
		// Garantir export do dia anterior (pode ainda ter pontos locais)
		yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
		_ = storageExportDayToS3(yesterday)
	}

	res, err := historyDB.Exec(`DELETE FROM history_raw WHERE ts < ?`, cutoff)
	if err == nil {
		if n, _ := res.RowsAffected(); n > 0 {
			log.Printf("storage: removidos %d pontos locais (>3 dias)", n)
		}
	}
	// VACUUM ocasional para compactar
	if time.Now().Hour() == 3 {
		_, _ = historyDB.Exec(`VACUUM`)
	}
	storageWALCheckpoint()
}

func storageWALCheckpoint() {
	if historyDB == nil {
		return
	}
	_, _ = historyDB.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
}

func storageDBPath() string {
	return filepath.Join(GetConfig().DataDir, "history.db")
}

func storageLocalStats() (points int, sizeBytes int64) {
	if historyDB == nil {
		return 0, 0
	}
	_ = historyDB.QueryRow(`SELECT COUNT(*) FROM history_raw WHERE ts >= ?`,
		time.Now().Unix()-localRetentionSec()).Scan(&points)
	if fi, err := os.Stat(storageDBPath()); err == nil {
		sizeBytes = fi.Size()
	}
	return
}
