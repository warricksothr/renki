package obscapture

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/coder/websocket"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func mockOBS(t *testing.T, authentication bool, f func(context.Context, *websocket.Conn)) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		hello := map[string]any{"rpcVersion": 1}
		if authentication {
			hello["authentication"] = map[string]any{"salt": "salt", "challenge": "challenge"}
		}
		b, _ := json.Marshal(map[string]any{"op": 0, "d": hello})
		if c.Write(ctx, websocket.MessageText, b) != nil {
			return
		}
		f(ctx, c)
	}))
	t.Cleanup(server.Close)
	return "ws" + strings.TrimPrefix(server.URL, "http")
}

func TestOBSAuthenticationAndRequestErrorsStayPrivate(t *testing.T) {
	address := mockOBS(t, true, func(ctx context.Context, c *websocket.Conn) {
		_, b, err := c.Read(ctx)
		if err != nil {
			return
		}
		if strings.Contains(string(b), "private-password") {
			t.Error("Plaintext password sent")
		}
		var identify struct {
			Op int
			D  struct {
				Authentication string
				RPCVersion     int `json:"rpcVersion"`
				Events         int `json:"eventSubscriptions"`
			}
		}
		if json.Unmarshal(b, &identify) != nil || identify.Op != 1 || identify.D.Authentication == "" || identify.D.RPCVersion != 1 || identify.D.Events != 0 {
			t.Error("Bad identify")
		}
		c.Write(ctx, websocket.MessageText, []byte(`{"op":2,"d":{"negotiatedRpcVersion":1}}`))
		_, b, err = c.Read(ctx)
		if err != nil {
			return
		}
		var request struct {
			D struct {
				ID   string `json:"requestId"`
				Kind string `json:"requestType"`
			}
		}
		json.Unmarshal(b, &request)
		if request.D.Kind != "GetStats" {
			t.Error("Wrong request")
		}
		answer, _ := json.Marshal(map[string]any{"op": 7, "d": map[string]any{"requestId": request.D.ID, "requestStatus": map[string]any{"result": false, "comment": "private-password"}}})
		c.Write(ctx, websocket.MessageText, answer)
	})
	o, err := connect(context.Background(), address, "private-password")
	if err != nil {
		t.Fatal(err)
	}
	defer o.Close()
	err = o.Call(context.Background(), "GetStats", nil, nil)
	if err == nil || strings.Contains(err.Error(), "private-password") {
		t.Fatalf("Unredacted request failure: %v", err)
	}
}

func TestOBSRefusesUnauthenticatedServerAndStalledHandshake(t *testing.T) {
	address := mockOBS(t, false, func(ctx context.Context, c *websocket.Conn) { _, _, _ = c.Read(ctx) })
	if o, err := connect(context.Background(), address, "password"); err == nil {
		o.Close()
		t.Fatal("Unauthenticated control accepted")
	}
	address = mockOBS(t, true, func(ctx context.Context, c *websocket.Conn) { <-ctx.Done() })
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	started := time.Now()
	if o, err := connect(ctx, address, "password"); err == nil {
		o.Close()
		t.Fatal("Stalled handshake accepted")
	}
	if time.Since(started) > time.Second {
		t.Fatal("Handshake deadline ignored")
	}
}

// fakeOBS authenticates any client, then answers each request with handle.
// A nil response is sent as an empty object.
func fakeOBS(t *testing.T, handle func(kind string, data json.RawMessage) (bool, any)) *OBS {
	t.Helper()
	address := mockOBS(t, true, func(ctx context.Context, c *websocket.Conn) {
		if _, _, err := c.Read(ctx); err != nil {
			return
		}
		c.Write(ctx, websocket.MessageText, []byte(`{"op":2,"d":{"negotiatedRpcVersion":1}}`))
		for {
			_, b, err := c.Read(ctx)
			if err != nil {
				return
			}
			var request struct {
				D struct {
					ID   string          `json:"requestId"`
					Kind string          `json:"requestType"`
					Data json.RawMessage `json:"requestData"`
				}
			}
			json.Unmarshal(b, &request)
			ok, response := handle(request.D.Kind, request.D.Data)
			if response == nil {
				response = map[string]any{}
			}
			answer, _ := json.Marshal(map[string]any{"op": 7, "d": map[string]any{"requestId": request.D.ID, "requestStatus": map[string]any{"result": ok}, "responseData": response}})
			if c.Write(ctx, websocket.MessageText, answer) != nil {
				return
			}
		}
	})
	o, err := connect(context.Background(), address, "password")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(o.Close)
	return o
}

// captureState is the game source as a fake OBS holds it.
type captureState struct {
	mu       sync.Mutex
	settings map[string]any
	sets     []map[string]any
}

func newCaptureState(window string) *captureState {
	return &captureState{settings: map[string]any{"capture_mode": "window", "window": window, "priority": 0}}
}

// input answers the source requests and reports whether it handled kind.
func (s *captureState) input(kind string, data json.RawMessage) (handled, ok bool, response any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch kind {
	case "GetInputSettings":
		settings := map[string]any{}
		for k, v := range s.settings {
			settings[k] = v
		}
		return true, true, map[string]any{"inputKind": "game_capture", "inputSettings": settings}
	case "SetInputSettings":
		var args struct {
			InputName     string         `json:"inputName"`
			Overlay       bool           `json:"overlay"`
			InputSettings map[string]any `json:"inputSettings"`
		}
		json.Unmarshal(data, &args)
		if args.InputName != Source {
			return true, true, nil
		}
		if !args.Overlay {
			s.settings = map[string]any{}
		}
		for k, v := range args.InputSettings {
			s.settings[k] = v
		}
		s.sets = append(s.sets, args.InputSettings)
		return true, true, nil
	}
	return false, false, nil
}

func (s *captureState) window() any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings["window"]
}

func TestSourceLossTrialRestoresWindowAfterCancel(t *testing.T) {
	state := newCaptureState(Window)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	o := fakeOBS(t, func(kind string, data json.RawMessage) (bool, any) {
		if handled, ok, response := state.input(kind, data); handled {
			return ok, response
		}
		if kind == "GetCurrentProgramScene" {
			// The interrupt arrives while standby is still pending.
			cancel()
			return true, map[string]any{"currentProgramSceneName": Gameplay}
		}
		return false, nil
	})
	err := o.SourceLossTrial(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Cancelled trial reported %v", err)
	}
	if state.window() != Window {
		t.Fatalf("Window left on %v", state.window())
	}
}

func TestSourceLossTrialFailsWhenRestoreFails(t *testing.T) {
	state := newCaptureState(Window)
	o := fakeOBS(t, func(kind string, data json.RawMessage) (bool, any) {
		if kind == "SetInputSettings" && strings.Contains(string(data), `"window":"`+Window+`"`) {
			return false, nil
		}
		if handled, ok, response := state.input(kind, data); handled {
			return ok, response
		}
		if kind == "GetCurrentProgramScene" {
			return true, map[string]any{"currentProgramSceneName": Offline}
		}
		return false, nil
	})
	err := o.SourceLossTrial(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Capture window not restored") {
		t.Fatalf("Refused restore reported %v", err)
	}
}

func TestConfigureRepairsExistingSourceSettings(t *testing.T) {
	state := newCaptureState("Renki nonexistent window:UnityWndClass:valheim.exe")
	state.settings["priority"] = 2
	o := fakeOBS(t, func(kind string, data json.RawMessage) (bool, any) {
		if handled, ok, response := state.input(kind, data); handled {
			return ok, response
		}
		switch kind {
		case "GetSceneList":
			return true, map[string]any{"scenes": []any{map[string]any{"sceneName": Offline}, map[string]any{"sceneName": Gameplay}}}
		case "GetCurrentProgramScene":
			return true, map[string]any{"currentProgramSceneName": Offline}
		case "GetInputList":
			return true, map[string]any{"inputs": []any{map[string]any{"inputName": "Renki Standby"}, map[string]any{"inputName": Source}}}
		case "GetSceneItemList":
			var args struct {
				SceneName string `json:"sceneName"`
			}
			json.Unmarshal(data, &args)
			name := "Renki Standby"
			if args.SceneName == Gameplay {
				name = Source
			}
			return true, map[string]any{"sceneItems": []any{map[string]any{"sourceName": name, "sceneItemEnabled": true}}}
		case "CreateScene", "CreateInput":
			t.Errorf("Configure recreated existing %s", kind)
			return false, nil
		}
		return true, nil
	})
	if err := o.Configure(context.Background(), "slate.html"); err != nil {
		t.Fatalf("Configure did not repair the source: %v", err)
	}
	if state.window() != Window || state.settings["priority"] != float64(0) {
		t.Fatalf("Source left with %v", state.settings)
	}
}

func TestSourceLossTrialInterruptedBeforeLossChangesNothing(t *testing.T) {
	state := newCaptureState(Window)
	ctx, cancel := context.WithCancel(context.Background())
	o := fakeOBS(t, func(kind string, data json.RawMessage) (bool, any) {
		if kind == "GetInputSettings" {
			cancel()
		}
		if handled, ok, response := state.input(kind, data); handled {
			return ok, response
		}
		return false, nil
	})
	if err := o.SourceLossTrial(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Interrupted trial reported %v", err)
	}
	if len(state.sets) != 0 {
		t.Fatalf("Interrupted trial changed the source: %v", state.sets)
	}
}

func TestSourceLossTrialFailsWhenRestoreDoesNotVerify(t *testing.T) {
	state := newCaptureState(Window)
	restored := false
	o := fakeOBS(t, func(kind string, data json.RawMessage) (bool, any) {
		if kind == "SetInputSettings" && strings.Contains(string(data), `"window":"`+Window+`"`) {
			// OBS accepts the restore but keeps the wrong window.
			restored = true
			return true, nil
		}
		if kind == "GetInputSettings" && restored {
			return true, map[string]any{"inputKind": "game_capture", "inputSettings": map[string]any{"capture_mode": "window", "window": "other", "priority": 0}}
		}
		if handled, ok, response := state.input(kind, data); handled {
			return ok, response
		}
		if kind == "GetCurrentProgramScene" {
			return true, map[string]any{"currentProgramSceneName": Offline}
		}
		return false, nil
	})
	err := o.SourceLossTrial(context.Background())
	if err == nil || !strings.Contains(err.Error(), "run configure") {
		t.Fatalf("Unverified restore reported %v", err)
	}
}

func TestLocalStartupRetainsExistingPrivateOutputAndRefusesOtherOutput(t *testing.T) {
	for _, server := range []string{"rtmp://127.0.0.1:11935/renki", "rtmp://example.invalid/live"} {
		t.Run(server, func(t *testing.T) {
			o := fakeOBS(t, func(kind string, data json.RawMessage) (bool, any) {
				switch kind {
				case "GetStreamStatus":
					return true, map[string]any{"outputActive": true}
				case "GetStreamServiceSettings":
					return true, map[string]any{"streamServiceSettings": map[string]any{"server": server}}
				case "GetSceneItemList":
					var req struct {
						Scene string `json:"sceneName"`
					}
					json.Unmarshal(data, &req)
					source := "Renki Standby"
					if req.Scene == Gameplay {
						source = Source
					}
					return true, map[string]any{"sceneItems": []any{map[string]any{"sourceName": source, "sceneItemEnabled": true}}}
				default:
					t.Errorf("existing output mutated: %s", kind)
					return false, nil
				}
			})
			err := o.StartLocal(context.Background())
			if (err == nil) != (server == "rtmp://127.0.0.1:11935/renki") {
				t.Fatalf("wrong result: %v", err)
			}
		})
	}
}

func TestLocalStartupSelectsStandbyAndFixedDestinationBeforeStarting(t *testing.T) {
	var mutations []string
	o := fakeOBS(t, func(kind string, data json.RawMessage) (bool, any) {
		switch kind {
		case "GetStreamStatus":
			return true, map[string]any{"outputActive": false}
		case "GetSceneItemList":
			var req struct {
				Scene string `json:"sceneName"`
			}
			json.Unmarshal(data, &req)
			source := "Renki Standby"
			if req.Scene == Gameplay {
				source = Source
			}
			return true, map[string]any{"sceneItems": []any{map[string]any{"sourceName": source, "sceneItemEnabled": true}}}
		case "GetCurrentProgramScene", "GetStudioModeEnabled":
			return true, nil
		case "SetCurrentProgramScene":
			if !strings.Contains(string(data), Offline) {
				t.Error("startup did not choose standby")
			}
		case "SetStreamServiceSettings":
			if !strings.Contains(string(data), "rtmp://127.0.0.1:11935/renki") {
				t.Error("wrong destination")
			}
		case "StartStream":
		default:
			t.Errorf("unexpected call %s", kind)
		}
		if strings.HasPrefix(kind, "Set") || kind == "StartStream" {
			mutations = append(mutations, kind)
		}
		return true, nil
	})
	if err := o.StartLocal(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Join(mutations, ",") != "SetCurrentProgramScene,SetStreamServiceSettings,StartStream" {
		t.Fatalf("unsafe ordering: %v", mutations)
	}
}
