package memory

import (
	"fmt"
	"time"

	"github.com/warricksothr/renki/internal/home"
)

// Called by the existing observation worker; bridge ingress never writes disk.
func (c *Containers) rememberPickups() {
	conn, observations := c.Session.PickupObservations()
	if conn == nil {
		return
	}
	if conn != c.pickupConn {
		c.pickupConn = conn
		c.pickupSaved = map[uint64]bool{}
	}
	pending := []home.Pickup{}
	for _, p := range observations {
		if c.pickupSaved[p.Seq] {
			continue
		}
		pending = append(pending, home.Pickup{ID: fmt.Sprintf("%d:%d:%d", p.At.UnixNano(), p.Connection, p.Seq), World: p.World, WorldUID: p.WorldUID, Prefab: p.Prefab, Entity: p.Entity, Pos: p.Pos, Count: p.Count, At: p.At})
	}
	if err := c.Store.RememberPickups(pending, time.Now().UTC()); err != nil {
		if c.Logger != nil {
			c.Logger.Warn("could not remember pickups", "err", err)
		}
		return
	}
	c.pickupSaved = map[uint64]bool{}
	for _, p := range observations {
		c.pickupSaved[p.Seq] = true
	}
}
