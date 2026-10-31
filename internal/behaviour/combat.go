package behaviour

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// Combat delegates one whole local encounter to C#; no movement or target-selection loop lives here.
type Combat struct{}
type combatParams struct {
	World        string          `json:"world"`
	Entity       string          `json:"entity,omitempty"`
	Direct       bool            `json:"direct,omitempty"`
	At           json.RawMessage `json:"at,omitempty"`
	SearchRadius *float64        `json:"searchRadius,omitempty"`
	SearchSector *int            `json:"searchSector,omitempty"`
}

func combatArgs(raw json.RawMessage) (combatParams, *protocol.AttackArgs, error) {
	var p combatParams
	if err := Decode(raw, &p); err != nil {
		return p, nil, err
	}
	if strings.TrimSpace(p.World) == "" {
		return p, nil, errors.New("combat needs a world")
	}
	if p.Direct && (p.Entity == "" || p.At != nil) {
		return p, nil, errors.New("direct combat needs only an entity")
	}
	args := map[string]any{"kite": !p.Direct, "style": "melee"}
	if p.Entity != "" {
		args["entity"] = p.Entity
	}
	if p.At != nil {
		args["at"] = p.At
	}
	if p.SearchRadius != nil {
		args["searchRadius"] = *p.SearchRadius
	}
	if p.SearchSector != nil {
		args["searchSector"] = *p.SearchSector
	}
	encoded, err := json.Marshal(args)
	if err != nil {
		return p, nil, err
	}
	req := &protocol.TaskRequest{Task: protocol.TaskAttack, Args: encoded}
	parsed, err := req.DecodeArgs()
	if err != nil {
		return p, nil, err
	}
	return p, parsed.(*protocol.AttackArgs), nil
}
func (Combat) Validate(raw json.RawMessage) error { _, _, err := combatArgs(raw); return err }
func (Combat) Start(raw json.RawMessage) (Runner, error) {
	p, args, err := combatArgs(raw)
	if err != nil {
		return nil, err
	}
	return &combatRun{world: p.World, args: args}, nil
}

type combatRun struct {
	world string
	args  *protocol.AttackArgs
	sent  bool
}

func (r *combatRun) Next(v View, _ *Result) (*session.Order, error) {
	if r.sent {
		return nil, nil
	}
	if v.State == nil || v.State.Phase != protocol.PhaseInWorld || v.State.World != r.world {
		return nil, errors.New("combat world is no longer current")
	}
	if v.State.Food == nil || len(v.State.Food.Active) == 0 {
		return nil, errors.New("eat before starting combat")
	}
	r.sent = true
	return &session.Order{Task: protocol.TaskAttack, Args: r.args, TimeoutS: 270}, nil
}
