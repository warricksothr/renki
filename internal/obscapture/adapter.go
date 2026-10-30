package obscapture

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image/png"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/video"
)

type Config struct {
	OBSPassword      string `json:"obsPassword"`
	ControllerConfig string `json:"controllerConfig"`
	HealthFile       string `json:"healthFile"`
	Slate            string `json:"slate"`
}

var ErrWarmup = errors.New("Waiting for distinct fresh game frames")

func localClient() *http.Client {
	return &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func ReadConfig(path string) (Config, error) {
	var c Config
	b, err := os.ReadFile(path)
	if err != nil {
		return c, errors.New("Private adapter config missing")
	}
	b = bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf})
	if json.Unmarshal(b, &c) != nil || c.OBSPassword == "" || !filepath.IsAbs(c.HealthFile) || c.ControllerConfig == "" || c.Slate == "" {
		return c, errors.New("Private adapter config invalid")
	}
	return c, nil
}

func status(ctx context.Context, c Config) (Status, error) {
	var settings struct {
		Panel struct{ Listen, Token string }
	}
	b, err := os.ReadFile(c.ControllerConfig)
	if err != nil {
		return Status{}, errors.New("Controller configuration unavailable")
	}
	if json.Unmarshal(bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf}), &settings) != nil || settings.Panel.Listen == "" || settings.Panel.Token == "" {
		return Status{}, errors.New("Controller configuration invalid")
	}
	address, err := localAddress(settings.Panel.Listen)
	if err != nil {
		return Status{}, err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "http://"+address+"/api/status", nil)
	if err != nil {
		return Status{}, errors.New("Controller address invalid")
	}
	req.Header.Set("Authorization", "Bearer "+settings.Panel.Token)
	res, err := localClient().Do(req)
	if err != nil {
		return Status{}, errors.New("Controller unavailable")
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	var s Status
	if err != nil || res.StatusCode != 200 || json.Unmarshal(data, &s) != nil {
		return s, errors.New("Controller status invalid")
	}
	return s, nil
}

// The panel credential may only be sent to a literal IP belonging to this VM.
// No DNS lookup, configured URL/path, redirect or proxy can move it elsewhere.
func localAddress(listen string) (string, error) {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return "", errors.New("Controller address invalid")
	}
	ip, err := netip.ParseAddr(host)
	p, e := strconv.Atoi(port)
	if err != nil || e != nil || p < 1 || p > 65535 {
		return "", errors.New("Controller address invalid")
	}
	if ip.IsUnspecified() {
		return net.JoinHostPort("127.0.0.1", port), nil
	}
	if ip.IsLoopback() {
		return listen, nil
	}
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return "", errors.New("Local addresses unavailable")
	}
	for _, address := range addresses {
		prefix, e := netip.ParsePrefix(address.String())
		if e == nil && prefix.Addr().Unmap() == ip.Unmap() {
			return listen, nil
		}
	}
	return "", errors.New("Controller address is not local")
}

func Atomic(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".capture-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, path)
}

func probe(ctx context.Context, o *OBS, c Config, g *Gate) (video.Health, error) {
	before, err := status(ctx, c)
	if err != nil {
		return video.Health{}, err
	}
	if !before.Connected || before.Phase != "in_world" || before.Proof == nil {
		return video.Health{}, errors.New("Tas is resting or capture beacon is unavailable")
	}
	if err = o.VerifySource(ctx); err != nil {
		return video.Health{}, err
	}
	if err = o.VerifyScenes(ctx); err != nil {
		return video.Health{}, err
	}
	var shot struct {
		ImageData string `json:"imageData"`
	}
	if err = o.Call(ctx, "GetSourceScreenshot", map[string]any{"sourceName": Source, "imageFormat": "png", "imageWidth": 1280, "imageHeight": 720}, &shot); err != nil {
		return video.Health{}, err
	}
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(shot.ImageData, prefix) || len(shot.ImageData) > 3<<20 {
		return video.Health{}, errors.New("Capture snapshot invalid")
	}
	b, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(shot.ImageData, prefix))
	if err != nil {
		return video.Health{}, errors.New("Capture snapshot invalid")
	}
	config, err := png.DecodeConfig(bytes.NewReader(b))
	if err != nil || config.Width != 1280 || config.Height != 720 {
		return video.Health{}, errors.New("Capture dimensions invalid")
	}
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return video.Health{}, errors.New("Capture snapshot invalid")
	}
	token, ok := Decode(img)
	if !ok {
		return video.Health{}, errors.New("Game capture beacon not visible")
	}
	after, err := status(ctx, c)
	if err != nil {
		return video.Health{}, err
	}
	if !g.Accept(before, after, token, time.Now()) {
		return video.Health{}, ErrWarmup
	}
	return video.Health{Gameplay: true, Connection: after.Connection, WorldUID: after.WorldUID, ObservedAt: time.Now()}, nil
}

// Run owns no game process or output lifecycle. The independent watchdog
// revokes Program scene if this process dies; panel assertions also expire.
func Run(ctx context.Context, c Config, watchdog bool) error {
	var o *OBS
	defer func() {
		Atomic(c.HealthFile, video.Health{})
		if o != nil {
			_ = o.Scene(context.Background(), Offline)
			o.Close()
		}
	}()
	var gate Gate
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for ctx.Err() == nil {
		if o == nil {
			var err error
			o, err = Connect(ctx, c.OBSPassword)
			if err == nil {
				err = o.Scene(ctx, Offline)
			}
			if err != nil {
				if o != nil {
					o.Close()
					o = nil
				}
				Atomic(c.HealthFile, video.Health{})
				gate.Reset()
			}
		}
		if o != nil {
			if watchdog {
				var h video.Health
				b, err := os.ReadFile(c.HealthFile)
				if err != nil || json.Unmarshal(b, &h) != nil || !h.Gameplay || time.Since(h.ObservedAt) < 0 || time.Since(h.ObservedAt) > 3*time.Second {
					if o.Scene(ctx, Offline) != nil {
						o.Close()
						o = nil
					}
				}
			} else {
				h, err := probe(ctx, o, c, &gate)
				reason := "Verified changing game frames"
				if err != nil {
					h = video.Health{}
					if !errors.Is(err, ErrWarmup) {
						gate.Reset()
					}
					reason = err.Error()
				}
				scene := Offline
				if h.Gameplay {
					scene = Gameplay
				}
				if o.Scene(ctx, scene) != nil {
					h = video.Health{}
					gate.Reset()
					reason = "OBS scene unavailable"
					o.Close()
					o = nil
				}
				if Atomic(c.HealthFile, h) != nil {
					return errors.New("Cannot write capture health")
				}
				_ = Atomic(filepath.Join(filepath.Dir(c.HealthFile), "status.json"), map[string]any{"gameplay": h.Gameplay, "reason": reason, "observedAt": time.Now()})
			}
		}
		select {
		case <-ctx.Done():
		case <-ticker.C:
		}
	}
	return nil
}
