package cmd

import (
	"context"
	"github.com/Digital-Shane/title-tidy/internal/core"
	"github.com/Digital-Shane/title-tidy/internal/media"
	"github.com/Digital-Shane/treeview"
)

var MoviesCommand = CommandConfig{
	maxDepth:    2,
	includeDirs: true,
	movieMode:   true,
	preprocess:  MoviePreprocess,
	annotate:    MovieAnnotate,
}

// MoviePreprocess groups standalone movie video files (and matching subtitles) into
// virtual directories, so they can be materialized atomically during rename.
// Matching for subtitles: the filename prefix before language + subtitle suffix must
// exactly match the video filename without its extension.
func MoviePreprocess(nodes []*treeview.Node[treeview.FileInfo]) []*treeview.Node[treeview.FileInfo] {
	type bundle struct {
		dir *treeview.Node[treeview.FileInfo]
	}
	bundles := map[string]*bundle{} // base name (without extension) -> bundle
	var out []*treeview.Node[treeview.FileInfo]

	// First pass: wrap loose video files
	for _, n := range nodes {
		if n.Data().IsDir() || !media.IsVideo(n.Name()) {
			continue
		}
		base := n.Name()
		if ext := media.ExtractExtension(base); ext != "" {
			base = base[:len(base)-len(ext)]
		}
		if _, exists := bundles[base]; !exists {
			vd := treeview.NewNode(base, base, treeview.FileInfo{FileInfo: &SimpleFileInfo{name: base, isDir: true}, Path: base})
			vm := core.EnsureMeta(vd)
			vm.Type = core.MediaMovie
			vm.NewName = media.FormatShowName(base)
			vm.IsVirtual = true
			vm.NeedsDirectory = true
			bundles[base] = &bundle{dir: vd}
		}
		b := bundles[base]
		b.dir.AddChild(n)
		cm := core.EnsureMeta(n)
		cm.Type = core.MediaMovieFile
		cm.NewName = media.FormatShowName(base) + media.ExtractExtension(n.Name())
	}

	// Second pass: attach related subtitle files
	for _, n := range nodes {
		if n.Data().IsDir() || !media.IsSubtitle(n.Name()) {
			continue
		}
		suffix := media.ExtractSubtitleSuffix(n.Name())
		if suffix == "" { // defensive
			continue
		}
		base := n.Name()[:len(n.Name())-len(suffix)]
		if b, ok := bundles[base]; ok {
			b.dir.AddChild(n)
			sm := core.EnsureMeta(n)
			sm.Type = core.MediaMovieFile
			sm.NewName = media.FormatShowName(base) + suffix
		}
	}

	// Build final node list: virtual dirs + untouched originals
	used := map[*treeview.Node[treeview.FileInfo]]bool{}
	for _, b := range bundles {
		out = append(out, b.dir)
		used[b.dir] = true
		for _, c := range b.dir.Children() {
			used[c] = true
		}
	}
	for _, n := range nodes {
		if used[n] {
			continue
		}
		out = append(out, n)
	}
	return out
}

// MovieAnnotate adds metadata to any remaining movie directories / files not handled
// during preprocess (e.g., pre-existing movie directories from the filesystem).
func MovieAnnotate(t *treeview.Tree[treeview.FileInfo]) {
	for ni := range t.All(context.Background()) {
		if core.GetMeta(ni.Node) != nil { // already annotated
			continue
		}
		if ni.Depth == 0 && ni.Node.Data().IsDir() { // only treat directories as movie containers
			m := core.EnsureMeta(ni.Node)
			m.Type = core.MediaMovie
			m.NewName = media.FormatShowName(ni.Node.Name())
			continue
		}
		p := ni.Node.Parent()
		pm := core.GetMeta(p)
		if pm == nil || pm.NewName == "" {
			continue
		}
		m := core.EnsureMeta(ni.Node)
		m.Type = core.MediaMovieFile
		if media.IsSubtitle(ni.Node.Name()) {
			m.NewName = pm.NewName + media.ExtractSubtitleSuffix(ni.Node.Name())
		} else {
			m.NewName = pm.NewName + media.ExtractExtension(ni.Node.Name())
		}
	}
}
