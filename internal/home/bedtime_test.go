package home

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestBedtimeLimitsPersistAcrossMidnightRestartAndClockRollback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	reserve := func(uid, name string, day, night int, want bool) {
		t.Helper()
		got, err := s.ReserveBedtime(uid, name, day, night)
		if err != nil || got != want {
			t.Fatalf("%s %s day%d night%d: %v %v", uid, name, day, night, got, err)
		}
	}
	reserve("123", "Tas", 213, 213, true)
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	reserve("123", "Tas", 213, 213, false)
	reserve("123", "Tas", 214, 213, false)  // Same night after midnight.
	reserve("123", "Tas", 212, 212, false)  // Clock moved backwards.
	reserve("123", "Tas", 214, 214, true)   // Next evening.
	reserve("124", "Tas", 214, 214, true)   // Different native world.
	reserve("123", "Other", 214, 214, true) // Different character.
	reserve("125", "Tas", 213, 212, true)   // Enabled before dawn.
	reserve("125", "Tas", 213, 213, false)  // Not again on that same calendar day.
	reserve("125", "Tas", 214, 213, true)
}

func TestBedtimeFailedPersistenceDoesNotConsumeAttemptAndScopesNeverEvict(t *testing.T) {
	s := Memory()
	block := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(block, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	s.path = filepath.Join(block, "memory.json")
	if ok, err := s.ReserveBedtime("123", "Tas", 10, 10); ok || err == nil {
		t.Fatal(ok, err)
	}
	s.path = ""
	if ok, err := s.ReserveBedtime("123", "Tas", 10, 10); !ok || err != nil {
		t.Fatal(ok, err)
	}
	for i := 1; i < 64; i++ {
		if ok, err := s.ReserveBedtime(strconv.Itoa(1000+i), "Tas", 10, 10); !ok || err != nil {
			t.Fatal(ok, err)
		}
	}
	if ok, err := s.ReserveBedtime("9999", "Tas", 10, 10); ok || err == nil {
		t.Fatal("scope cap failed", ok, err)
	}
	if ok, err := s.ReserveBedtime("123", "Tas", 10, 10); ok || err != nil {
		t.Fatal("old scope forgotten", ok, err)
	}
}

func TestBedtimeRejectsUnknownIdentityAndInvalidCalendar(t *testing.T) {
	s := Memory()
	for _, tc := range []struct {
		uid, name  string
		day, night int
	}{
		{"", "Tas", 1, 1}, {"0", "Tas", 1, 1}, {"123", "", 1, 1}, {"123", "Tas", 0, 0}, {"123", "Tas", 10, 8}, {"123", "Tas", 10, 11},
	} {
		if ok, err := s.ReserveBedtime(tc.uid, tc.name, tc.day, tc.night); ok || err == nil {
			t.Fatal(tc, ok, err)
		}
	}
}
