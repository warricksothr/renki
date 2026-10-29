package app

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/panel"
	"github.com/warricksothr/renki/internal/protocol"
)

func TestContainerMemoryAPIIsAuthenticatedAndReportsAge(t *testing.T) {
	store := home.Memory()
	now := time.Now().Add(-time.Minute)
	if err := store.RememberContainers("world", []home.RememberedContainer{{Entity: "chest", Size: [2]int{2, 2}, At: now, Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 10}}}}, time.Now()); err != nil {
		t.Fatal(err)
	}
	a := New(config.Default(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, store, nil)
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome { return &protocol.Welcome{} }}, a.Session)
	p := bridgetest.Connect(t, url, nil)
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "world", HP: 25})
	until := time.Now().Add(2 * time.Second)
	for a.Session.State() == nil {
		if time.Now().After(until) {
			t.Fatal("state not received")
		}
		time.Sleep(time.Millisecond)
	}
	handler := panel.Handler(panel.Options{Token: "test", Status: func() any { return nil }, ContainerMemory: a.ContainerMemory})
	get := func(auth bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/api/memory/containers", nil)
		if auth {
			r.Header.Set("Authorization", "Bearer test")
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if w := get(false); w.Code != http.StatusUnauthorized {
		t.Fatal(w.Code)
	}
	w := get(true)
	if w.Code != http.StatusOK {
		t.Fatal(w.Code)
	}
	var result ContainerMemoryStatus
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.World != "world" || len(result.Containers) != 1 || result.Containers[0].AgeS < 60 || result.Containers[0].Inventory().Count("Wood") != 10 {
		t.Fatal(result)
	}
}
