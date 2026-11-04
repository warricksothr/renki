package home

import (
	"encoding/json"
	"errors"
	"maps"
	"strconv"
	"strings"
	"unicode"
)

// BedtimeAttempt limits both a calendar day and the night spanning midnight.
type BedtimeAttempt struct {
	Day   int `json:"day"`
	Night int `json:"night"`
}

// ReserveBedtime durably consumes one invitation attempt before it is sent.
// A high-water mark also prevents replay after a backwards clock adjustment.
// No eviction: a full store refuses new scopes instead of forgetting deduplication.
func (s *Store) ReserveBedtime(worldUID, character string, day, night int) (bool, error) {
	uid, err := strconv.ParseInt(worldUID, 10, 64)
	if err != nil || uid == 0 || strconv.FormatInt(uid, 10) != worldUID || day < 1 || day > 1e9 || (night != day && night != day-1) || strings.TrimSpace(character) == "" || len(character) > 128 || strings.ContainsFunc(character, unicode.IsControl) {
		return false, errors.New("bedtime requires a known native world, character and night")
	}
	raw, _ := json.Marshal([]string{worldUID, character})
	key := string(raw)
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.mem.BedtimeNights
	if prior, ok := old[key]; ok && (prior.Day >= day || prior.Night >= night) {
		return false, nil
	}
	if _, ok := old[key]; !ok && len(old) >= 64 {
		return false, errors.New("bedtime invitation scope limit reached")
	}
	next := maps.Clone(old)
	if next == nil {
		next = make(map[string]BedtimeAttempt)
	}
	next[key] = BedtimeAttempt{Day: day, Night: night}
	s.mem.BedtimeNights = next
	if err := s.save(); err != nil {
		s.mem.BedtimeNights = old
		return false, err
	}
	return true, nil
}
