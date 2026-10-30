package obscapture

import (
	"github.com/warricksothr/renki/internal/protocol"
	"image"
	"image/color"
	"testing"
	"time"
)

func marked(token string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 1280, 720))
	bits := "d39a" + token
	for i := 0; i < 80; i++ {
		var v byte
		c := bits[i/4]
		if c >= 'a' {
			v = c - 'a' + 10
		} else {
			v = c - '0'
		}
		if v&(1<<uint(3-i%4)) != 0 {
			for x := 8 + i*3; x < 11+i*3; x++ {
				for y := 4; y < 12; y++ {
					img.Set(x, y, color.White)
				}
			}
		}
	}
	return img
}

func TestDecodeRejectsSlateCorruptionAndResizing(t *testing.T) {
	const token = "cafe123409abcdef"
	img := marked(token)
	if got, ok := Decode(img); !ok || got != token {
		t.Fatalf("decode %q %v", got, ok)
	}
	if _, ok := Decode(image.NewRGBA(img.Bounds())); ok {
		t.Fatal("blank trusted")
	}
	img.Set(9, 8, color.RGBA{128, 128, 128, 255})
	if _, ok := Decode(img); ok {
		t.Fatal("corruption trusted")
	}
	if _, ok := Decode(image.NewRGBA(image.Rect(0, 0, 1920, 1080))); ok {
		t.Fatal("unverified resize trusted")
	}
}

func TestGateRequiresProgressAndStableFreshScope(t *testing.T) {
	now := time.Now()
	state := func(token string) Status {
		return Status{Connected: true, Phase: "in_world", Connection: 7, WorldUID: "42", Proof: &protocol.CaptureProof{Token: token, At: now.Add(-time.Second)}}
	}
	a, b := state("1234567890abcdef"), state("abcdef1234567890")
	var g Gate
	if g.Accept(a, a, a.Proof.Token, now) {
		t.Fatal("single image trusted")
	}
	if !g.Accept(b, b, b.Proof.Token, now) {
		t.Fatal("progress not trusted")
	}
	if g.Accept(b, b, b.Proof.Token, now) {
		t.Fatal("frozen frame trusted")
	}
	for _, change := range []func(*Status){
		func(s *Status) { s.Connected = false }, func(s *Status) { s.Phase = "menu" },
		func(s *Status) { s.Connection++ }, func(s *Status) { s.WorldUID = "43" },
		func(s *Status) { s.Proof = &protocol.CaptureProof{Token: s.Proof.Token, At: now.Add(-3 * time.Second)} },
		func(s *Status) { s.Proof = &protocol.CaptureProof{Token: s.Proof.Token, At: now.Add(time.Second)} },
	} {
		g.Reset()
		g.Accept(a, a, a.Proof.Token, now)
		changed := b
		change(&changed)
		if g.Accept(b, changed, b.Proof.Token, now) {
			t.Fatal("uncertain scope trusted")
		}
	}
	g.Reset()
	g.Accept(a, a, a.Proof.Token, now)
	if g.Accept(b, b, "0000000000000000", now) {
		t.Fatal("unmatched capture trusted")
	}
}

func TestControllerCredentialStaysLocal(t *testing.T) {
	for _, address := range []string{"example.com:8780", "https://127.0.0.1:8780/", "203.0.113.17:8780", "127.0.0.1:0", "127.0.0.1:99999"} {
		if _, err := localAddress(address); err == nil {
			t.Fatalf("Nonlocal/invalid address accepted: %s", address)
		}
	}
	for _, address := range []string{"127.0.0.1:8780", "[::1]:8780", "0.0.0.0:8780"} {
		if _, err := localAddress(address); err != nil {
			t.Fatalf("Local address refused: %s", address)
		}
	}
}
