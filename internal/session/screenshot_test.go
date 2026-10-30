package session

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestScreenshotsValidateAndNeverCrossSessionOrWorld(t *testing.T) {
	s, _, p := setup(t, nil, nil)
	p.Send(&protocol.State{World: "one", WorldUID: "uid", Phase: protocol.PhaseInWorld})
	waitFor(t, func() bool { return s.State() != nil })
	c := currentConn(t, s)
	var b bytes.Buffer
	if err := jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, 8, 4)), nil); err != nil {
		t.Fatal(err)
	}
	shot := &protocol.Screenshot{Envelope: protocol.Envelope{Seq: 100}, World: "one", WorldUID: "uid", CapturedAt: time.Now(), Width: 8, Height: 4, JPEG: b.Bytes()}
	s.Message(c, shot)
	if got := s.Screenshot(); got == nil || got.Seq != 100 {
		t.Fatal("valid frame absent", got)
	}
	// Returned buffers do not mutate shared evidence.
	got := s.Screenshot()
	got.JPEG[0] = 0
	if s.Screenshot().JPEG[0] != 0xff {
		t.Fatal("mutable image")
	}
	shot.Seq = 101
	s.Message(c, shot)
	if s.Screenshot().Seq != 100 {
		t.Fatal("receive rate limit bypassed")
	}
	s.ClearScreenshot()
	shot.WorldUID = "other"
	s.Message(c, shot)
	if s.Screenshot() != nil {
		t.Fatal("world identity ignored")
	}
	shot.WorldUID = "uid"
	shot.Height = 5
	s.Message(c, shot)
	if s.Screenshot() != nil {
		t.Fatal("invalid JPEG dimensions accepted")
	}
	shot.Height = 4
	s.Message(c, shot)
	s.Message(c, &protocol.State{Envelope: protocol.Envelope{Seq: 200}, Phase: protocol.PhaseMenu})
	if s.Screenshot() != nil {
		t.Fatal("menu kept old image")
	}
	s.Message(c, &protocol.State{Envelope: protocol.Envelope{Seq: 201}, World: "one", WorldUID: "uid", Phase: protocol.PhaseInWorld})
	s.Message(c, shot)
	s.Disconnected(c, nil)
	if s.Screenshot() != nil {
		t.Fatal("disconnect kept image")
	}
}
