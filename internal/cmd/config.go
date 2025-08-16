package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Digital-Shane/title-tidy/internal/core"
	"github.com/Digital-Shane/title-tidy/internal/media"
	"github.com/Digital-Shane/title-tidy/internal/tui"

	"github.com/Digital-Shane/treeview"
	tea "github.com/charmbracelet/bubbletea"
)

// CommandConfig describes how to construct and annotate a tree for a given subcommand. Fields:
//   - maxDepth: depth budget for filesystem enumeration.
//   - includeDirs: whether directory entries pass the filter.
//   - preprocess: optional in-memory node transformation prior to tree
//     construction (e.g. injecting virtual directories around loose movie files).
//   - annotate: optional pass to attach MediaMeta (type + proposed name).
//   - movieMode: toggles movie-oriented statistics & wording in the TUI.
//   - InstantMode: apply renames immediately without interactive preview.
//   - DeleteNFO: mark NFO files for deletion during rename.
//   - DeleteImages: mark image files for deletion during rename.
type CommandConfig struct {
	maxDepth     int
	includeDirs  bool
	preprocess   func([]*treeview.Node[treeview.FileInfo]) []*treeview.Node[treeview.FileInfo]
	annotate     func(*treeview.Tree[treeview.FileInfo])
	movieMode    bool
	InstantMode  bool
	DeleteNFO    bool
	DeleteImages bool
}

func RunCommand(cfg CommandConfig) error {
	// Build initial filesystem tree
	t, err := treeview.NewTreeFromFileSystem(context.Background(), ".", false,
		treeview.WithMaxDepth[treeview.FileInfo](cfg.maxDepth),
		treeview.WithFilterFunc(CreateMediaFilter(cfg.includeDirs)),
	)
	if err != nil {
		return err
	}

	// Unwrap root (avoid direct indexing & panic)
	nodes := UnwrapRoot(t)
	if cfg.preprocess != nil {
		nodes = cfg.preprocess(nodes)
	}

	t = treeview.NewTree(nodes,
		treeview.WithExpandAll[treeview.FileInfo](),
		treeview.WithProvider(tui.CreateRenameProvider()),
	)
	if cfg.annotate != nil {
		cfg.annotate(t)
	}

	// Mark files for deletion based on flags
	MarkFilesForDeletion(t, cfg.DeleteNFO, cfg.DeleteImages)

	// Create model
	model := tui.NewRenameModel(t)
	model.IsMovieMode = cfg.movieMode
	model.DeleteNFO = cfg.DeleteNFO
	model.DeleteImages = cfg.DeleteImages

	// If instant mode, perform renames immediately
	if cfg.InstantMode {
		cmd := model.PerformRenames()
		if cmd != nil {
			msg := cmd()
			if result, ok := msg.(tui.RenameCompleteMsg); ok {
				if result.ErrorCount() > 0 {
					return fmt.Errorf("%d errors occurred during renaming", result.ErrorCount())
				}
			}
		}
		return nil
	}

	// Launch TUI
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()
	return err
}

// CreateMediaFilter returns a filter function that excludes common junk files
// and optionally filters for specific file types based on the includeDirectories parameter.
func CreateMediaFilter(includeDirectories bool) func(info treeview.FileInfo) bool {
	return func(info treeview.FileInfo) bool {
		if info.Name() == ".DS_Store" || strings.HasPrefix(info.Name(), "._") {
			return false
		}
		if includeDirectories {
			return info.IsDir() || media.IsSubtitle(info.Name()) || media.IsVideo(info.Name()) || media.IsNFO(info.Name()) || media.IsImage(info.Name())
		}
		return media.IsSubtitle(info.Name()) || media.IsVideo(info.Name()) || media.IsNFO(info.Name()) || media.IsImage(info.Name())
	}
}

// UnwrapRoot returns children of single root directory, otherwise original nodes
func UnwrapRoot(t *treeview.Tree[treeview.FileInfo]) []*treeview.Node[treeview.FileInfo] {
	ns := t.Nodes()
	if len(ns) == 1 && ns[0].Data().IsDir() {
		return ns[0].Children()
	}
	return ns
}

// SimpleFileInfo implements os.FileInfo for synthetic (virtual) nodes inserted
// into the tree (e.g. wrapping a standalone movie file in a virtual directory
// before materialization on disk).
type SimpleFileInfo struct {
	name  string
	isDir bool
}

func (m *SimpleFileInfo) Name() string { return m.name }
func (m *SimpleFileInfo) Size() int64  { return 0 }
func (m *SimpleFileInfo) Mode() os.FileMode {
	if m.isDir {
		return os.ModeDir | 0755
	}
	return 0644
}
func (m *SimpleFileInfo) ModTime() time.Time { return time.Now() }
func (m *SimpleFileInfo) IsDir() bool        { return m.isDir }
func (m *SimpleFileInfo) Sys() any           { return nil }

// MarkFilesForDeletion traverses the tree and marks NFO and/or image files for deletion
func MarkFilesForDeletion(t *treeview.Tree[treeview.FileInfo], deleteNFO, deleteImages bool) {
	if !deleteNFO && !deleteImages {
		return
	}

	for ni := range t.All(context.Background()) {
		if ni.Node.Data().IsDir() {
			continue
		}

		filename := ni.Node.Name()
		shouldDelete := false

		if deleteNFO && media.IsNFO(filename) {
			shouldDelete = true
		}

		if deleteImages && media.IsImage(filename) {
			shouldDelete = true
		}

		if shouldDelete {
			meta := core.EnsureMeta(ni.Node)
			meta.MarkedForDeletion = true
		}
	}
}
