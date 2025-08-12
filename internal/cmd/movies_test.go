package cmd

import (
	"testing"

	"github.com/Digital-Shane/treeview"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/Digital-Shane/title-tidy/internal/core"
)

func TestMoviePreprocess_GroupsLooseFiles(t *testing.T) {
	video1 := testNewFileNode("movie.one.mkv")
	sub1 := testNewFileNode("movie.one.en.srt")
	video2 := testNewFileNode("second.mp4")
	straySub := testNewFileNode("orphan.en.srt")
	existingDir := testNewDirNode("ExistingMovieDir")

	nodes := []*treeview.Node[treeview.FileInfo]{video1, sub1, video2, straySub, existingDir}
	out := MoviePreprocess(nodes)

	var virtuals []*treeview.Node[treeview.FileInfo]
	present := map[*treeview.Node[treeview.FileInfo]]bool{}
	for _, n := range out {
		present[n] = true
		if m := core.GetMeta(n); m != nil && m.IsVirtual {
			virtuals = append(virtuals, n)
		}
	}

	if len(virtuals) != 2 {
		t.Fatalf("MoviePreprocess virtual count = %d, want 2", len(virtuals))
	}

	var bundle1, bundle2 *treeview.Node[treeview.FileInfo]
	for _, v := range virtuals {
		switch v.Name() {
		case "movie.one":
			bundle1 = v
		case "second":
			bundle2 = v
		}
	}
	if bundle1 == nil || bundle2 == nil {
		t.Fatalf("MoviePreprocess missing expected virtual dirs: movie.one=%v second=%v", bundle1, bundle2)
	}

	if len(bundle1.Children()) != 2 {
		t.Errorf("bundle1.Children len = %d, want 2", len(bundle1.Children()))
	}
	for _, c := range bundle1.Children() {
		mm := core.GetMeta(c)
		if mm == nil || mm.Type != core.MediaMovieFile || mm.NewName == "" {
			t.Errorf("bundle1 child meta = %#v, want populated MediaMovieFile", mm)
		}
	}
	vm1 := core.GetMeta(bundle1)
	if vm1 == nil || vm1.Type != core.MediaMovie || !vm1.IsVirtual || !vm1.NeedsDirectory {
		t.Errorf("bundle1 meta = %#v, want movie virtual NeedsDirectory", vm1)
	}

	if len(bundle2.Children()) != 1 {
		t.Errorf("bundle2.Children len = %d, want 1", len(bundle2.Children()))
	}

	if present[video1] || present[sub1] || present[video2] {
		t.Errorf("grouped file nodes unexpectedly still present at top-level")
	}
	if !present[straySub] {
		t.Errorf("orphan subtitle dropped; expected to remain at top-level")
	}
	if !present[existingDir] {
		t.Errorf("existing directory missing from output")
	}
}

func TestMovieAnnotate_ExistingDirectory(t *testing.T) {
	dir := testNewDirNode("Some.Movie.2024.1080p")
	vid := testNewFileNode("Some.Movie.2024.1080p.mkv")
	sub := testNewFileNode("Some.Movie.2024.1080p.en.srt")
	dir.AddChild(vid)
	dir.AddChild(sub)
	tr := testNewTree(dir)

	MovieAnnotate(tr)

	dm := core.GetMeta(dir)
	if dm == nil || dm.Type != core.MediaMovie || dm.NewName == "" {
		t.Fatalf("MovieAnnotate dir meta = %#v, want movie with NewName", dm)
	}
	vm := core.GetMeta(vid)
	sm := core.GetMeta(sub)
	if vm == nil || vm.Type != core.MediaMovieFile || vm.NewName == "" {
		t.Errorf("video meta = %#v, want movie file name", vm)
	}
	if sm == nil || sm.Type != core.MediaMovieFile || sm.NewName == "" {
		t.Errorf("subtitle meta = %#v, want movie file name", sm)
	}
	if sm != nil && vm != nil && cmp.Equal(vm.NewName, sm.NewName, cmpopts.EquateEmpty()) {
		t.Errorf("subtitle NewName %q equals video NewName %q; expected language+ext suffix", sm.NewName, vm.NewName)
	}
}
