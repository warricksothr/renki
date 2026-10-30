# Local-only video tooling; all listeners bind to loopback. No VM recipes.
set working-directory := '../..'

install:
    python3 scripts/streaming/local.py install

# Render the shared browser slate once; encode a matching H264 offline segment.
prepare:
    just --justfile justfile panel build
    node panel/ui/slate.mjs
    ffmpeg -hide_banner -loglevel error -y -loop 1 -i .checks/streaming/standby.png -t 4 -r 30 -an -c:v libx264 -preset fast -profile:v baseline -pix_fmt yuv420p -g 60 -bf 0 -movflags +faststart .checks/streaming/standby.mp4
    python3 scripts/streaming/local.py prepare

relay:
    python3 scripts/streaming/local.py relay

source:
    python3 scripts/streaming/local.py source

slate:
    just --justfile justfile panel build
    python3 scripts/streaming/local.py slate

# Read-only authenticated synthetic panel, never connected to a game.
proof-host:
    just --justfile justfile controller panel-assets
    go run -buildvcs=false ./cmd/videoproof

# Starts/stops its own loopback relay, FFmpeg, proof host and Chromium.
proof: prepare
    node panel/ui/video-proof.mjs

test:
    python3 -m unittest discover -s scripts/streaming -p '*_test.py'

# Explicit synthetic codec alternative for Chromium builds without H264.
prepare-vp9:
    just --justfile justfile panel build
    node panel/ui/slate.mjs
    ffmpeg -hide_banner -loglevel error -y -loop 1 -i .checks/streaming/standby.png -t 4 -r 30 -an -c:v libvpx-vp9 -deadline realtime -cpu-used 8 -row-mt 1 -pix_fmt yuv420p -g 60 .checks/streaming/standby.mp4
    python3 scripts/streaming/local.py prepare

source-vp9:
    python3 scripts/streaming/local.py source-vp9

proof-vp9: prepare-vp9
    RENKI_PROOF_CODEC=vp9 node panel/ui/video-proof.mjs

# Synthetic host only: offline, online, source-on or source-off. No game action.
proof-state state:
    python3 scripts/streaming/local.py proof-state '{{state}}'
