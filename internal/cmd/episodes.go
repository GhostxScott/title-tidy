package cmd

import (
	"context"
	"github.com/Digital-Shane/treeview"
	"github.com/Digital-Shane/title-tidy/internal/core"
	"github.com/Digital-Shane/title-tidy/internal/media"
)

// EpisodesCommand processes a flat directory of episode files (no parent season folder).
// Each top-level media file is classified as an episode and renamed solely based on
// information present in its own filename (no contextual season inference).
var EpisodesCommand = CommandConfig{
	maxDepth:    1,
	includeDirs: false,
	annotate: func(t *treeview.Tree[treeview.FileInfo]) {
		for ni := range t.All(context.Background()) {
			// Only operate on leaf nodes (files) at depth 0; directories are excluded by includeDirs=false.
			if ni.Node.Data().IsDir() {
				continue
			}
			m := core.EnsureMeta(ni.Node)
			m.Type = core.MediaEpisode
			m.NewName = media.FormatEpisodeName(ni.Node.Name(), ni.Node)
		}
	},
}
