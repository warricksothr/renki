// Command controller is renki's brain: it accepts the plugin's WebSocket,
// decides what the bot does, and routes chat. docs/BRIDGE-PROTOCOL.md is the
// contract it speaks.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/warricksothr/renki/internal/app"
	"github.com/warricksothr/renki/internal/audit"
	"github.com/warricksothr/renki/internal/chat"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/settings"
)

// version is stamped at build time with -ldflags "-X main.version=...".
var version = "devel"

func main() {
	os.Exit(run())
}

func run() int {
	showVersion := flag.Bool("version", false, "print the version and exit")
	configPath := flag.String("config", "", "the configuration file (JSON); defaults apply to anything it leaves out")
	listen := flag.String("listen", "", "override the configured listen address")
	debug := flag.Bool("debug", false, "log at debug level")
	flag.Parse()

	if *showVersion {
		fmt.Println("renki controller", version)
		return 0
	}

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	console := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	log := slog.New(console)
	slog.SetDefault(log)

	cfg, err := config.Load(*configPath)
	if *listen != "" && err == nil {
		cfg.Listen = *listen
		err = cfg.Validate()
	}
	if err != nil {
		log.Error("configuration", "err", err)
		return 2
	}

	// The audit log takes every record at debug level, whatever the
	// console shows. With no directory it writes no files but still feeds
	// the panel.
	trail, err := audit.Open(cfg.Audit.Dir, cfg.Audit.KeepDays, nil)
	if err != nil {
		log.Error("audit log", "dir", cfg.Audit.Dir, "err", err)
		return 2
	}
	defer trail.Close()
	log = slog.New(slog.NewMultiHandler(console, trail.Handler(slog.LevelDebug)))
	slog.SetDefault(log)
	trail.Write(audit.Record{Kind: audit.KindStart, Msg: "renki controller " + version,
		Attrs: map[string]any{"config": redacted(cfg), "pid": os.Getpid()}})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ln, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		log.Error("listen", "err", err)
		return 1
	}
	homes, err := home.Open(filepath.Join(cfg.DataDir, "memory.json"))
	if err != nil {
		log.Error("memory", "dir", cfg.DataDir, "err", err)
		return 2
	}
	set, err := settings.Open(filepath.Join(cfg.DataDir, "settings.json"), cfg, log.With("part", "settings"), nil)
	if err != nil {
		log.Error("settings", "dir", cfg.DataDir, "err", err)
		return 2
	}
	warn := func() {
		for _, w := range set.Warnings() {
			log.Warn("settings", "warning", w)
		}
	}
	warn()
	set.OnChange(func(settings.Values) { warn() })

	// A voice that cannot be set up, such as for a key that is not set,
	// leaves the bot on the command grammar rather than off.
	var voice chat.Voice
	if cfg.LLM != nil {
		if voice, err = newVoice(cfg, func() settings.Model { return set.Values().Model }, log); err != nil {
			log.Error("llm voice off; the bot answers the command grammar only", "err", err)
			voice = nil
		}
	}
	log.Info("renki controller starting", "version", version, "agent", cfg.AgentName,
		"owners", set.Values().Owners, "llm", voice != nil, "audit", cfg.Audit.Dir)
	a := app.New(cfg, voice, log, trail, homes, set)
	go a.ServePanel(ctx, trail, version, log.With("part", "panel"))
	if err := a.Serve(ctx, ln); err != nil {
		log.Error("serve", "err", err)
		return 1
	}
	return 0
}

// redacted is cfg without the secrets it may hold: the bridge token, and
// the terva block, which can carry provider keys.
func redacted(cfg config.Config) config.Config {
	if cfg.Token != "" {
		cfg.Token = "[redacted]"
	}
	if cfg.Panel.Token != "" {
		cfg.Panel.Token = "[redacted]"
	}
	if cfg.LLM != nil {
		llm := *cfg.LLM
		if len(llm.Terva) > 0 {
			llm.Terva = []byte(`"[redacted]"`)
		}
		cfg.LLM = &llm
	}
	return cfg
}
