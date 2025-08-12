package cmd

import (
	"context"
	"github.com/Digital-Shane/treeview"
	"github.com/Digital-Shane/title-tidy/internal/core"
	"github.com/Digital-Shane/title-tidy/internal/media"
)

var SeasonsCommand = CommandConfig{
	maxDepth:    2,
	includeDirs: true,
	annotate: func(t *treeview.Tree[treeview.FileInfo]) {
		for ni := range t.All(context.Background()) {
			m := core.EnsureMeta(ni.Node)
			if ni.Depth == 0 {
				m.Type = core.MediaSeason
				m.NewName = media.FormatSeasonName(ni.Node.Name())
			} else {
				m.Type = core.MediaEpisode
				m.NewName = media.FormatEpisodeName(ni.Node.Name(), ni.Node)
			}
		}
	},
}
