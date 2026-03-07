// NOTE: Replace "github.com/yourname/receptionist" with your actual module path
// (e.g. github.com/yourgithubuser/receptionist) in this file AND in all import
// statements across the codebase before running go mod tidy.
module receptionist

go 1.24.0

require (
	github.com/disgoorg/disgo v0.19.2
	github.com/disgoorg/godave v0.1.0
	github.com/disgoorg/snowflake/v2 v2.0.3
	github.com/hraban/opus v0.0.0-20230925203106-0188a62cb302
	github.com/redis/go-redis/v9 v9.5.1
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/disgoorg/json/v2 v2.0.0 // indirect
	github.com/disgoorg/omit v1.0.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/klauspost/compress v1.18.4 // indirect
	github.com/sasha-s/go-csync v0.0.0-20240107134140-fcbab37b09ad // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
)

// Run `go mod tidy` after cloning to generate go.sum and fetch indirect deps.
// Requires: libopus-dev installed (Linux/WSL2: sudo apt install libopus-dev)
// On Windows native: install opus via https://www.opus-codec.org/downloads/
