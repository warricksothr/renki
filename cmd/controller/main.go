// Command controller is renki's brain: it accepts the plugin's WebSocket,
// decides what the bot does, and routes chat. docs/BRIDGE-PROTOCOL.md is the
// contract it speaks.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
)

// version is stamped at build time with -ldflags "-X main.version=...".
var version = "devel"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("renki controller", version)
		return
	}

	slog.Info("renki controller starting", "version", version)
	slog.Error("the bridge server is not written yet")
	os.Exit(1)
}
