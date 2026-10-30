// Package web holds the built UI (web/dist, from `bun run build`), embedded
// into the binary.
package web

import "embed"

//go:embed all:dist
var Dist embed.FS
