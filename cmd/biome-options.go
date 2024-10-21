package cmd

import (
	"context"

	"github.com/orirawlings/gh-biome/internal/biome"
)

// biomeOptions can be overridden during tests
var biomeOptions []biome.BiomeOption

func load(ctx context.Context) (biome.Biome, error) {
	return biome.Load(ctx, ".", biomeOptions...)
}
