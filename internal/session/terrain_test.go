package session

import (
	"testing"

	"github.com/warricksothr/renki/internal/protocol"
)

func terrainTile(x, z int) protocol.TerrainTile {
	h, biome := 32.0, "Meadows"
	t := protocol.TerrainTile{X: x, Z: z, WaterLevel: 30, Heights: make([]*float64, 25), Biomes: make([]*string, 25)}
	for i := range t.Heights {
		t.Heights[i], t.Biomes[i] = &h, &biome
	}
	return t
}

func TestTerrainPatchesAreOrderedValidatedAndScopedToWorldAndConnection(t *testing.T) {
	s, _, p := setup(t, nil, nil)
	p.Send(&protocol.State{World: "one", Phase: protocol.PhaseInWorld})
	waitFor(t, func() bool { return s.State() != nil })
	c := currentConn(t, s)
	patch := &protocol.Terrain{Envelope: protocol.Envelope{Seq: 100}, World: "one", Tiles: []protocol.TerrainTile{terrainTile(-1, 0)}}
	s.Message(c, patch)
	w, conn, got := s.TerrainSnapshot()
	if w != "one" || conn == 0 || len(got) != 1 || got[0].X != -1 {
		t.Fatal(w, conn, got)
	}
	patch = &protocol.Terrain{Envelope: protocol.Envelope{Seq: 99}, World: "one", Tiles: []protocol.TerrainTile{terrainTile(2, 0)}}
	s.Message(c, patch)
	patch.Seq = 101
	patch.World = "two"
	s.Message(c, patch)
	patch.World = "one"
	patch.Tiles[0].Heights = nil
	s.Message(c, patch)
	_, _, got = s.TerrainSnapshot()
	if len(got) != 1 {
		t.Fatal("stale, invalid or wrong-world patch was accepted")
	}
	s.Message(c, &protocol.State{Envelope: protocol.Envelope{Seq: 200}, Phase: protocol.PhaseInWorld, World: "two"})
	w, _, got = s.TerrainSnapshot()
	if w != "two" || len(got) != 0 {
		t.Fatal("old terrain crossed world change")
	}
	patch.World, patch.Seq, patch.Tiles = "two", 201, []protocol.TerrainTile{terrainTile(0, 0)}
	s.Message(c, patch)
	s.Connected(c)
	_, _, got = s.TerrainSnapshot()
	if len(got) != 0 {
		t.Fatal("terrain survived replacement without a new observation")
	}
}

func TestTerrainRetentionIsBoundedAndKeepsNearestTiles(t *testing.T) {
	s, _, p := setup(t, nil, nil)
	p.Send(&protocol.State{World: "one", Phase: protocol.PhaseInWorld})
	waitFor(t, func() bool { return s.State() != nil })
	c := currentConn(t, s)
	for i := 0; i < MaxTerrainTiles+50; i++ {
		s.Message(c, &protocol.Terrain{Envelope: protocol.Envelope{Seq: uint64(100 + i)}, World: "one", Tiles: []protocol.TerrainTile{terrainTile(i, 0)}})
	}
	_, _, got := s.TerrainSnapshot()
	if len(got) != MaxTerrainTiles || got[0].X != 0 || got[len(got)-1].X != MaxTerrainTiles-1 {
		t.Fatal("terrain was unbounded or nearest tiles were evicted")
	}
	s.Disconnected(c, nil)
	if w, _, got := s.TerrainSnapshot(); w != "" || len(got) != 0 {
		t.Fatal("disconnected terrain presented as live")
	}
}

func TestDurableTerrainRequiresFreshnessAndMatchingIdentity(t *testing.T) {
	s, _, p := setup(t, nil, nil)
	p.Send(&protocol.State{World: "one", WorldUID: "9223372036854775807", Phase: protocol.PhaseInWorld})
	waitFor(t, func() bool { return s.State() != nil })
	c := currentConn(t, s)
	scope := protocol.TerrainScope{Namespace: "server", WorldUID: "9223372036854775807", Epoch: "1", Source: "bot", Stream: "run", Game: "game", Generator: 2, Mods: "hash"}
	tile := terrainTile(-1, 0)
	at := 9.0
	tile.MeasuredAt = make([]*float64, 25)
	for i := range tile.MeasuredAt {
		tile.MeasuredAt[i] = &at
	}
	patch := &protocol.Terrain{Envelope: protocol.Envelope{Seq: 100}, World: "one", Scope: &scope, CapturedAt: 10, Tiles: []protocol.TerrainTile{tile}}
	s.Message(c, patch)
	batches, dropped := s.DrainTerrain()
	if len(batches) != 1 || dropped != 0 || batches[0].Scope.WorldUID != scope.WorldUID || batches[0].ReceivedAt.IsZero() {
		t.Fatal(batches, dropped)
	}
	s.Message(c, patch) // duplicate sequence
	patch.Seq++
	patch.CapturedAt = 8 // measurements from the future
	s.Message(c, patch)
	patch.Seq++
	patch.CapturedAt = 10
	other := scope
	other.WorldUID = "42"
	patch.Scope = &other
	s.Message(c, patch) // same name, different save
	if got, _ := s.DrainTerrain(); len(got) != 0 {
		t.Fatal("invalid/stale identity reached disk queue")
	}
	// Accepted batches retain their original scope after reconnect.
	patch.Scope = &scope
	patch.Seq++
	s.Message(c, patch)
	s.Connected(c)
	batches, _ = s.DrainTerrain()
	if len(batches) != 1 || batches[0].Scope != scope {
		t.Fatal("accepted scope lost on reconnect")
	}
	_, _, current, _ := s.TerrainView()
	if current != nil {
		t.Fatal("live identity survived reconnect")
	}
}

func TestLegacyTerrainNeverEntersDurableQueue(t *testing.T) {
	s, _, p := setup(t, nil, nil)
	p.Send(&protocol.State{World: "one", Phase: protocol.PhaseInWorld})
	waitFor(t, func() bool { return s.State() != nil })
	s.Message(currentConn(t, s), &protocol.Terrain{Envelope: protocol.Envelope{Seq: 100}, World: "one", Tiles: []protocol.TerrainTile{terrainTile(0, 0)}})
	if got, _ := s.DrainTerrain(); len(got) != 0 {
		t.Fatal("legacy tiles received artificial freshness")
	}
}

func TestDurableTerrainQueueIsBoundedAndKeepsAcceptedBatches(t *testing.T) {
	s, _, p := setup(t, nil, nil)
	p.Send(&protocol.State{World: "one", WorldUID: "42", Phase: protocol.PhaseInWorld})
	waitFor(t, func() bool { return s.State() != nil })
	c := currentConn(t, s)
	scope := protocol.TerrainScope{Namespace: "server", WorldUID: "42", Epoch: "1", Source: "bot", Stream: "run", Game: "game", Generator: 2, Mods: "hash"}
	tile := terrainTile(0, 0)
	at := 1.0
	tile.MeasuredAt = make([]*float64, 25)
	for i := range tile.MeasuredAt {
		tile.MeasuredAt[i] = &at
	}
	for i := 0; i < MaxTerrainPending+1; i++ {
		s.Message(c, &protocol.Terrain{Envelope: protocol.Envelope{Seq: uint64(100 + i)}, World: "one", Scope: &scope, CapturedAt: 1, Tiles: []protocol.TerrainTile{tile}})
	}
	batches, dropped := s.DrainTerrain()
	if len(batches) != MaxTerrainPending || dropped != 1 {
		t.Fatal(len(batches), dropped)
	}
	if rest, drops := s.DrainTerrain(); len(rest) != 0 || drops != 0 {
		t.Fatal("drain did not clear queue")
	}
}
