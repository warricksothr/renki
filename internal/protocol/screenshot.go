package protocol

import (
	"bytes"
	"errors"
	"image/jpeg"
	"math"
	"time"
)

const TypeScreenshot = "screenshot"
const MaxScreenshotBytes = 512 * 1024

// ScreenshotConfig is independent of body-task permissions. The plugin also
// clamps every capture start to at most once per second.
type ScreenshotConfig struct {
	Enabled   bool    `json:"enabled"`
	IntervalS float64 `json:"intervalS"`
	Width     int     `json:"width"`
	Quality   int     `json:"quality"`
}

func DefaultScreenshotConfig() ScreenshotConfig {
	return ScreenshotConfig{IntervalS: 5, Width: 960, Quality: 70}
}
func (c ScreenshotConfig) Validate() error {
	if math.IsNaN(c.IntervalS) || math.IsInf(c.IntervalS, 0) || c.IntervalS < 1 || c.IntervalS > 300 || c.Width < 320 || c.Width > 1280 || c.Quality < 30 || c.Quality > 90 {
		return errors.New("screenshot requires intervalS 1–300, width 320–1280 and quality 30–90")
	}
	return nil
}

// Screenshot is visual context, never a native task success receipt.
type Screenshot struct {
	Envelope
	World        string    `json:"world"`
	WorldUID     string    `json:"worldUid"`
	CapturedAt   time.Time `json:"capturedAt"`
	Pos          Vec3      `json:"pos"`
	CameraPos    Vec3      `json:"cameraPos"`
	CameraAngles Vec3      `json:"cameraAngles"`
	Task         string    `json:"task,omitempty"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	CaptureMS    float64   `json:"captureMs"`
	JPEG         []byte    `json:"jpeg,omitempty"`
}

func (*Screenshot) MessageType() string { return TypeScreenshot }
func (s Screenshot) Validate() error {
	if s.World == "" || s.WorldUID == "" || s.CapturedAt.IsZero() || len(s.JPEG) == 0 || len(s.JPEG) > MaxScreenshotBytes || s.Width < 1 || s.Width > 1280 || s.Height < 1 || s.Height > 720 || math.IsNaN(s.CaptureMS) || math.IsInf(s.CaptureMS, 0) || s.CaptureMS < 0 {
		return errors.New("invalid screenshot bounds or metadata")
	}
	for _, p := range []Vec3{s.Pos, s.CameraPos, s.CameraAngles} {
		for _, v := range p {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return errors.New("invalid screenshot pose")
			}
		}
	}
	c, err := jpeg.DecodeConfig(bytes.NewReader(s.JPEG))
	if err != nil || c.Width != s.Width || c.Height != s.Height {
		return errors.New("screenshot is not a matching bounded JPEG")
	}
	return nil
}
