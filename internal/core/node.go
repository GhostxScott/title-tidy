package core

import (
	"os"
	"time"
)

// SimpleFileInfo implements os.FileInfo for synthetic (virtual) nodes inserted
// into the tree (e.g. wrapping a standalone movie file in a virtual directory
// before materialization on disk).
type SimpleFileInfo struct {
	name  string
	isDir bool
}

func NewSimpleFileInfo(name string, isDir bool) *SimpleFileInfo {
	return &SimpleFileInfo{
		name:  name,
		isDir: isDir,
	}
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
