package webui

import "embed"

var (
	//go:embed dist/*
	DistFiles embed.FS
)
