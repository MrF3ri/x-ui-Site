package frontend

import "embed"

//go:embed assets/templates/store/*.html assets/public/css/*
var EmbeddedFiles embed.FS
