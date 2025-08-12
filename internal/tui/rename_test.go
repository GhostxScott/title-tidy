package tui

import (
	"github.com/Digital-Shane/title-tidy/internal/core"
	"os"
	"path/filepath"
	"testing"

	"github.com/Digital-Shane/treeview"
)

// fsTestNode creates a node representing a filesystem entry; path provided explicitly.
func fsTestNode(name string, isDir bool, path string) *treeview.Node[treeview.FileInfo] {
	fi := core.NewSimpleFileInfo(name, isDir)
	return treeview.NewNode(name, name, treeview.FileInfo{FileInfo: fi, Path: path})
}

func TestRenameRegular_NoChange(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "same.txt"), []byte("data"), 0644); err != nil {
		t.Fatalf("prep: %v", err)
	}
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(tmp)
	n := fsTestNode("same.txt", false, "same.txt")
	mm := core.EnsureMeta(n)
	mm.NewName = "same.txt" // identical
	renamed, err := RenameRegular(n, mm)
	if err != nil || renamed {
		t.Errorf("renameRegular(identical) = (%v,%v), want (false,<nil>)", renamed, err)
	}
	if mm.RenameStatus != core.RenameStatusNone {
		t.Errorf("renameRegular(identical) status = %v, want %v", mm.RenameStatus, core.RenameStatusNone)
	}
}

func TestRenameRegular_DestinationExists(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(tmp)
	if err := os.WriteFile("src.txt", []byte("src"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("dest.txt", []byte("dest"), 0644); err != nil {
		t.Fatal(err)
	}
	n := fsTestNode("src.txt", false, "src.txt")
	mm := core.EnsureMeta(n)
	mm.NewName = "dest.txt"
	renamed, err := RenameRegular(n, mm)
	if err == nil || renamed {
		t.Errorf("renameRegular(destExists) = (%v,%v), want (false,error)", renamed, err)
	}
	if mm.RenameStatus != core.RenameStatusError || mm.RenameError == "" {
		t.Errorf("renameRegular(destExists) meta = %+v, want error status with message", mm)
	}
	if n.Data().Path != "src.txt" {
		t.Errorf("renameRegular(destExists) path = %s, want src.txt", n.Data().Path)
	}
}

func TestRenameRegular_SourceMissingCausesError(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(tmp)
	n := fsTestNode("src.txt", false, "src.txt")
	mm := core.EnsureMeta(n)
	mm.NewName = "renamed.txt"
	renamed, err := RenameRegular(n, mm)
	if err == nil || renamed {
		t.Errorf("renameRegular(missingSource) = (%v,%v), want (false,error)", renamed, err)
	}
	if mm.RenameStatus != core.RenameStatusError {
		t.Errorf("renameRegular(missingSource) status = %v, want %v", mm.RenameStatus, core.RenameStatusError)
	}
}

func TestRenameRegular_Success(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(tmp)
	if err := os.WriteFile("orig.txt", []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	n := fsTestNode("orig.txt", false, "orig.txt")
	mm := core.EnsureMeta(n)
	mm.NewName = "new.txt"
	renamed, err := RenameRegular(n, mm)
	if err != nil || !renamed {
		t.Errorf("renameRegular(success) = (%v,%v), want (true,<nil>)", renamed, err)
	}
	if mm.RenameStatus != core.RenameStatusSuccess {
		t.Errorf("renameRegular(success) status = %v, want %v", mm.RenameStatus, core.RenameStatusSuccess)
	}
	if n.Data().Path != "new.txt" {
		t.Errorf("renameRegular(success) path = %s, want new.txt", n.Data().Path)
	}
	if _, err := os.Stat("new.txt"); err != nil {
		t.Errorf("renameRegular(success) new file stat error = %v", err)
	}
}

func TestCreateVirtualDir_MkdirFails(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(tmp)
	if err := os.Mkdir("Already", 0755); err != nil {
		t.Fatal(err)
	}
	n := fsTestNode("virtual", true, "virtual")
	mm := core.EnsureMeta(n)
	mm.NewName = "Already"
	mm.IsVirtual = true
	mm.NeedsDirectory = true
	successes, errs := CreateVirtualDir(n, mm)
	if successes != 0 || len(errs) != 1 {
		t.Errorf("createVirtualDir(mkdirFail) = (%d,%d errs), want (0,1)", successes, len(errs))
	}
	if mm.RenameStatus != core.RenameStatusError {
		t.Errorf("createVirtualDir(mkdirFail) status = %v, want %v", mm.RenameStatus, core.RenameStatusError)
	}
	if n.Data().Path == "./Already" {
		t.Errorf("createVirtualDir(mkdirFail) path updated unexpectedly")
	}
}

func TestCreateVirtualDir_SuccessChildrenMixed(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(tmp)
	if err := os.WriteFile("child1.mkv", []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("child2.mkv", []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("childSkip.mkv", []byte("c"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove("child2.mkv"); err != nil {
		t.Fatal(err)
	} // remove to force failure
	vdir := fsTestNode("virt-old", true, "virt-old")
	mm := core.EnsureMeta(vdir)
	mm.NewName = "Movie Name"
	mm.IsVirtual = true
	mm.NeedsDirectory = true
	c1 := fsTestNode("child1.mkv", false, "child1.mkv")
	cm1 := core.EnsureMeta(c1)
	cm1.NewName = "Renamed1.mkv"
	c2 := fsTestNode("child2.mkv", false, "child2.mkv")
	cm2 := core.EnsureMeta(c2)
	cm2.NewName = "Renamed2.mkv"                              // will fail
	c3 := fsTestNode("childSkip.mkv", false, "childSkip.mkv") // no metadata new name => skipped
	vdir.AddChild(c1)
	vdir.AddChild(c2)
	vdir.AddChild(c3)
	successes, errs := CreateVirtualDir(vdir, mm)
	if successes != 2 || len(errs) != 1 {
		t.Errorf("createVirtualDir(mixed) counts = (%d successes,%d errs), want (2,1)", successes, len(errs))
	}
	if mm.RenameStatus != core.RenameStatusSuccess {
		t.Errorf("createVirtualDir(mixed) dir status = %v, want %v", mm.RenameStatus, core.RenameStatusSuccess)
	}
	if cm1.RenameStatus != core.RenameStatusSuccess {
		t.Errorf("createVirtualDir(mixed) child1 status = %v, want %v", cm1.RenameStatus, core.RenameStatusSuccess)
	}
	if cm2.RenameStatus != core.RenameStatusError {
		t.Errorf("createVirtualDir(mixed) child2 status = %v, want %v", cm2.RenameStatus, core.RenameStatusError)
	}
	if c1.Data().Path != "Movie Name/Renamed1.mkv" {
		t.Errorf("createVirtualDir(mixed) child1 path = %s, want Movie Name/Renamed1.mkv", c1.Data().Path)
	}
	if c2.Data().Path != "child2.mkv" {
		t.Errorf("createVirtualDir(mixed) child2 path = %s, want child2.mkv", c2.Data().Path)
	}
	if vdir.Data().Path != "Movie Name" {
		t.Errorf("createVirtualDir(mixed) dir path = %s, want Movie Name", vdir.Data().Path)
	}
}
