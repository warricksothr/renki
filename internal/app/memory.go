package app

import (
	"github.com/warricksothr/renki/internal/home"
	"time"
)

type ContainerMemoryEntry struct {
	home.RememberedContainer
	AgeS float64 `json:"ageS"`
}
type ContainerMemoryStatus struct {
	World      string                 `json:"world"`
	Containers []ContainerMemoryEntry `json:"containers"`
}

func (a *App) ContainerMemory() any {
	out := ContainerMemoryStatus{Containers: []ContainerMemoryEntry{}}
	state := a.Session.State()
	if state == nil {
		return out
	}
	out.World = state.World
	now := time.Now()
	for _, c := range a.Memory.Store.RememberedContainers(out.World) {
		out.Containers = append(out.Containers, ContainerMemoryEntry{c, now.Sub(c.At).Seconds()})
	}
	return out
}
