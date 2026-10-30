package app

import (
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
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

type LandmarkMemoryEntry struct {
	home.Landmark
	AgeS float64 `json:"ageS"`
}

type LandmarkMemoryStatus struct {
	World     string                `json:"world"`
	Landmarks []LandmarkMemoryEntry `json:"landmarks"`
}

func (a *App) LandmarkMemory() any {
	out := LandmarkMemoryStatus{Landmarks: []LandmarkMemoryEntry{}}
	state := a.Session.State()
	if state == nil || state.Phase != protocol.PhaseInWorld {
		return out
	}
	out.World = state.World
	now := time.Now()
	for _, landmark := range a.Landmarks.Store.Landmarks(out.World) {
		out.Landmarks = append(out.Landmarks, LandmarkMemoryEntry{landmark, now.Sub(landmark.At).Seconds()})
	}
	return out
}

type PlayerMemoryStatus struct {
	World      string                 `json:"world"`
	Encounters []home.PlayerEncounter `json:"encounters"`
}

func (a *App) PlayerMemory() any {
	out := PlayerMemoryStatus{Encounters: []home.PlayerEncounter{}}
	st := a.Session.State()
	if st == nil || st.Phase != protocol.PhaseInWorld {
		return out
	}
	out.World = st.World
	out.Encounters = a.Players.History(st.World, time.Now())
	if out.Encounters == nil {
		out.Encounters = []home.PlayerEncounter{}
	}
	return out
}
