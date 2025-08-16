package tui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Digital-Shane/title-tidy/internal/core"
	"github.com/Digital-Shane/title-tidy/internal/media"

	"github.com/Digital-Shane/treeview"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Cached base styles (applied with dynamic Width each render) to avoid
// re-allocating identical style pipelines on every View() call.
var (
	headerStyleBase = lipgloss.NewStyle().
			Bold(true).
			Background(colorPrimary).
			Foreground(colorBackground).
			Align(lipgloss.Center)

	statusStyleBase = lipgloss.NewStyle().
			Background(colorSecondary).
			Foreground(colorBackground).
			Padding(0, 1)
)

// RenameModel wraps the underlying treeview TUI model to add media rename
// functionality and real‑time statistics.
type RenameModel struct {
	*treeview.TuiTreeModel[treeview.FileInfo]
	renameInProgress bool
	renameComplete   bool
	successCount     int
	errorCount       int
	width            int
	height           int
	IsMovieMode      bool
	DeleteNFO        bool
	DeleteImages     bool

	// Layout metrics
	treeWidth   int
	treeHeight  int
	statsWidth  int
	statsHeight int

	// Stat tracking
	statsCache Statistics
	statsDirty bool
}

// NewRenameModel returns an initialized RenameModel for the provided tree with
// default dimensions (later adjusted on the first WindowSize message).
func NewRenameModel(tree *treeview.Tree[treeview.FileInfo]) *RenameModel {
	m := &RenameModel{
		width:      80,
		height:     24,
		statsDirty: true,
	}
	// establish initial layout metrics before building underlying model
	m.CalculateLayout()
	m.TuiTreeModel = m.createSizedTuiModel(tree)
	return m
}

// CalculateLayout recomputes panel dimensions from current window size.
func (m *RenameModel) CalculateLayout() {
	// Set tree width to 60%
	tw := m.width * 6 / 10
	// Add virtual space for header, status, and white space
	th := m.height - 3
	// Ensure min height
	if th < 5 {
		th = 5
	}
	m.treeWidth = tw
	m.treeHeight = th
	// Stats panel uses remaining width; rounding differences fall to stats.
	m.statsWidth = m.width - tw
	// Height subtracts 2 additional lines to compensate for the rounded border.
	// (Previously m.height-5 == (m.height-3)-2.)
	m.statsHeight = th - 2
	// ensure a minimal positive stats height
	if m.statsHeight < 1 {
		m.statsHeight = 1
	}
}

// createSizedTuiModel builds a tree model sized to current dimensions and
// disables treeview features (search/reset) not needed for this application.
func (m *RenameModel) createSizedTuiModel(tree *treeview.Tree[treeview.FileInfo]) *treeview.TuiTreeModel[treeview.FileInfo] {
	// Create custom key map without search and reset
	keyMap := treeview.DefaultKeyMap()
	keyMap.SearchStart = []string{} // Disable search
	keyMap.Reset = []string{}       // Disable ctrl+r reset

	return treeview.NewTuiTreeModel(tree,
		treeview.WithTuiWidth[treeview.FileInfo](m.treeWidth),
		treeview.WithTuiHeight[treeview.FileInfo](m.treeHeight),
		treeview.WithTuiAllowResize[treeview.FileInfo](true),
		treeview.WithTuiDisableNavBar[treeview.FileInfo](true),
		treeview.WithTuiKeyMap[treeview.FileInfo](keyMap),
	)
}

// Init initializes the embedded tree model and requests an initial window size.
func (m *RenameModel) Init() tea.Cmd {
	return tea.Batch(
		m.TuiTreeModel.Init(),
		tea.WindowSize(),
	)
}

// Update handles Bubble Tea messages (resize, key events, internal completion).
func (m *RenameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle window size changes
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Record full window size
		m.width = msg.Width
		m.height = msg.Height
		// Recalculate layout metrics once, then forward scaled size to tree model
		m.CalculateLayout()
		internalMsg := tea.WindowSizeMsg{Width: m.treeWidth, Height: m.treeHeight}
		updated, cmd := m.TuiTreeModel.Update(internalMsg)
		if tm, ok := updated.(*treeview.TuiTreeModel[treeview.FileInfo]); ok {
			m.TuiTreeModel = tm
		}
		return m, cmd

	case tea.KeyMsg:
		// Handle custom keys before passing to tree model
		switch msg.String() {
		case "r":
			if !m.renameInProgress {
				m.renameInProgress = true
				return m, m.PerformRenames()
			}
		case "pgup":
			// Page up - move up by viewport height
			pageSize := m.treeHeight
			if pageSize <= 0 {
				pageSize = 10
			}
			m.TuiTreeModel.Tree.Move(context.Background(), -pageSize)
			return m, nil
		case "pgdown":
			// Page down - move down by viewport height
			pageSize := m.treeHeight
			if pageSize <= 0 {
				pageSize = 10
			}
			m.TuiTreeModel.Tree.Move(context.Background(), pageSize)
			return m, nil
		}
	
	case tea.MouseMsg:
		// Handle mouse wheel scrolling
		switch msg.Type {
		case tea.MouseWheelUp:
			// Scroll up by 1 line
			m.TuiTreeModel.Tree.Move(context.Background(), -1)
			return m, nil
		case tea.MouseWheelDown:
			// Scroll down by 1 line
			m.TuiTreeModel.Tree.Move(context.Background(), 1)
			return m, nil
		}

	case RenameCompleteMsg:
		m.renameInProgress = false
		m.renameComplete = true
		m.successCount = msg.successCount
		m.errorCount = msg.errorCount
		m.statsDirty = true
		return m, nil
	}

	// Pass through to embedded tree model for other messages
	updatedModel, cmd := m.TuiTreeModel.Update(msg)
	if tm, ok := updatedModel.(*treeview.TuiTreeModel[treeview.FileInfo]); ok {
		m.TuiTreeModel = tm
	}

	return m, cmd
}

// View returns the full TUI string (header, tree+stats layout, status bar).
func (m *RenameModel) View() string {
	var b strings.Builder

	// Render header
	b.WriteString(m.renderHeader())
	b.WriteByte('\n')

	// Stats Panel
	b.WriteString(m.renderTwoPanelLayout())
	b.WriteByte('\n')

	// Render integrated status bar
	b.WriteString(m.renderStatusBar())
	return b.String()
}

// renderHeader creates the single‑line header bar with mode + working directory.
func (m *RenameModel) renderHeader() string {
	style := headerStyleBase.Width(m.width)

	path, _ := os.Getwd()
	var title string
	if m.IsMovieMode {
		title = fmt.Sprintf("🎬 Movie Rename - %s", path)
	} else {
		title = fmt.Sprintf("📺 TV Show Rename - %s", path)
	}
	return style.Render(title)
}

// renderStatusBar renders a single line of key hints and actions.
func (m *RenameModel) renderStatusBar() string {
	style := statusStyleBase.Width(m.width)

	statusText := "↑↓: Navigate  PgUp/PgDn: Page  ←→: Expand/Collapse  │  r: Rename  │  esc: Quit"
	return style.Render(statusText)
}

// renderTwoPanelLayout joins the tree view and statistics panel horizontally.
func (m *RenameModel) renderTwoPanelLayout() string {
	statsPanel := m.renderStatsPanel()
	treeView := m.TuiTreeModel.View()

	return lipgloss.JoinHorizontal(lipgloss.Top, treeView, statsPanel)
}

// renderStatsPanel builds and formats the statistics panel content.
func (m *RenameModel) renderStatsPanel() string {
	style := lipgloss.NewStyle().
		Width(m.statsWidth - 6).
		Height(m.statsHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(1)

	stats := m.calculateStats()
	var b strings.Builder
	b.Grow(512)

	b.WriteString("📊 Statistics\n\n")
	b.WriteString("Files Found:\n")
	if m.IsMovieMode {
		fmt.Fprintf(&b, "  🎬 Movies:      %d\n", stats.movieCount)
		fmt.Fprintf(&b, "  🎥 Video Files: %d\n", stats.movieFileCount-stats.subtitleCount)
		fmt.Fprintf(&b, "  📄 Subtitles:   %d\n", stats.subtitleCount)
	} else {
		fmt.Fprintf(&b, "  📺 TV Shows:    %d\n", stats.showCount)
		fmt.Fprintf(&b, "  📁 Seasons:     %d\n", stats.seasonCount)
		fmt.Fprintf(&b, "  🎬 Episodes:    %d\n", stats.episodeCount)
		fmt.Fprintf(&b, "  📄 Subtitles:   %d\n", stats.subtitleCount)
	}

	b.WriteString("\nRename Status:\n")
	fmt.Fprintf(&b, "  ✓ Need rename:  %d\n", stats.needRenameCount)
	fmt.Fprintf(&b, "  = No change:    %d\n", stats.noChangeCount)
	if stats.toDeleteCount > 0 {
		fmt.Fprintf(&b, "  🗑 To delete:    %d\n", stats.toDeleteCount)
	}

	if stats.successCount > 0 || stats.errorCount > 0 {
		b.WriteString("\nLast Operation:\n")
		if stats.successCount > 0 {
			fmt.Fprintf(&b, "  ✅ Success:     %d\n", stats.successCount)
		}
		if stats.errorCount > 0 {
			fmt.Fprintf(&b, "  ❌ Errors:      %d\n", stats.errorCount)
		}
	}

	var totalItems int
	if m.IsMovieMode {
		totalItems = stats.movieCount + stats.movieFileCount
	} else {
		totalItems = stats.showCount + stats.seasonCount + stats.episodeCount + stats.subtitleCount
	}

	fmt.Fprintf(&b, "\nTotal items: %d\n", totalItems)
	if totalItems > 0 {
		percentNeedRename := (stats.needRenameCount * 100) / totalItems
		fmt.Fprintf(&b, "Need rename: %d%%", percentNeedRename)
	}

	return style.Render(b.String())
}

// Statistics aggregates counts derived from the current tree plus the most
// recent batch rename operation.
//
// Fields:
//   - showCount / seasonCount / episodeCount: counts of TV hierarchy nodes.
//   - movieCount / movieFileCount: counts for movie mode (directories & files).
//   - subtitleCount: number of subtitle files (subset of episode/movie files).
//   - needRenameCount: nodes where NewName differs from current name.
//   - noChangeCount: nodes with a proposed name identical to current name.
//   - successCount / errorCount: results from the last performRenames run.
//   - toDeleteCount: nodes marked for deletion.
type Statistics struct {
	showCount       int
	seasonCount     int
	episodeCount    int
	subtitleCount   int
	movieCount      int
	movieFileCount  int
	needRenameCount int
	noChangeCount   int
	successCount    int
	errorCount      int
	toDeleteCount   int
}

// calculateStats walks the tree to produce aggregate counts while preserving
// previously recorded success/error tallies from the last rename operation.
func (m *RenameModel) calculateStats() Statistics {
	// Fast path: return cached stats if still valid
	if !m.statsDirty {
		// always ensure latest success/error counts reflected even if cache reused
		m.statsCache.successCount = m.successCount
		m.statsCache.errorCount = m.errorCount
		return m.statsCache
	}

	stats := Statistics{}
	for nodeInfo := range m.Tree.All(context.Background()) {
		node := nodeInfo.Node
		mm := core.GetMeta(node)
		if mm == nil {
			continue
		}
		switch mm.Type {
		case core.MediaShow:
			stats.showCount++
		case core.MediaSeason:
			stats.seasonCount++
		case core.MediaEpisode:
			stats.episodeCount++
		case core.MediaMovie:
			stats.movieCount++
		case core.MediaMovieFile:
			stats.movieFileCount++
		}
		if !node.Data().IsDir() && media.IsSubtitle(node.Data().Name()) {
			stats.subtitleCount++
		}
		if mm.MarkedForDeletion {
			stats.toDeleteCount++
		} else if mm.NewName != "" {
			if mm.NewName != node.Name() {
				stats.needRenameCount++
			} else {
				stats.noChangeCount++
			}
		}
	}
	stats.successCount = m.successCount
	stats.errorCount = m.errorCount
	m.statsCache = stats
	m.statsDirty = false
	return stats
}

// PerformRenames walks the tree bottom‑up executing pending rename operations.
// It skips children of virtual directories (handled by the virtual parent) and
// aggregates success / error counts into a renameCompleteMsg.
func (m *RenameModel) PerformRenames() tea.Cmd {
	return func() tea.Msg {
		var successCount int
		var errs []error
		for nodeInfo, iterErr := range m.Tree.AllBottomUp(context.Background()) {
			if iterErr != nil {
				errs = append(errs, fmt.Errorf("iterator: %w", iterErr))
				break
			}
			node := nodeInfo.Node
			mm := core.GetMeta(node)
			
			// Handle file deletion if marked
			if mm != nil && mm.MarkedForDeletion {
				if err := os.Remove(node.Data().Path); err != nil {
					mm.Fail(err)
					errs = append(errs, err)
				} else {
					mm.Success()
					successCount++
				}
				continue
			}
			
			if mm == nil || mm.NewName == "" {
				continue
			}
			// Skip children inside virtual containers (handled by createVirtualDir)
			if parent := node.Parent(); parent != nil {
				if pm := core.GetMeta(parent); pm != nil && pm.IsVirtual {
					continue
				}
			}

			// Virtual directory creation
			if mm.NeedsDirectory && mm.IsVirtual {
				s, vErrs := CreateVirtualDir(node, mm)
				successCount += s
				errs = append(errs, vErrs...)
				continue
			}

			// Regular rename
			if renamed, err := RenameRegular(node, mm); err != nil {
				errs = append(errs, err)
			} else if renamed {
				successCount++
			}
		}
		return RenameCompleteMsg{
			successCount: successCount,
			errorCount:   len(errs),
		}
	}
}
