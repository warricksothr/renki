package obscapture

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/coder/websocket"
)

const Offline = "Renki Offline"
const Gameplay = "Renki Gameplay"
const Source = "Renki Game Capture"
const Window = "Valheim:UnityWndClass:valheim.exe"

type OBS struct {
	conn *websocket.Conn
	seq  int
}
type packet struct {
	Op int             `json:"op"`
	D  json.RawMessage `json:"d"`
}

func Connect(ctx context.Context, password string) (*OBS, error) {
	return connect(ctx, "ws://127.0.0.1:4455", password)
}

func connect(ctx context.Context, address, password string) (*OBS, error) {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, address, &websocket.DialOptions{HTTPClient: localClient()})
	if err != nil {
		return nil, errors.New("OBS control unavailable")
	}
	c.SetReadLimit(4 << 20)
	o := &OBS{conn: c}
	failed := true
	defer func() {
		if failed {
			c.CloseNow()
		}
	}()
	p, err := o.read(ctx)
	if err != nil || p.Op != 0 {
		return nil, errors.New("OBS handshake unavailable")
	}
	var hello struct {
		Authentication *struct{ Challenge, Salt string }
	}
	if json.Unmarshal(p.D, &hello) != nil || hello.Authentication == nil || password == "" {
		return nil, errors.New("OBS authentication required")
	}
	digest := sha256.Sum256([]byte(password + hello.Authentication.Salt))
	secret := base64.StdEncoding.EncodeToString(digest[:])
	digest = sha256.Sum256([]byte(secret + hello.Authentication.Challenge))
	if err = o.write(ctx, 1, map[string]any{"rpcVersion": 1, "authentication": base64.StdEncoding.EncodeToString(digest[:]), "eventSubscriptions": 0}); err != nil {
		return nil, errors.New("OBS identify failed")
	}
	p, err = o.read(ctx)
	if err != nil || p.Op != 2 {
		return nil, errors.New("OBS authentication failed")
	}
	failed = false
	return o, nil
}

func (o *OBS) Close() { o.conn.CloseNow() }
func (o *OBS) read(ctx context.Context) (packet, error) {
	_, data, err := o.conn.Read(ctx)
	if err != nil {
		return packet{}, err
	}
	var p packet
	err = json.Unmarshal(data, &p)
	return p, err
}
func (o *OBS) write(ctx context.Context, op int, d any) error {
	data, err := json.Marshal(map[string]any{"op": op, "d": d})
	if err != nil {
		return err
	}
	return o.conn.Write(ctx, websocket.MessageText, data)
}

// Call is serial and private to the adapter: no arbitrary remote request API.
func (o *OBS) Call(ctx context.Context, kind string, args any, out any) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	o.seq++
	id := fmt.Sprint(o.seq)
	if o.write(ctx, 6, map[string]any{"requestType": kind, "requestId": id, "requestData": args}) != nil {
		return errors.New("OBS request unavailable")
	}
	for i := 0; i < 8; i++ {
		p, err := o.read(ctx)
		if err != nil {
			return errors.New("OBS response unavailable")
		}
		if p.Op != 7 {
			continue
		}
		var r struct {
			RequestID     string `json:"requestId"`
			RequestStatus struct {
				Result bool `json:"result"`
			} `json:"requestStatus"`
			ResponseData json.RawMessage `json:"responseData"`
		}
		if json.Unmarshal(p.D, &r) != nil || r.RequestID != id || !r.RequestStatus.Result {
			return errors.New("OBS operation refused")
		}
		if out != nil && json.Unmarshal(r.ResponseData, out) != nil {
			return errors.New("OBS response invalid")
		}
		return nil
	}
	return errors.New("OBS response limit")
}

func (o *OBS) Scene(ctx context.Context, name string) error {
	var current struct {
		CurrentProgramSceneName string `json:"currentProgramSceneName"`
	}
	if err := o.Call(ctx, "GetCurrentProgramScene", nil, &current); err != nil {
		return err
	}
	if current.CurrentProgramSceneName != name {
		if err := o.Call(ctx, "SetCurrentProgramScene", map[string]any{"sceneName": name}, nil); err != nil {
			return err
		}
	}
	var studio struct {
		StudioModeEnabled bool `json:"studioModeEnabled"`
	}
	if err := o.Call(ctx, "GetStudioModeEnabled", nil, &studio); err != nil {
		return err
	}
	if studio.StudioModeEnabled {
		return o.Call(ctx, "SetCurrentPreviewScene", map[string]any{"sceneName": Gameplay}, nil)
	}
	return nil
}

func (o *OBS) VerifySource(ctx context.Context) error {
	var input struct {
		InputKind     string `json:"inputKind"`
		InputSettings struct {
			Mode     string `json:"capture_mode"`
			Window   string `json:"window"`
			Priority int    `json:"priority"`
		} `json:"inputSettings"`
	}
	if o.Call(ctx, "GetInputSettings", map[string]any{"inputName": Source}, &input) != nil || input.InputKind != "game_capture" || input.InputSettings.Mode != "window" || input.InputSettings.Window != Window || input.InputSettings.Priority != 0 {
		return errors.New("Exact game source not verified")
	}
	return nil
}

// VerifyScenes refuses added inputs/overlays in either encoded scene.
func (o *OBS) VerifyScenes(ctx context.Context) error {
	for scene, source := range map[string]string{Offline: "Renki Standby", Gameplay: Source} {
		var items struct {
			SceneItems []struct {
				SourceName       string `json:"sourceName"`
				SceneItemEnabled bool   `json:"sceneItemEnabled"`
			} `json:"sceneItems"`
		}
		if o.Call(ctx, "GetSceneItemList", map[string]any{"sceneName": scene}, &items) != nil || len(items.SceneItems) != 1 || items.SceneItems[0].SourceName != source || !items.SceneItems[0].SceneItemEnabled {
			return errors.New("Dedicated video scene not verified")
		}
	}
	return nil
}

func (o *OBS) StartLocal(ctx context.Context) error {
	if err := o.VerifyScenes(ctx); err != nil {
		return err
	}
	if err := o.Scene(ctx, Offline); err != nil {
		return err
	}
	if err := o.Call(ctx, "SetStreamServiceSettings", map[string]any{"streamServiceType": "rtmp_custom", "streamServiceSettings": map[string]any{"server": "rtmp://127.0.0.1:11935/renki", "key": "", "use_auth": false}}, nil); err != nil {
		return err
	}
	return o.Call(ctx, "StartStream", nil, nil)
}

func (o *OBS) StopLocal(ctx context.Context) error {
	var state struct {
		OutputActive bool `json:"outputActive"`
	}
	if err := o.Call(ctx, "GetStreamStatus", nil, &state); err != nil {
		return err
	}
	if !state.OutputActive {
		return nil
	}
	var settings struct {
		StreamServiceSettings struct {
			Server string `json:"server"`
		} `json:"streamServiceSettings"`
	}
	if err := o.Call(ctx, "GetStreamServiceSettings", nil, &settings); err != nil {
		return err
	}
	if settings.StreamServiceSettings.Server != "rtmp://127.0.0.1:11935/renki" {
		return errors.New("Refusing nonlocal output control")
	}
	return o.Call(ctx, "StopStream", nil, nil)
}

// SourceLossTrial selects only a nonexistent exact game window, waits for the
// running adapter to select standby, then restores the verified exact window.
// It never changes the game process, output destination or capture mode.
func (o *OBS) SourceLossTrial(ctx context.Context) error {
	if err := o.VerifySource(ctx); err != nil {
		return err
	}
	set := func(window string) error {
		return o.Call(ctx, "SetInputSettings", map[string]any{"inputName": Source, "overlay": true, "inputSettings": map[string]any{"window": window}}, nil)
	}
	defer set(Window)
	if err := set("Renki nonexistent window:UnityWndClass:valheim.exe"); err != nil {
		return err
	}
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		var current struct {
			CurrentProgramSceneName string `json:"currentProgramSceneName"`
		}
		if err := o.Call(ctx, "GetCurrentProgramScene", nil, &current); err != nil {
			return err
		}
		if current.CurrentProgramSceneName == Offline {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return errors.New("Capture loss did not select standby")
}

// Diagnostics contains no OBS configuration, passwords, URLs or raw errors.
func (o *OBS) Diagnostics(ctx context.Context) (map[string]any, error) {
	result := map[string]any{}
	var stats map[string]any
	if err := o.Call(ctx, "GetStats", nil, &stats); err != nil {
		return nil, err
	}
	for _, key := range []string{"activeFps", "cpuUsage", "memoryUsage", "averageFrameRenderTime", "renderSkippedFrames", "renderTotalFrames", "outputSkippedFrames", "outputTotalFrames"} {
		result[key] = stats[key]
	}
	var stream struct {
		OutputActive   bool   `json:"outputActive"`
		OutputBytes    uint64 `json:"outputBytes"`
		OutputDuration uint64 `json:"outputDuration"`
	}
	if err := o.Call(ctx, "GetStreamStatus", nil, &stream); err != nil {
		return nil, err
	}
	result["stream"] = stream
	var scene struct {
		CurrentProgramSceneName string `json:"currentProgramSceneName"`
	}
	if err := o.Call(ctx, "GetCurrentProgramScene", nil, &scene); err != nil {
		return nil, err
	}
	result["scene"] = scene.CurrentProgramSceneName
	var preview struct {
		CurrentPreviewSceneName string `json:"currentPreviewSceneName"`
	}
	if o.Call(ctx, "GetCurrentPreviewScene", nil, &preview) == nil {
		result["preview"] = preview.CurrentPreviewSceneName
	}
	var active struct {
		VideoActive  bool `json:"videoActive"`
		VideoShowing bool `json:"videoShowing"`
	}
	if o.Call(ctx, "GetSourceActive", map[string]any{"sourceName": Source}, &active) == nil {
		result["captureActive"] = active
	}
	var windows struct {
		PropertyItems []struct {
			ItemValue string `json:"itemValue"`
		} `json:"propertyItems"`
	}
	if o.Call(ctx, "GetInputPropertiesListPropertyItems", map[string]any{"inputName": Source, "propertyName": "window"}, &windows) == nil {
		var matches []string
		for _, item := range windows.PropertyItems {
			if strings.HasSuffix(strings.ToLower(item.ItemValue), ":valheim.exe") {
				matches = append(matches, item.ItemValue)
			}
		}
		result["gameWindows"] = matches
	}
	return result, nil
}

func (o *OBS) Snapshot(ctx context.Context, source string) ([]byte, error) {
	if source != Source && source != Offline {
		return nil, errors.New("Snapshot source not allowed")
	}
	var shot struct {
		ImageData string `json:"imageData"`
	}
	if err := o.Call(ctx, "GetSourceScreenshot", map[string]any{"sourceName": source, "imageFormat": "png", "imageWidth": 1280, "imageHeight": 720}, &shot); err != nil {
		return nil, err
	}
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(shot.ImageData, prefix) || len(shot.ImageData) > 3<<20 {
		return nil, errors.New("Snapshot invalid")
	}
	return base64.StdEncoding.DecodeString(strings.TrimPrefix(shot.ImageData, prefix))
}

func (o *OBS) Configure(ctx context.Context, slate string) error {
	var scenes struct {
		Scenes []struct {
			SceneName string `json:"sceneName"`
		} `json:"scenes"`
	}
	if err := o.Call(ctx, "GetSceneList", nil, &scenes); err != nil {
		return err
	}
	for _, name := range []string{Offline, Gameplay} {
		found := false
		for _, s := range scenes.Scenes {
			if s.SceneName == name {
				found = true
			}
		}
		if !found {
			if err := o.Call(ctx, "CreateScene", map[string]any{"sceneName": name}, nil); err != nil {
				return err
			}
		}
	}
	if err := o.Scene(ctx, Offline); err != nil {
		return err
	}
	var inputs struct {
		Inputs []struct {
			InputName string `json:"inputName"`
		} `json:"inputs"`
	}
	if err := o.Call(ctx, "GetInputList", nil, &inputs); err != nil {
		return err
	}
	for _, spec := range []struct {
		name, kind, scene string
		settings          map[string]any
	}{
		{"Renki Standby", "browser_source", Offline, map[string]any{"is_local_file": true, "local_file": slate, "width": 1280, "height": 720, "shutdown": false, "reroute_audio": false}},
		{Source, "game_capture", Gameplay, map[string]any{"capture_mode": "window", "window": Window, "priority": 0, "capture_cursor": false, "capture_overlays": false}},
	} {
		found := false
		for _, input := range inputs.Inputs {
			if input.InputName == spec.name {
				found = true
			}
		}
		if !found {
			if err := o.Call(ctx, "CreateInput", map[string]any{"sceneName": spec.scene, "inputName": spec.name, "inputKind": spec.kind, "inputSettings": spec.settings, "sceneItemEnabled": true}, nil); err != nil {
				return err
			}
		}
	}
	// Keep the game source active in the private studio preview while Program
	// remains on the slate. Preview never reaches the encoder.
	for _, request := range []struct {
		name string
		args map[string]any
	}{
		{"SetStudioModeEnabled", map[string]any{"studioModeEnabled": true}},
		{"SetCurrentPreviewScene", map[string]any{"sceneName": Gameplay}},
		{"SetCurrentSceneTransition", map[string]any{"transitionName": "Cut"}},
	} {
		if err := o.Call(ctx, request.name, request.args, nil); err != nil {
			return err
		}
	}
	var special map[string]any
	if err := o.Call(ctx, "GetSpecialInputs", nil, &special); err != nil {
		return err
	}
	for _, name := range special {
		if n, ok := name.(string); ok && n != "" {
			if err := o.Call(ctx, "SetInputMute", map[string]any{"inputName": n, "inputMuted": true}, nil); err != nil {
				return err
			}
		}
	}
	if err := o.VerifyScenes(ctx); err != nil {
		return err
	}
	return o.VerifySource(ctx)
}
