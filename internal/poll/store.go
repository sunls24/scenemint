package poll

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunls24/gox/server"
	"go.etcd.io/bbolt"
)

const DefaultPath = "data/poll.db"

var (
	ErrInvalidFingerprint = errors.New("invalid fingerprint")
	bucketPaidSiteSupport = []byte("paid_site_support")
)

type Store struct {
	db  *bbolt.DB
	now func() time.Time
}

type VoteRequest struct {
	Fingerprint string `json:"fingerprint"`
}

type StatsResponse struct {
	Count int `json:"count"`
}

type VoteResponse struct {
	Count    int  `json:"count"`
	Recorded bool `json:"recorded"`
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("creating poll data directory: %w", err)
	}
	db, err := bbolt.Open(path, 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, fmt.Errorf("opening poll store: %w", err)
	}
	store := &Store{db: db, now: time.Now}
	if err := store.ensureBuckets(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Stats(_ context.Context) (StatsResponse, error) {
	var resp StatsResponse
	err := s.db.View(func(tx *bbolt.Tx) error {
		resp.Count = tx.Bucket(bucketPaidSiteSupport).Stats().KeyN
		return nil
	})
	return resp, err
}

func (s *Store) Vote(ctx context.Context, req VoteRequest) (VoteResponse, error) {
	fingerprint, err := normalizeFingerprint(req.Fingerprint)
	if err != nil {
		return VoteResponse{}, server.ErrMsg("浏览器指纹无效")
	}

	resp := VoteResponse{}
	err = s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(bucketPaidSiteSupport)
		key := []byte(fingerprint)
		if bucket.Get(key) == nil {
			if err := bucket.Put(key, []byte(s.now().Format(time.RFC3339))); err != nil {
				return fmt.Errorf("recording paid site support: %w", err)
			}
			resp.Recorded = true
		}
		return nil
	})
	if err != nil {
		return VoteResponse{}, err
	}
	stats, err := s.Stats(ctx)
	resp.Count = stats.Count
	return resp, err
}

func (s *Store) ensureBuckets() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketPaidSiteSupport)
		return err
	})
}

func normalizeFingerprint(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) < 8 || len(value) > 256 {
		return "", ErrInvalidFingerprint
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return "", ErrInvalidFingerprint
	}
	return value, nil
}
