// Package obscapture verifies opt-in game-rendered nonces in an exact OBS
// source. A moving compositor, a cached screenshot and source activity cannot
// substitute for fresh game pixels.
package obscapture

import (
	"encoding/hex"
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

type Status struct {
	Connected  bool                   `json:"connected"`
	Phase      string                 `json:"phase"`
	Connection uint64                 `json:"connection"`
	WorldUID   string                 `json:"worldUid"`
	Proof      *protocol.CaptureProof `json:"captureProof"`
}

func (s Status) Scope() string { return fmt.Sprintf("%d:%s", s.Connection, s.WorldUID) }

// Decode samples only the nonce strip in an unscaled 1280x720 source snapshot.
// Midtones are rejected instead of guessed; compression or resizing may deny
// playback but cannot turn a slate into a valid nonce.
func Decode(img image.Image) (string, bool) {
	if img.Bounds() != image.Rect(0, 0, 1280, 720) {
		return "", false
	}
	var out strings.Builder
	for cell := 0; cell < 80; cell += 4 {
		value := 0
		for bit := 0; bit < 4; bit++ {
			var selected int = -1
			for _, y := range []int{6, 8, 10} {
				r, g, b, _ := img.At(9+(cell+bit)*3, y).RGBA()
				v := (r + g + b) / (3 * 257)
				current := 0
				if v > 210 {
					current = 1
				} else if v >= 45 {
					return "", false
				}
				if selected >= 0 && current != selected {
					return "", false
				}
				selected = current
			}
			value = value<<1 | selected
		}
		out.WriteByte("0123456789abcdef"[value])
	}
	result := out.String()
	return result[4:], strings.HasPrefix(result, "d39a")
}

type Gate struct {
	scope, token string
	count        int
}

func (g *Gate) Reset() { *g = Gate{} }

// Accept requires stable connection/world, current telemetry and distinct
// observed nonces. Replaying the previous accepted image revokes trust.
func (g *Gate) Accept(before, after Status, token string, now time.Time) bool {
	valid := func(s Status) bool {
		if !s.Connected || s.Phase != "in_world" || s.WorldUID == "" || s.Connection == 0 || s.Proof == nil {
			return false
		}
		age := now.Sub(s.Proof.At)
		decoded, err := hex.DecodeString(s.Proof.Token)
		return err == nil && len(decoded) == 8 && age >= 0 && age <= 2*time.Second
	}
	if !valid(before) || !valid(after) || before.Scope() != after.Scope() ||
		(token != before.Proof.Token && token != after.Proof.Token) || token == g.token {
		g.Reset()
		return false
	}
	if g.scope != after.Scope() {
		g.Reset()
		g.scope = after.Scope()
	}
	g.token = token
	g.count++
	return g.count >= 2
}
