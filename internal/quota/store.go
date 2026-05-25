package quota

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.etcd.io/bbolt"
)

const (
	DefaultPath = "data/quota.db"
	DailyGrant  = 10
	BalanceCap  = 100
)

var (
	ErrInvalidFingerprint = errors.New("invalid fingerprint")
	ErrNoCredits          = errors.New("no credits")
)

var bucketAccounts = []byte("accounts")

type Store struct {
	db  *bbolt.DB
	now func() time.Time
}

type Request struct {
	Fingerprint string `json:"fingerprint"`
}

type StatusResponse struct {
	Balance     int  `json:"balance"`
	SignedToday bool `json:"signedToday"`
	Cap         int  `json:"cap"`
	DailyGrant  int  `json:"dailyGrant"`
}

type account struct {
	Balance         int    `json:"balance"`
	LastCheckInDate string `json:"lastCheckInDate,omitempty"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

type Spend struct {
	store       *Store
	fingerprint string
	refunded    bool
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("creating quota data directory: %w", err)
	}
	db, err := bbolt.Open(path, 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, fmt.Errorf("opening quota store: %w", err)
	}
	store := &Store{
		db:  db,
		now: time.Now,
	}
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

func (s *Store) Get(fingerprint string) (StatusResponse, error) {
	fingerprint, err := normalizeFingerprint(fingerprint)
	if err != nil {
		return StatusResponse{}, err
	}

	var resp StatusResponse
	err = s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketAccounts)
		acct, ok, err := readAccount(b, fingerprint)
		if err != nil {
			return err
		}
		if !ok {
			resp = status(account{}, s.today())
			return nil
		}
		resp = status(acct, s.today())
		return nil
	})
	return resp, err
}

func (s *Store) ApplyCheckIn(fingerprint string) (StatusResponse, error) {
	fingerprint, err := normalizeFingerprint(fingerprint)
	if err != nil {
		return StatusResponse{}, err
	}

	var resp StatusResponse
	err = s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketAccounts)
		acct, ok, err := readAccount(b, fingerprint)
		if err != nil {
			return err
		}

		today := s.today()
		if ok && acct.LastCheckInDate == today {
			resp = status(acct, today)
			return nil
		}
		if ok && acct.Balance >= BalanceCap {
			resp = status(acct, today)
			return nil
		}

		now := s.now().Format(time.RFC3339)
		if !ok {
			acct.CreatedAt = now
		}
		acct.Balance = min(acct.Balance+DailyGrant, BalanceCap)
		acct.LastCheckInDate = today
		acct.UpdatedAt = now
		if err := writeAccount(b, fingerprint, acct); err != nil {
			return err
		}
		resp = status(acct, today)
		return nil
	})
	return resp, err
}

func (s *Store) Spend(fingerprint string) (*Spend, StatusResponse, error) {
	fingerprint, err := normalizeFingerprint(fingerprint)
	if err != nil {
		return nil, StatusResponse{}, err
	}

	spend := &Spend{store: s, fingerprint: fingerprint}
	var resp StatusResponse
	err = s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketAccounts)
		acct, ok, err := readAccount(b, fingerprint)
		if err != nil {
			return err
		}
		if !ok || acct.Balance <= 0 {
			return ErrNoCredits
		}

		acct.Balance--
		acct.UpdatedAt = s.now().Format(time.RFC3339)
		if err := writeAccount(b, fingerprint, acct); err != nil {
			return err
		}
		resp = status(acct, s.today())
		return nil
	})
	if err != nil {
		return nil, StatusResponse{}, err
	}
	return spend, resp, nil
}

func (s *Spend) Refund() error {
	if s == nil || s.store == nil || s.refunded {
		return nil
	}

	err := s.store.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketAccounts)
		acct, ok, err := readAccount(b, s.fingerprint)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		acct.Balance = min(acct.Balance+1, BalanceCap)
		acct.UpdatedAt = s.store.now().Format(time.RFC3339)
		return writeAccount(b, s.fingerprint, acct)
	})
	if err != nil {
		return err
	}
	s.refunded = true
	return nil
}

func (s *Store) ensureBuckets() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketAccounts)
		return err
	})
}

func (s *Store) today() string {
	return s.now().Format("2006-01-02")
}

func readAccount(b *bbolt.Bucket, fingerprint string) (account, bool, error) {
	raw := b.Get([]byte(fingerprint))
	if raw == nil {
		return account{}, false, nil
	}
	var acct account
	if err := json.Unmarshal(raw, &acct); err != nil {
		return account{}, false, fmt.Errorf("decoding quota account: %w", err)
	}
	if acct.Balance < 0 {
		acct.Balance = 0
	}
	if acct.Balance > BalanceCap {
		acct.Balance = BalanceCap
	}
	return acct, true, nil
}

func writeAccount(b *bbolt.Bucket, fingerprint string, acct account) error {
	raw, err := json.Marshal(acct)
	if err != nil {
		return fmt.Errorf("encoding quota account: %w", err)
	}
	if err := b.Put([]byte(fingerprint), raw); err != nil {
		return fmt.Errorf("writing quota account: %w", err)
	}
	return nil
}

func status(acct account, today string) StatusResponse {
	return StatusResponse{
		Balance:     acct.Balance,
		SignedToday: acct.LastCheckInDate == today,
		Cap:         BalanceCap,
		DailyGrant:  DailyGrant,
	}
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
