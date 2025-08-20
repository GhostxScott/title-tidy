package cmd

import (
	"os"
	"testing"

	"github.com/Digital-Shane/treeview"
	"github.com/google/go-cmp/cmp"
)

// helper to assert bool with message
func assertBool(t *testing.T, got bool, want bool, desc string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", desc, got, want)
	}
}

func TestCreateMediaFilter(t *testing.T) {
	t.Run("exclude junk files", func(t *testing.T) {
		f := CreateMediaFilter(false)
		// .DS_Store always filtered
		assertBool(t, f(testTreeviewFileInfo(".DS_Store", false)), false, "CreateMediaFilter(.DS_Store)")
		assertBool(t, f(testTreeviewFileInfo("._thumbs", false)), false, "CreateMediaFilter(._ prefix)")
	})
	t.Run("include directories when flag set", func(t *testing.T) {
		fInc := CreateMediaFilter(true)
		fExc := CreateMediaFilter(false)
		assertBool(t, fInc(testTreeviewFileInfo("dir", true)), true, "CreateMediaFilter(includeDirs).dir")
		assertBool(t, fExc(testTreeviewFileInfo("dir", true)), false, "CreateMediaFilter(!includeDirs).dir")
	})
	t.Run("video and subtitle filtering", func(t *testing.T) {
		f := CreateMediaFilter(false)
		assertBool(t, f(testTreeviewFileInfo("movie.mkv", false)), true, "video allowed")
		assertBool(t, f(testTreeviewFileInfo("episode.en.srt", false)), true, "subtitle allowed")
		assertBool(t, f(testTreeviewFileInfo("notes.txt", false)), false, "non media filtered")
	})
}

func TestUnwrapRoot(t *testing.T) {
	// single root directory => unwrap children (cloned to clear parent refs)
	rootDir := testNewDirNode("Root")
	childA := testNewFileNode("a.mkv")
	childB := testNewFileNode("b.srt")
	rootDir.AddChild(childA)
	rootDir.AddChild(childB)
	tr1 := testNewTree(rootDir)
	got := UnwrapRoot(tr1)
	if len(got) != 2 {
		t.Errorf("UnwrapRoot(single) returned %d nodes, want 2", len(got))
	}
	// Check that we got clones with same data but no parent
	if got[0].Name() != childA.Name() || got[0].Data().Name() != childA.Data().Name() {
		t.Errorf("UnwrapRoot(single) first node = %v, want clone of %v", got[0].Name(), childA.Name())
	}
	if got[1].Name() != childB.Name() || got[1].Data().Name() != childB.Data().Name() {
		t.Errorf("UnwrapRoot(single) second node = %v, want clone of %v", got[1].Name(), childB.Name())
	}
	// Verify parent references are cleared
	if got[0].Parent() != nil {
		t.Errorf("UnwrapRoot(single) first node still has parent reference")
	}
	if got[1].Parent() != nil {
		t.Errorf("UnwrapRoot(single) second node still has parent reference")
	}

	// multiple top nodes => unchanged
	a := testNewFileNode("a.mkv")
	b := testNewFileNode("b.mkv")
	tr2 := testNewTree(a, b)
	got2 := UnwrapRoot(tr2)
	if len(got2) != 2 || got2[0] != a || got2[1] != b {
		t.Errorf("UnwrapRoot(multi) = %v, want originals [%v %v]", got2, a, b)
	}
}

func TestSimpleFileInfo(t *testing.T) {
	d := &SimpleFileInfo{name: "Dir", isDir: true}
	if !d.IsDir() {
		t.Errorf("SimpleFileInfo.IsDir(dir) = false, want true")
	}
	if d.Name() != "Dir" {
		t.Errorf("SimpleFileFileInfo.Name() = %v, want %v", d.Name(), "Dir")
	}
	if d.Mode()&os.ModeDir == 0 {
		t.Errorf("SimpleFileInfo.Mode() missing directory bit: %v", d.Mode())
	}

	f := &SimpleFileInfo{name: "file.txt", isDir: false}
	if f.IsDir() {
		t.Errorf("SimpleFileInfo.IsDir(file) = true, want false")
	}
	if diff := cmp.Diff("file.txt", f.Name()); diff != "" {
		t.Errorf("SimpleFileInfo.Name() mismatch (-want +got)\n%s", diff)
	}
	if f.Mode()&os.ModeDir != 0 {
		t.Errorf("SimpleFileInfo.Mode() has directory bit set for file: %v", f.Mode())
	}
}

// Shared helpers for cmd package tests
func testNewFileNode(name string) *treeview.Node[treeview.FileInfo] {
	return treeview.NewNode(name, name, treeview.FileInfo{FileInfo: &SimpleFileInfo{name: name, isDir: false}, Path: name})
}
func testNewDirNode(name string) *treeview.Node[treeview.FileInfo] {
	return treeview.NewNode(name, name, treeview.FileInfo{FileInfo: &SimpleFileInfo{name: name, isDir: true}, Path: name})
}
func testNewTree(nodes ...*treeview.Node[treeview.FileInfo]) *treeview.Tree[treeview.FileInfo] {
	return treeview.NewTree(nodes)
}
func testTreeviewFileInfo(name string, isDir bool) treeview.FileInfo {
	return treeview.FileInfo{FileInfo: &SimpleFileInfo{name: name, isDir: isDir}, Path: name}
}
