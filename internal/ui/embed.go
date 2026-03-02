package ui

import "embed"

// Embed статики веб-приложения. static/ заполняется при сборке (task build).
//go:embed static
var Static embed.FS
