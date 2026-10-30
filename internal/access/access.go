// Package access resolves player-specific command and routine permissions.
package access

import (
	"errors"
	"slices"
	"sort"
	"strings"
)

type Permission string

const (
	Command Permission = "command"
	Locate  Permission = "locate"
	Greet   Permission = "greet"
)

// Rules assigns named control profiles. Ownership is an independent admin grant.
type Rules struct {
	Profiles map[string]string `json:"profiles"`
}

func (r Rules) Validate() error {
	if len(r.Profiles) > 128 {
		return errors.New("players.profiles: at most 128 assignments")
	}
	for name, profile := range r.Profiles {
		if strings.TrimSpace(name) != name || name == "" || len(name) > 64 || strings.ContainsAny(name, "\r\n\t") {
			return errors.New("players.profiles: invalid player name")
		}
		switch profile {
		case "operator", "companion", "guest":
		default:
			return errors.New("players.profiles: profile must be operator, companion or guest")
		}
	}
	return nil
}
func (r Rules) Has(owners []string, name string, permission Permission) bool {
	if slices.Contains(owners, name) {
		return true
	}
	switch r.Profiles[name] {
	case "operator":
		return permission == Command || permission == Locate || permission == Greet
	case "companion":
		return permission == Locate || permission == Greet
	case "guest":
		return permission == Locate
	}
	return false
}

// Commanders adapts existing chat authorization without granting administration.
func (r Rules) Commanders(owners []string) []string {
	out := slices.Clone(owners)
	var extra []string
	for name := range r.Profiles {
		if r.Has(nil, name, Command) && !slices.Contains(out, name) {
			extra = append(extra, name)
		}
	}
	sort.Strings(extra)
	return append(out, extra...)
}
