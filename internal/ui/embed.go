package ui

import "embed"

// Static содержит embed статики веб-приложения. static/ заполняется при сборке (task build).
//go:embed static
var Static embed.FS
