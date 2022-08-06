package webui

import "embed"

var (
	//go:embed dist/*
	DistFiles embed.FS

	//go:embed assets/favicon.ico
	FavIco []byte

	//go:embed docs/version
	Version []byte
)
