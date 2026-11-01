package session

import (
	"slices"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

// InspectionSnapshot captures controller ownership and stream scope under one
// lock. State frames are immutable. Task origins stay internal to the app.
type InspectionSnapshot struct {
	Connected             bool
	Connection            uint64
	State                 *protocol.State
	At                    time.Time
	Automation            bool
	ControlRevision       uint64
	Capabilities, Allowed []string
	Task                  *InspectionTask
}

type InspectionTask struct {
	ID, Name, Status  string
	SentAt            time.Time
	Origin            any
	AutomationCurrent bool
}

func (s *Session) InspectionSnapshot() InspectionSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	o := InspectionSnapshot{ControlRevision: s.orders}
	if s.conn == nil {
		return o
	}
	o.Connected, o.Connection, o.State, o.At = true, s.conn.ID(), s.state, s.stateAt
	o.Automation = s.automation != nil
	o.Capabilities = slices.Clone(s.conn.Hello.Capabilities)
	for _, task := range s.allowed {
		if s.permitted(task) {
			o.Allowed = append(o.Allowed, task)
		}
	}
	if s.state == nil || s.state.Phase != protocol.PhaseInWorld {
		return o
	}
	var latest *Task
	for _, task := range s.tasks {
		if protocol.Instant(task.Name) || task.connection != o.Connection || task.world != s.state.World || task.worldUID != s.state.WorldUID {
			continue
		}
		if latest == nil || task.serial > latest.serial {
			latest = task
		}
	}
	if latest != nil {
		status, _, _ := latest.Status()
		o.Task = &InspectionTask{ID: latest.ID, Name: latest.Name, Status: status, SentAt: latest.sentAt,
			Origin: latest.Origin, AutomationCurrent: latest.automation != nil && latest.automation == s.automation}
	}
	return o
}
