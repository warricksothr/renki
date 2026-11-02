package obscapture

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/warricksothr/renki/internal/video"
)

func TestCaptureHealthWriteFailureStaysContainedAndRetries(t *testing.T) {
	for _, persistent := range []bool{false, true} {
		t.Run(map[bool]string{false: "transient", true: "persistent"}[persistent], func(t *testing.T) {
			var mu sync.Mutex
			var scenes []string
			address := mockOBS(t, true, func(ctx context.Context, c *websocket.Conn) {
				if _, _, err := c.Read(ctx); err != nil {
					return
				}
				if c.Write(ctx, websocket.MessageText, []byte(`{"op":2,"d":{"negotiatedRpcVersion":1}}`)) != nil {
					return
				}
				for {
					_, raw, err := c.Read(ctx)
					if err != nil {
						return
					}
					var req struct {
						D struct {
							RequestType string
							RequestID   string
							RequestData struct{ SceneName string }
						}
					}
					if json.Unmarshal(raw, &req) != nil {
						return
					}
					data := map[string]any{}
					switch req.D.RequestType {
					case "GetCurrentProgramScene":
						data["currentProgramSceneName"] = "different"
					case "GetStudioModeEnabled":
						data["studioModeEnabled"] = false
					case "SetCurrentProgramScene":
						mu.Lock()
						scenes = append(scenes, req.D.RequestData.SceneName)
						mu.Unlock()
					}
					out, _ := json.Marshal(map[string]any{"op": 7, "d": map[string]any{"requestId": req.D.RequestID, "requestStatus": map[string]any{"result": true, "code": 100}, "responseData": data}})
					if c.Write(ctx, websocket.MessageText, out) != nil {
						return
					}
				}
			})
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			attempts, published, failures := 0, 0, 0
			config := Config{OBSPassword: "private-password", HealthFile: filepath.Join(t.TempDir(), "health.json")}
			deps := runDependencies{
				connect: func(ctx context.Context, password string) (*OBS, error) { return connect(ctx, address, password) },
				probe: func(context.Context, *OBS, Config, *Gate) (video.Health, error) {
					return video.Health{Gameplay: true, Connection: 1, WorldUID: "world", ObservedAt: time.Now()}, nil
				},
				interval: time.Millisecond,
				write: func(path string, value any) error {
					if path == config.HealthFile {
						h := value.(video.Health)
						if h.Gameplay {
							attempts++
							if attempts == 1 || persistent {
								return errors.New("simulated Windows sharing violation")
							}
							published++
						}
					} else if state, ok := value.(map[string]any); ok {
						if state["reason"] == "Cannot write capture health; retrying" {
							if state["gameplay"] != false {
								t.Error("failed publication reported gameplay")
							}
							failures++
							mu.Lock()
							last := scenes[len(scenes)-1]
							mu.Unlock()
							if last != Offline {
								t.Error("failed publication did not revoke Program")
							}
						}
						if attempts >= 3 {
							cancel()
						}
					}
					return Atomic(path, value)
				},
			}
			if err := run(ctx, config, false, deps); err != nil {
				t.Fatal("verifier terminated on recoverable publication failure:", err)
			}
			if attempts < 3 || failures == 0 {
				t.Fatalf("attempts=%d failures=%d", attempts, failures)
			}
			if persistent && published != 0 || !persistent && published == 0 {
				t.Fatalf("published=%d", published)
			}
		})
	}
}
