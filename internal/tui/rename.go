package tui

import (
	"fmt"
	"github.com/Digital-Shane/title-tidy/internal/core"
	"os"
	"path/filepath"

	"github.com/Digital-Shane/treeview"
)

// RenameCompleteMsg is emitted once performRenames finishes walking the tree.
type RenameCompleteMsg struct{ successCount, errorCount int }

// SuccessCount returns the number of successful renames
func (r RenameCompleteMsg) SuccessCount() int { return r.successCount }

// ErrorCount returns the number of errors during renames
func (r RenameCompleteMsg) ErrorCount() int { return r.errorCount }

// RenameRegular renames a node; returns true only when an actual filesystem rename occurred.
func RenameRegular(node *treeview.Node[treeview.FileInfo], mm *core.MediaMeta) (bool, error) {
	oldPath := node.Data().Path
	newPath := filepath.Join(filepath.Dir(oldPath), mm.NewName)
	if oldPath == newPath {
		return false, nil
	}
	if _, err := os.Stat(newPath); err == nil {
		return false, mm.Fail(fmt.Errorf("destination already exists"))
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return false, mm.Fail(err)
	}
	mm.Success()
	node.Data().Path = newPath
	return true, nil
}

// CreateVirtualDir materializes a virtual movie directory then renames its children beneath it.
//
// Returns a count of successful operations (directory creation + child renames), and contextual errors
func CreateVirtualDir(node *treeview.Node[treeview.FileInfo], mm *core.MediaMeta) (int, []error) {
	successes := 0
	errs := []error{}

	dirPath := filepath.Join(".", mm.NewName)
	if err := os.Mkdir(dirPath, 0755); err != nil {
		errs = append(errs, fmt.Errorf("create %s: %w", mm.NewName, mm.Fail(err)))
		return successes, errs
	}
	// Directory created successfully
	successes++
	mm.Success()
	node.Data().Path = dirPath

	// Rename children into the new directory
	for _, child := range node.Children() {
		cm := core.GetMeta(child)
		if cm == nil || cm.NewName == "" {
			continue
		}
		oldChildPath := child.Data().Path
		newChildPath := filepath.Join(dirPath, cm.NewName)
		if err := os.Rename(oldChildPath, newChildPath); err != nil {
			errs = append(errs, fmt.Errorf("%s -> %s: %w", child.Name(), cm.NewName, cm.Fail(err)))
			continue
		}
		successes++
		cm.Success()
		child.Data().Path = newChildPath
	}
	return successes, errs
}
