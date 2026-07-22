package poll

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

const testFingerprint = "9a3f2c1234567890d8e1"

func TestVoteCountsEachFingerprintOnce(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)

	first, err := store.Vote(context.Background(), VoteRequest{Fingerprint: testFingerprint})
	if err != nil {
		t.Fatalf("first Vote returned error: %v", err)
	}
	if first.Count != 1 || !first.Recorded {
		t.Fatalf("first Vote = %+v, want one recorded vote", first)
	}

	second, err := store.Vote(context.Background(), VoteRequest{Fingerprint: testFingerprint})
	if err != nil {
		t.Fatalf("second Vote returned error: %v", err)
	}
	if second.Count != 1 || second.Recorded {
		t.Fatalf("second Vote = %+v, want unchanged duplicate vote", second)
	}
}

func TestStatsReturnsSupportCount(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	for _, fingerprint := range []string{testFingerprint, "anotherFingerprint123"} {
		if _, err := store.Vote(context.Background(), VoteRequest{Fingerprint: fingerprint}); err != nil {
			t.Fatalf("Vote(%q) returned error: %v", fingerprint, err)
		}
	}

	got, err := store.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats returned error: %v", err)
	}
	if got.Count != 2 {
		t.Fatalf("Stats count = %d, want 2", got.Count)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "poll.db"))
	if err != nil {
		t.Fatalf("Open test store: %v", err)
	}
	store.now = func() time.Time {
		return time.Date(2026, 7, 22, 23, 0, 0, 0, time.Local)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close test store: %v", err)
		}
	})
	return store
}
