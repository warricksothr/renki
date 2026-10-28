// Command fakeplugin stands in for the Valheim plugin, so the controller can
// be exercised with no game running. Each line on stdin is chat from a
// player; tasks the controller sends are printed and acknowledged as the
// real plugin would: body tasks accepted then running, instant tasks
// succeeded. `Name: text` speaks as Name; a line without a prefix speaks as
// -player. `/whisper text` whispers.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"

	"github.com/warricksothr/renki/internal/protocol"
)

func main() {
	url := flag.String("url", "ws://127.0.0.1:8765/ws/", "the controller's bridge URL")
	token := flag.String("token", os.Getenv("RENKI_TOKEN"), "the token to send in hello")
	player := flag.String("player", "Sothr", "who a line with no `Name:` prefix is from")
	linger := flag.Duration("linger", 30*time.Second, "how long to keep listening after stdin ends")
	flag.Parse()
	if err := run(*url, *token, *player, *linger); err != nil {
		fmt.Fprintln(os.Stderr, "fakeplugin:", err)
		os.Exit(1)
	}
}

type plugin struct {
	ws  *websocket.Conn
	seq atomic.Uint64
	mu  sync.Mutex
}

func (p *plugin) send(m protocol.Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	env := protocol.EnvelopeOf(m)
	env.Seq = p.seq.Add(1)
	env.T = float64(time.Now().Unix() % 100000)
	data, err := protocol.Encode(m)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return p.ws.Write(ctx, websocket.MessageText, data)
}

func run(url, token, player string, linger time.Duration) error {
	ctx := context.Background()
	ws, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		return err
	}
	defer ws.CloseNow()
	p := &plugin{ws: ws}
	if err := p.send(&protocol.Hello{Token: token, Plugin: "fakeplugin", Game: "none", Phase: protocol.PhaseInWorld,
		Character: "Bjorn", World: "Nowhere",
		Capabilities: []string{"stop", "move_to", "follow", "look_at", "say", "emote"}}); err != nil {
		return err
	}

	gone := make(chan error, 1)
	go func() { gone <- p.read() }()
	go p.stream()

	lines := make(chan string)
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			lines <- sc.Text()
		}
	}()
	for {
		select {
		case err := <-gone:
			return err
		case line, ok := <-lines:
			if !ok {
				select {
				case err := <-gone:
					return err
				case <-time.After(linger):
					_ = p.send(&protocol.Bye{Reason: "quit"})
					return nil
				}
			}
			if strings.TrimSpace(line) == "" {
				continue
			}
			if err := p.send(chatLine(line, player)); err != nil {
				return err
			}
		}
	}
}

func chatLine(line, player string) *protocol.Chat {
	c := &protocol.Chat{Channel: protocol.ChannelNormal, Pos: protocol.Vec3{10, 30, 10}, Dist: 4}
	if rest, ok := strings.CutPrefix(line, "/whisper "); ok {
		c.Channel, line = protocol.ChannelWhisper, rest
	}
	from := player
	if name, text, ok := strings.Cut(line, ":"); ok && !strings.ContainsRune(name, ' ') {
		from, line = name, strings.TrimSpace(text)
	}
	c.From = protocol.ChatSender{Name: from, PlayerID: "1"}
	c.Text = line
	fmt.Printf("%-8s %s\n", from+">", line)
	return c
}

// stream sends a plausible state once a second.
func (p *plugin) stream() {
	for range time.Tick(time.Second) {
		_ = p.send(&protocol.State{Phase: protocol.PhaseInWorld, Pos: protocol.Vec3{0, 30, 0}, HP: 47, HPMax: 60,
			Stamina: 38, StaminaMax: 75, Biome: "Meadows", Env: "Clear",
			Time: protocol.GameTime{Day: 3, DayFraction: 0.4}})
	}
}

func (p *plugin) read() error {
	for {
		_, data, err := p.ws.Read(context.Background())
		if err != nil {
			return fmt.Errorf("connection closed: %w", err)
		}
		m, err := protocol.Decode(data, protocol.FromController)
		if err != nil {
			fmt.Printf("!! %v: %s\n", err, data)
			continue
		}
		switch m := m.(type) {
		case *protocol.Ping:
			_ = p.send(&protocol.Pong{Envelope: protocol.Envelope{Re: m.ID}})
		case *protocol.Welcome:
			fmt.Printf("-- welcome: %s, owners %v, allowed %v\n", m.AgentName, m.Owners, m.AllowedTasks)
		case *protocol.TaskRequest:
			p.task(m)
		default:
			fmt.Printf("<< %s\n", data)
		}
	}
}

func (p *plugin) task(req *protocol.TaskRequest) {
	args, err := req.DecodeArgs()
	if err != nil {
		fmt.Printf("<< %s %s: rejected, %v\n", req.ID, req.Task, err)
		_ = p.send(&protocol.TaskStatus{Envelope: protocol.Envelope{Re: req.ID}, Status: protocol.StatusRejected, Reason: protocol.ReasonBadArgs})
		return
	}
	if say, ok := args.(*protocol.SayArgs); ok {
		fmt.Printf("%-8s %s\n", "Bjorn>", say.Text)
	} else {
		fmt.Printf("<< %s %s %s\n", req.ID, req.Task, req.Args)
	}
	if protocol.Instant(req.Task) {
		_ = p.send(&protocol.TaskStatus{Envelope: protocol.Envelope{Re: req.ID}, Status: protocol.StatusSucceeded})
		return
	}
	_ = p.send(&protocol.TaskStatus{Envelope: protocol.Envelope{Re: req.ID}, Status: protocol.StatusAccepted})
	status := protocol.StatusRunning
	if req.Task == protocol.TaskStop {
		status = protocol.StatusSucceeded
	}
	_ = p.send(&protocol.TaskStatus{Envelope: protocol.Envelope{Re: req.ID}, Status: status})
}
