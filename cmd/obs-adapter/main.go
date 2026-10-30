// obs-adapter has fixed local OBS operations and no network control listener.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/warricksothr/renki/internal/obscapture"
)

func main() {
	path := flag.String("config", "", "Private configuration path")
	action := flag.String("action", "run", "configure, run, watchdog, or standby")
	flag.Parse()
	c, err := obscapture.ReadConfig(*path)
	if err == nil {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		switch *action {
		case "run", "watchdog":
			err = obscapture.Run(ctx, c, *action == "watchdog")
		case "configure", "standby", "local-start", "local-stop", "status", "snapshot", "game-snapshot", "trial-source-loss":
			var o *obscapture.OBS
			o, err = obscapture.Connect(ctx, c.OBSPassword)
			if err == nil {
				defer o.Close()
				switch *action {
				case "configure":
					err = o.Configure(ctx, c.Slate)
				case "standby":
					err = o.Scene(ctx, obscapture.Offline)
				case "local-start":
					err = o.StartLocal(ctx)
				case "local-stop":
					err = o.StopLocal(ctx)
				case "trial-source-loss":
					err = o.SourceLossTrial(ctx)
				case "status":
					var result map[string]any
					result, err = o.Diagnostics(ctx)
					if err == nil {
						err = json.NewEncoder(os.Stdout).Encode(result)
					}
				case "snapshot", "game-snapshot":
					var b []byte
					source := obscapture.Offline
					if *action == "game-snapshot" {
						source = obscapture.Source
					}
					b, err = o.Snapshot(ctx, source)
					if err == nil {
						err = os.WriteFile(c.Slate+".snapshot.png", b, 0600)
					}
				}
			}
		default:
			err = fmt.Errorf("Unknown adapter action")
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
