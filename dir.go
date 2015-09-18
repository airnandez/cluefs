package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

// This is a temporal fix: don't rewrite the targets of symbolic links by
// default. Some applications check when they create a symbolic link that
// the value they specified as target is actually the one the file system uses.
// TODO: this should be a configurable feature for the ClueFS.
const rewriteSymlinkTargets = false

var skipDirEntry func(n string) bool

func init() {
	switch runtime.GOOS {
	case "darwin":
		// On Darwin we skip all directory entries starting by '._'
		skipDirEntry = func(n string) bool {
			return strings.HasPrefix(n, "._")
		}
	default:
		skipDirEntry = func(n string) bool {
			return false
		}
	}
}

type Dir struct {
	*Node
	*Handle
	ProcessInfo

	// mutex protects the entries map
	mutex sync.RWMutex

	// entries maps the name of a node with its node object
	entries map[string]interface{}
}

func NewDir(parent string, name string, fs *ClueFS) *Dir {
	return &Dir{
		Node:    NewNode(parent, name, fs),
		Handle:  &Handle{},
		entries: make(map[string]interface{}, 32),
	}
}

func (d Dir) String() string {
	return fmt.Sprintf("[%s %s %s]", d.Node, d.Handle, d.ProcessInfo)
}

func (d *Dir) SetProcessInfo(h fuse.Header) {
	d.ProcessInfo = ProcessInfo{Uid: h.Uid, Gid: h.Gid, Pid: h.Pid}
}

// saveEntry saves a *File or *Dir object associated to a
// name in this directory
func (d *Dir) saveEntry(name string, entry interface{}) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.entries[name] = entry
}

// dropEntry deletes the *File or *Dir object associated to a
// name in this directory
func (d *Dir) dropEntry(name string) interface{} {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if e := d.entries[name]; e != nil {
		delete(d.entries, name)
		return e
	}
	return nil
}

// getEntry returns the *File or *Dir object currently associated
// to a name in this directory. It may return nil if there is
// no such object associate to name
func (d *Dir) getEntry(name string) interface{} {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.entries[name]
}

func (d *Dir) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fusefs.Handle, error) {
	op := NewOpenOp(req, d.path)
	defer trace(op)
	newdir := NewDir(d.parent, d.name, d.fs)
	size, err := newdir.doOpen(d.path, req.Flags)
	if err != nil {
		return nil, err
	}
	newdir.SetProcessInfo(req.Header)
	resp.Handle = fuse.HandleID(newdir.handleID)
	op.FileSize = size
	op.BlockSize = newdir.blksize
	op.OpenID = newdir.handleID
	return newdir, nil
}

func (d *Dir) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	if !d.isOpen() {
		return nil
	}
	defer trace(NewReleaseOp(req, d.path, d.handleID))
	if req.ReleaseFlags&fuse.ReleaseFlush != 0 {
		d.doSync()
	}
	return d.doClose()
}

func (d *Dir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fusefs.Node, error) {
	if skipDirEntry(req.Name) {
		return nil, fuse.ENOENT
	}
	path := filepath.Join(d.path, req.Name)
	isDir := false
	defer trace(NewLookupOp(req, path, isDir))
	var st syscall.Stat_t
	if err := syscall.Lstat(path, &st); err != nil {
		return nil, fuse.ENOENT
	}
	resp.Attr = statToFuseAttr(st)
	resp.Node = fuse.NodeID(resp.Attr.Inode)
	// TODO: should we overwrite resp.EntryValid?
	// resp.EntryValid = time.Duration(500) * time.Millisecond

	// Is there any saved entry for the name being looked up?
	// If so, return it
	isDir = resp.Attr.Mode.IsDir()
	if e := d.getEntry(req.Name); e != nil {
		switch e.(type) {
		case *Dir:
			return e.(*Dir), nil
		case *File:
			return e.(*File), nil
		}
	}

	// No saved entry found. Save it and return the appropriate
	// file or directory
	if isDir {
		dd := NewDir(d.path, req.Name, d.fs)
		d.saveEntry(req.Name, dd)
		return dd, nil
	}
	ff := NewFile(d.path, req.Name, d.fs)
	d.saveEntry(req.Name, ff)
	return ff, nil
}

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	if !d.isOpen() {
		return nil, fuse.ENOTSUP
	}
	defer trace(NewReadDirOp(d.path, d.ProcessInfo, d.handleID))
	names, err := d.file.Readdirnames(0)
	if err != nil {
		return nil, fuse.EIO
	}
	result := make([]fuse.Dirent, 0, len(names)+2)
	for _, n := range names {
		if skipDirEntry(n) {
			continue
		}
		entry := getFuseDirent(filepath.Join(d.path, n), n)
		result = append(result, entry)
	}

	// Add '.' and '..' to the result
	dots := make([]fuse.Dirent, 2)
	dots[0] = getFuseDirent(d.path, ".")
	if len(d.parent) > 0 {
		dots[1] = getFuseDirent(d.parent, "..")
	} else {
		dots[1] = dots[0]
		dots[1].Name = ".."
	}
	return append(result, dots...), nil
}

func (d *Dir) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fusefs.Node, error) {
	path := filepath.Join(d.path, req.Name)
	defer trace(NewMkdirOp(req, path, req.Mode))
	if err := os.Mkdir(path, req.Mode); err != nil {
		return nil, osErrorToFuseError(err)
	}
	newdir := NewDir(d.path, req.Name, d.fs)
	d.saveEntry(req.Name, newdir)
	return newdir, nil
}

func (d *Dir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	path := filepath.Join(d.path, req.Name)
	defer trace(NewRemoveOp(req, path))
	if err := os.Remove(path); err != nil {
		return osErrorToFuseError(err)
	}
	d.dropEntry(req.Name)
	return nil
}

func (d *Dir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fusefs.Node, fusefs.Handle, error) {
	path := filepath.Join(d.path, req.Name)
	op := NewCreateOp(req, path)
	defer trace(op)
	h := NewHandle()
	if err := h.doCreate(path, req.Flags, req.Mode); err != nil {
		return nil, nil, err
	}
	newfile := NewFileWithHandle(d.path, req.Name, d.fs, h)
	d.saveEntry(req.Name, newfile)
	op.OpenID = newfile.handleID
	return newfile, newfile, nil
}

func (d *Dir) Symlink(ctx context.Context, req *fuse.SymlinkRequest) (fusefs.Node, error) {
	absNewName := filepath.Join(d.path, req.NewName)
	targetIsDir := false
	defer trace(NewSymlinkOp(req, absNewName, req.Target, targetIsDir))

	linkTarget, absTarget := req.Target, req.Target
	if rewriteSymlinkTargets {
		// Make sure the target of the symbolic link which will be created
		// is jailed within the boundaries of the shadow file system. That is,
		// the link target path file name uses a path under the shadow file
		// system and not under this file system mount point.
		// The goal of this is for the symbolic link not to be broken when
		// ClueFS is unmounted.

		// In the link target path, replace this file system's mount directory by
		// the corresponding directory in the shadow file system. Do this
		// rewriting of the symbolic link target path only when the target
		// of the symbolic link an absolute path. When it is a relative path,
		// it will be considered to be relative to the directory when the
		// symbolic link is created.
		if !filepath.IsAbs(req.Target) {
			absTarget = filepath.Join(d.path, req.Target)
		}

		// If the link target path is under the mount directory, rewrite it
		// to be under the shadow directory
		if strings.HasPrefix(absTarget, d.fs.mountDir) {
			absTarget = strings.Replace(absTarget, d.fs.mountDir, d.fs.shadowDir, 1)
			linkTarget = absTarget
		}
	}

	// Does the link target actually exist?
	if info, err := os.Lstat(absTarget); err == nil {
		// The symbolic link target does exist
		targetIsDir = info.IsDir()
	}

	// Create the symbolic link: absNewName --> linkTarget
	if err := os.Symlink(linkTarget, absNewName); err != nil {
		return nil, osErrorToFuseError(err)
	}
	if targetIsDir {
		entry := NewDir(d.path, req.NewName, d.fs)
		d.saveEntry(req.NewName, entry)
		return entry, nil
	}
	entry := NewFile(d.path, req.NewName, d.fs)
	d.saveEntry(req.NewName, entry)
	return entry, nil
}

func (d *Dir) Rename(ctx context.Context, req *fuse.RenameRequest, newDir fusefs.Node) error {
	destDir, ok := newDir.(*Dir)
	if !ok {
		return fuse.EIO
	}
	oldpath := filepath.Join(d.path, req.OldName)
	newpath := filepath.Join(destDir.path, req.NewName)
	defer trace(NewRenameOp(req, oldpath, newpath))
	if err := os.Rename(oldpath, newpath); err != nil {
		return osErrorToFuseError(err)
	}

	// Delete the name of the node just renamed from the set of entries
	// of the source directory and add the new name to the entries of the
	// destination directory.
	// In addition, set the new parent and new name of the renamed node
	if e := d.dropEntry(req.OldName); e != nil {
		switch e.(type) {
		case *Dir:
			// The renamed node is a directory
			dd := e.(*Dir)
			dd.setParentAndName(destDir.path, req.NewName)
			destDir.saveEntry(req.NewName, dd)
		case *File:
			// The renamed node is a file
			ff := e.(*File)
			ff.setParentAndName(destDir.path, req.NewName)
			destDir.saveEntry(req.NewName, ff)
		default:
			// should not happen. Do nothing
		}
	}
	return nil
}

// osErrorToFuseError converts an os.PathError, os.LinkError or
// syscall.Errno into an error
func osErrorToFuseError(err error) error {
	if err == nil {
		return nil
	}
	errno := syscall.EIO
	if patherr, ok := err.(*os.PathError); ok {
		errno = patherr.Err.(syscall.Errno)
	} else if linkerr, ok := err.(*os.LinkError); ok {
		errno = linkerr.Err.(syscall.Errno)
	} else if _, ok := err.(*syscall.Errno); ok {
		errno = err.(syscall.Errno)
	}
	return fuse.Errno(errno)
}
