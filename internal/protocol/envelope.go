// Package protocol holds the Go side of bridge protocol v1, the contract
// between the C# plugin inside Valheim and this controller. The types follow
// docs/BRIDGE-PROTOCOL.md section by section, and the fixtures under testdata/
// are the doc's examples, shared with the plugin's tests so the two
// implementations cannot drift apart quietly.
package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

// Version is the protocol major version this package speaks (section 3).
const Version = 1

// Envelope is the set of fields every message carries (section 3). Each
// message type embeds it.
type Envelope struct {
	V    int     `json:"v"`
	Type string  `json:"type"`
	Seq  uint64  `json:"seq"`
	T    float64 `json:"t"`
	ID   string  `json:"id,omitempty"`
	Re   string  `json:"re,omitempty"`
}

func (e *Envelope) envelope() *Envelope { return e }

// Message is any v1 message. MessageType is the value of its `type` field.
type Message interface {
	MessageType() string
	envelope() *Envelope
}

// Sender says which side wrote a message. `task` is a request when the
// controller sends it and a status report when the plugin does, so decoding
// needs to know.
type Sender int

const (
	FromPlugin Sender = iota
	FromController
)

func (s Sender) String() string {
	switch s {
	case FromPlugin:
		return "plugin"
	case FromController:
		return "controller"
	}
	return fmt.Sprintf("Sender(%d)", int(s))
}

// ErrUnknownType is returned for a `type` the sender is not known to send.
// The receiver answers it with an `error` message carrying CodeBadMessage.
var ErrUnknownType = errors.New("unknown message type")

// VersionError is returned for a major version other than Version. The
// receiver rejects the message rather than guessing at it.
type VersionError struct{ Got int }

func (e *VersionError) Error() string {
	return fmt.Sprintf("unsupported protocol version %d (this side speaks %d)", e.Got, Version)
}

var registry = map[Sender]map[string]func() Message{
	FromPlugin: {
		TypeHello:      func() Message { return &Hello{} },
		TypeState:      func() Message { return &State{} },
		TypePerception: func() Message { return &Perception{} },
		TypeInventory:  func() Message { return &Inventory{} },
		TypeLayout:     func() Message { return &Layout{} },
		TypeChat:       func() Message { return &Chat{} },
		TypeEvent:      func() Message { return &Event{} },
		TypeTask:       func() Message { return &TaskStatus{} },
		TypePong:       func() Message { return &Pong{} },
		TypeBye:        func() Message { return &Bye{} },
		TypeError:      func() Message { return &Error{} },
		TypeTrace:      func() Message { return &Trace{} },
	},
	FromController: {
		TypeWelcome: func() Message { return &Welcome{} },
		TypeTask:    func() Message { return &TaskRequest{} },
		TypePing:    func() Message { return &Ping{} },
		TypeConfig:  func() Message { return &Config{} },
		TypeLogout:  func() Message { return &Logout{} },
		TypeBye:     func() Message { return &Bye{} },
		TypeError:   func() Message { return &Error{} },
	},
}

// Decode reads one frame written by from. It checks the version before the
// type, so a v2 peer is told about the version rather than about a type that
// v2 may have renamed. Unknown fields are ignored.
//
// When the version and type are fine but the body is not, Decode returns the
// envelope it managed to read alongside the error, so the caller can address
// its `error` reply with `re`.
func Decode(data []byte, from Sender) (Message, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("decode envelope: %w", err)
	}
	if env.V != Version {
		return nil, &VersionError{Got: env.V}
	}
	ctor, ok := registry[from][env.Type]
	if !ok {
		return nil, fmt.Errorf("%w %q from the %s", ErrUnknownType, env.Type, from)
	}
	msg := ctor()
	if err := json.Unmarshal(data, msg); err != nil {
		return nil, fmt.Errorf("decode %s: %w", env.Type, err)
	}
	return msg, nil
}

// Encode writes msg as one frame, stamping `v` and `type` so callers only
// fill in the fields that vary.
func Encode(msg Message) ([]byte, error) {
	env := msg.envelope()
	env.V = Version
	env.Type = msg.MessageType()
	return json.Marshal(msg)
}

// Vec3 is a Unity world-space position in metres, y up (section 4).
type Vec3 [3]float64

// FlatDistance is the distance from v to w on the ground, ignoring height.
func (v Vec3) FlatDistance(w Vec3) float64 { return math.Hypot(v[0]-w[0], v[2]-w[2]) }

// EnvelopeOf returns the envelope inside m, for code that stamps `seq` and
// `t` or reads `id` and `re` without knowing the message type.
func EnvelopeOf(m Message) *Envelope { return m.envelope() }

// IDOf reads `id` from a frame that may not decode, so an `error` reply can
// still name what it answers. It returns "" when there is none.
func IDOf(data []byte) string {
	var env struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(data, &env)
	return env.ID
}
