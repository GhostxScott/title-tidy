package cmd

import (
	"context"
	"github.com/Digital-Shane/treeview"
	"github.com/Digital-Shane/title-tidy/internal/core"
	"github.com/Digital-Shane/title-tidy/internal/media"
)

var ShowsCommand = CommandConfig{
	maxDepth:    3,
	includeDirs: true,
	annotate: func(t *treeview.Tree[treeview.FileInfo]) {
		for ni := range t.All(context.Background()) {
			m := core.EnsureMeta(ni.Node)
			switch ni.Depth {
			case 0:
				m.Type = core.MediaShow
				m.NewName = media.FormatShowName(ni.Node.Name())
			case 1:
				m.Type = core.MediaSeason
				m.NewName = media.FormatSeasonName(ni.Node.Name())
			default:
				m.Type = core.MediaEpisode
				m.NewName = media.FormatEpisodeName(ni.Node.Name(), ni.Node)
			}
		}
	},
}
