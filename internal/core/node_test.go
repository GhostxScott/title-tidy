package core

import (
	"os"
	"testing"
	"time"
)

func TestSimpleFileInfo(t *testing.T) {
	// Test construction and basic properties
	fileInfo := NewSimpleFileInfo("document.pdf", false)
	if fileInfo == nil {
		t.Fatal("NewSimpleFileInfo returned nil")
	}
	
	dirInfo := NewSimpleFileInfo("testdir", true)
	if dirInfo == nil {
		t.Fatal("NewSimpleFileInfo returned nil for directory")
	}
	
	// Test Name method
	if got := fileInfo.Name(); got != "document.pdf" {
		t.Errorf("Name() = %q, want %q", got, "document.pdf")
	}
	if got := dirInfo.Name(); got != "testdir" {
		t.Errorf("Name() = %q, want %q", got, "testdir")
	}
	
	// Test with special characters and empty name
	specialInfo := NewSimpleFileInfo("file-with_special.chars", false)
	if got := specialInfo.Name(); got != "file-with_special.chars" {
		t.Errorf("Name() with special chars = %q, want %q", got, "file-with_special.chars")
	}
	
	emptyInfo := NewSimpleFileInfo("", false)
	if got := emptyInfo.Name(); got != "" {
		t.Errorf("Name() with empty string = %q, want %q", got, "")
	}
	
	// Test Size method (always returns 0)
	if got := fileInfo.Size(); got != 0 {
		t.Errorf("Size() = %d, want 0", got)
	}
	if got := dirInfo.Size(); got != 0 {
		t.Errorf("Size() = %d, want 0", got)
	}
	
	// Test Mode method
	if got := fileInfo.Mode(); got != 0644 {
		t.Errorf("Mode() for file = %v, want %v", got, os.FileMode(0644))
	}
	if got := dirInfo.Mode(); got != (os.ModeDir | 0755) {
		t.Errorf("Mode() for dir = %v, want %v", got, os.ModeDir|0755)
	}
	
	// Test ModTime method
	testInfo := NewSimpleFileInfo("test", false)
	modTime := testInfo.ModTime()
	
	// ModTime should be close to now (within 1 second)
	since := time.Since(modTime)
	if since < 0 || since > time.Second {
		t.Errorf("ModTime() = %v, want recent time (difference: %v)", modTime, since)
	}
	
	// Test IsDir method
	if fileInfo.IsDir() {
		t.Error("IsDir() for file = true, want false")
	}
	if !dirInfo.IsDir() {
		t.Error("IsDir() for directory = false, want true")
	}
	
	// Test Sys method (always returns nil)
	if got := fileInfo.Sys(); got != nil {
		t.Errorf("Sys() = %v, want nil", got)
	}
	if got := dirInfo.Sys(); got != nil {
		t.Errorf("Sys() = %v, want nil", got)
	}
}