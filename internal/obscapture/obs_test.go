package obscapture

import (
	"context"
	"encoding/json"
	"github.com/coder/websocket"
	"net/http"
	"net/http/httptest"
	"strings"
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
