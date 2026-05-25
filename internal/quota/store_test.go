package quota

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"go.etcd.io/bbolt"
)

const testFingerprint = "9a3f2c1234567890d8e1"

func TestGetMissingFingerprintReturnsZero(t *testing.T) {
	t.Parallel()
	store := newTestStore(t, time.Date(2026, 5, 23, 10, 0, 0, 0, time.Local))

	got, err := store.Get(testFingerprint)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.Balance != 0 || got.SignedToday {
		t.Fatalf("Get() = %+v, want zero unsigned quota", got)
	}
	if got.Cap != BalanceCap || got.DailyGrant != DailyGrant {
		t.Fatalf("Get() metadata = %+v, want quota constants", got)
	}
}

func TestCheckInGrantsDailyCreditsOnce(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 5, 23, 10, 0, 0, 0, time.Local)
	store := newTestStore(t, now)

	first, err := store.ApplyCheckIn(testFingerprint)
	if err != nil {
		t.Fatalf("ApplyCheckIn returned error: %v", err)
	}
	if first.Balance != DailyGrant || !first.SignedToday {
		t.Fatalf("first check-in = %+v, want %d signed credits", first, DailyGrant)
	}

	second, err := store.ApplyCheckIn(testFingerprint)
	if err != nil {
		t.Fatalf("second ApplyCheckIn returned error: %v", err)
	}
	if second.Balance != DailyGrant || !second.SignedToday {
		t.Fatalf("second check-in = %+v, want unchanged signed credits", second)
	}
}

func TestCheckInCapsBalanceAtOneHundred(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 5, 23, 10, 0, 0, 0, time.Local)
	store := newTestStore(t, now)
	writeTestAccount(t, store, testFingerprint, account{
		Balance:         95,
		LastCheckInDate: "2026-05-22",
		CreatedAt:       "2026-05-22T10:00:00+08:00",
		UpdatedAt:       "2026-05-22T10:00:00+08:00",
	})

	got, err := store.ApplyCheckIn(testFingerprint)
	if err != nil {
		t.Fatalf("ApplyCheckIn returned error: %v", err)
	}
	if got.Balance != BalanceCap || !got.SignedToday {
		t.Fatalf("ApplyCheckIn = %+v, want capped signed balance", got)
	}
}

func TestCheckInAtCapDoesNotConsumeToday(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 5, 23, 10, 0, 0, 0, time.Local)
	store := newTestStore(t, now)
	writeTestAccount(t, store, testFingerprint, account{
		Balance:         BalanceCap,
		LastCheckInDate: "2026-05-22",
		CreatedAt:       "2026-05-22T10:00:00+08:00",
		UpdatedAt:       "2026-05-22T10:00:00+08:00",
	})

	got, err := store.ApplyCheckIn(testFingerprint)
	if err != nil {
		t.Fatalf("ApplyCheckIn returned error: %v", err)
	}
	if got.Balance != BalanceCap || got.SignedToday {
		t.Fatalf("ApplyCheckIn = %+v, want full balance without today's sign-in", got)
	}

	spend, afterSpend, err := store.Spend(testFingerprint)
	if err != nil {
		t.Fatalf("Spend returned error: %v", err)
	}
	if afterSpend.Balance != BalanceCap-1 || afterSpend.SignedToday {
		t.Fatalf("Spend = %+v, want one credit spent and still unsigned today", afterSpend)
	}
	t.Cleanup(func() {
		if err := spend.Refund(); err != nil {
			t.Fatalf("Refund returned error: %v", err)
		}
	})

	afterCheckIn, err := store.ApplyCheckIn(testFingerprint)
	if err != nil {
		t.Fatalf("ApplyCheckIn after spend returned error: %v", err)
	}
	if afterCheckIn.Balance != BalanceCap || !afterCheckIn.SignedToday {
		t.Fatalf("ApplyCheckIn after spend = %+v, want today's sign-in restored to cap", afterCheckIn)
	}
}

func TestSpendRequiresCreditsAndRefunds(t *testing.T) {
	t.Parallel()
	store := newTestStore(t, time.Date(2026, 5, 23, 10, 0, 0, 0, time.Local))

	if _, _, err := store.Spend(testFingerprint); !errors.Is(err, ErrNoCredits) {
		t.Fatalf("Spend without credits error = %v, want ErrNoCredits", err)
	}

	if _, err := store.ApplyCheckIn(testFingerprint); err != nil {
		t.Fatalf("ApplyCheckIn returned error: %v", err)
	}
	spend, afterSpend, err := store.Spend(testFingerprint)
	if err != nil {
		t.Fatalf("Spend returned error: %v", err)
	}
	if afterSpend.Balance != DailyGrant-1 {
		t.Fatalf("Spend balance = %d, want %d", afterSpend.Balance, DailyGrant-1)
	}

	if err := spend.Refund(); err != nil {
		t.Fatalf("Refund returned error: %v", err)
	}
	afterRefund, err := store.Get(testFingerprint)
	if err != nil {
		t.Fatalf("Get after refund returned error: %v", err)
	}
	if afterRefund.Balance != DailyGrant {
		t.Fatalf("Refund balance = %d, want %d", afterRefund.Balance, DailyGrant)
	}
}

func newTestStore(t *testing.T, now time.Time) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "quota.db"))
	if err != nil {
		t.Fatalf("Open test store: %v", err)
	}
	store.now = func() time.Time { return now }
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close test store: %v", err)
		}
	})
	return store
}

func writeTestAccount(t *testing.T, store *Store, fingerprint string, acct account) {
	t.Helper()
	if err := store.db.Update(func(tx *bbolt.Tx) error {
		return writeAccount(tx.Bucket(bucketAccounts), fingerprint, acct)
	}); err != nil {
		t.Fatalf("write test account: %v", err)
	}
}
