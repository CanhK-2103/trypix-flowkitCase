package examples

import "embed"

//go:embed models/*.json workflows/*.json
var Files embed.FS
