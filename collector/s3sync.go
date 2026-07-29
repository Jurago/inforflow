package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func s3Enabled() bool {
	c := GetConfig()
	return c.S3AccessKey != "" && c.S3SecretKey != "" && c.S3Bucket != ""
}

func s3Service() (*s3.S3, error) {
	c := GetConfig()
	endpoint := strings.TrimRight(c.S3Endpoint, "/")
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(c.S3Region),
		Credentials:      credentials.NewStaticCredentials(c.S3AccessKey, c.S3SecretKey, ""),
		Endpoint:         aws.String(endpoint),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	return s3.New(sess), nil
}

func s3UploadBytes(data []byte, objectKey, contentType string) error {
	if !s3Enabled() {
		return nil
	}
	svc, err := s3Service()
	if err != nil {
		return err
	}
	c := GetConfig()
	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(c.S3Bucket),
		Key:         aws.String(objectKey),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	return err
}

func s3UploadFile(localPath, objectKey string) error {
	b, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	return s3UploadBytes(b, objectKey, "application/octet-stream")
}

func s3UploadHistoryDay(day string, pts []HistoryPoint) error {
	b, err := json.Marshal(pts)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, _ = gw.Write(b)
	_ = gw.Close()
	key := fmt.Sprintf("history/daily/%s.json.gz", day)
	return s3UploadBytes(buf.Bytes(), key, "application/gzip")
}

func s3DownloadBytes(objectKey string) ([]byte, error) {
	if !s3Enabled() {
		return nil, fmt.Errorf("s3 disabled")
	}
	svc, err := s3Service()
	if err != nil {
		return nil, err
	}
	c := GetConfig()
	out, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(c.S3Bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

func s3LoadHistoryRange(since, until int64) []HistoryPoint {
	if !s3Enabled() || since >= until {
		return nil
	}
	var out []HistoryPoint
	startDay := time.Unix(since, 0).UTC()
	for d := startDay; d.Unix() < until; d = d.Add(24 * time.Hour) {
		day := d.Format("2006-01-02")
		key := fmt.Sprintf("history/daily/%s.json.gz", day)
		raw, err := s3DownloadBytes(key)
		if err != nil {
			continue
		}
		gz, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			continue
		}
		body, err := io.ReadAll(gz)
		_ = gz.Close()
		if err != nil {
			continue
		}
		var pts []HistoryPoint
		if json.Unmarshal(body, &pts) != nil {
			continue
		}
		for _, pt := range pts {
			if pt.Ts >= since && pt.Ts < until {
				out = append(out, pt)
			}
		}
	}
	return out
}

func s3SyncHistoryExport() {
	if !s3Enabled() {
		return
	}
	yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	_ = storageExportDayToS3(yesterday)
}

func s3SyncDBBackup() {
	if !s3Enabled() {
		return
	}
	dbPath := storageDBPath()
	if _, err := os.Stat(dbPath); err != nil {
		return
	}
	ts := time.Now().Format("2006-01-02-15")
	key := fmt.Sprintf("backups/history-%s.db", ts)
	if err := s3UploadFile(dbPath, key); err != nil {
		log.Printf("s3 backup db: %v", err)
	} else {
		log.Printf("s3: backup %s", key)
	}
}

func s3SyncFlowsSnapshot() {
	if !s3Enabled() {
		return
	}
	flows := store.GetRecentFlows(500)
	if len(flows) == 0 {
		return
	}
	b, _ := json.Marshal(flows)
	ts := time.Now().Format("2006-01-02-15")
	key := fmt.Sprintf("flows/snapshot-%s.json", ts)
	_ = s3UploadBytes(b, key, "application/json")
}

func StartS3Sync() {
	if !s3Enabled() {
		log.Printf("s3: desabilitado (defina INFORFLOW_S3_* no ambiente)")
		return
	}
	log.Printf("s3: sync ativo → bucket %s @ %s", GetConfig().S3Bucket, GetConfig().S3Endpoint)
	go func() {
		time.Sleep(30 * time.Second)
		s3SyncDBBackup()
		s3SyncHistoryExport()
		ticker := time.NewTicker(6 * time.Hour)
		for range ticker.C {
			s3SyncDBBackup()
			s3SyncHistoryExport()
			s3SyncFlowsSnapshot()
		}
	}()
}

func s3ListBackups(prefix string) ([]string, error) {
	if !s3Enabled() {
		return nil, fmt.Errorf("s3 disabled")
	}
	svc, err := s3Service()
	if err != nil {
		return nil, err
	}
	c := GetConfig()
	out, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(c.S3Bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, o := range out.Contents {
		if o.Key != nil {
			keys = append(keys, *o.Key)
		}
	}
	return keys, nil
}
