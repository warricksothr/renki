// Package memory records durable observations without requesting game actions.
package memory

import (
	"context"
	"log/slog"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

type Containers struct {
	Session *session.Session
	Store   *home.Store
	Logger  *slog.Logger
	conn    *protocol.Hello
	world   string
	saved   map[string]*protocol.Container
}

func (c *Containers) Run(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			c.Tick()
			return
		case <-t.C:
			c.Tick()
		}
	}
}
func (c *Containers) Tick() {
	conn, world, perception, observed := c.Session.ContainerObservations()
	if conn == nil {
		return
	}
	if conn != c.conn || world != c.world {
		c.conn, c.world = conn, world
		c.saved = map[string]*protocol.Container{}
	}
	updates := make([]home.RememberedContainer, 0, len(observed))
	for _, observation := range observed {
		frame := observation.Frame
		if c.saved[frame.Entity] == frame {
			continue
		}
		item := home.RememberedContainer{Entity: frame.Entity, Kind: protocol.StructureContainer, Pos: frame.Pos, Size: frame.Size, Items: frame.Items, At: observation.At}
		if perception != nil {
			for _, piece := range perception.Structures {
				if piece.ID == frame.Entity {
					item.Prefab = piece.Prefab
					break
				}
			}
		}
		updates = append(updates, item)
	}
	if len(updates) == 0 {
		return
	}
	if err := c.Store.RememberContainers(world, updates, time.Now()); err != nil {
		if c.Logger != nil {
			c.Logger.Warn("could not remember containers", "world", world, "err", err)
		}
		return
	}
	for _, observation := range observed {
		c.saved[observation.Frame.Entity] = observation.Frame
	}
	if c.Logger != nil {
		c.Logger.Info("remembered containers", "world", world, "count", len(updates))
	}
}
