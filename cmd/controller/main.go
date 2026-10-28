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
	"syscall"

	"github.com/warricksothr/renki/internal/app"
	"github.com/warricksothr/renki/internal/chat"
	"github.com/warricksothr/renki/internal/config"
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
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
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

	var voice chat.Voice
	if cfg.LLM != nil {
		voice, err = newVoice(cfg, log)
		if err != nil {
			log.Error("llm", "err", err)
			return 2
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ln, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		log.Error("listen", "err", err)
		return 1
	}
	log.Info("renki controller starting", "version", version, "agent", cfg.AgentName,
		"owners", cfg.Owners, "llm", voice != nil)
	if err := app.New(cfg, voice, log).Serve(ctx, ln); err != nil {
		log.Error("serve", "err", err)
		return 1
	}
	return 0
}
