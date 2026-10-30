module github.com/KeesTucker/dash-comma

go 1.27.3

// bun puts a Go package in node_modules (flatted); keep it out of ./...
ignore ./web/node_modules

require (
	github.com/NVIDIA/go-nvml v0.13.4-1
	github.com/klauspost/compress v1.20.3
	github.com/robfig/cron/v3 v3.0.1
	github.com/zishang520/socket.io/servers/socket/v3 v3.0.6
	github.com/zishang520/socket.io/v3 v3.0.6
	golang.org/x/sys v0.49.0
)

require (
	github.com/andybalholm/brotli v1.2.4 // indirect
	github.com/dunglas/httpsfv v1.1.1 // indirect
	github.com/gookit/color v1.6.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.62.0 // indirect
	github.com/quic-go/webtransport-go v0.13.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/xo/terminfo v1.2.0 // indirect
	github.com/zishang520/socket.io/parsers/engine/v3 v3.0.6 // indirect
	github.com/zishang520/socket.io/parsers/socket/v3 v3.0.6 // indirect
	github.com/zishang520/socket.io/servers/engine/v3 v3.0.6 // indirect
	golang.org/x/crypto v0.58.0 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/text v0.43.0 // indirect
)
