package cluefs

import (
	"fmt"
	"os"
	"path/filepath"

	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

type ClueFS struct {
	shadowDir string
	mountDir  string
	root      *Dir
}

var trace func(op FsOperTracer)

func FuseDebug(msg interface{}) {
	Debug(4, "*** FUSE: %s", msg)
}

func NewClueFS(shadowDir string, tracer Tracer) (*ClueFS, error) {
	if !filepath.IsAbs(shadowDir) {
		return nil, fmt.Errorf("'%s' is not an absolute path", shadowDir)
	}
	dir, err := os.Open(shadowDir)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	// Initialize the trace function this file system will use to
	// emit file I/O events
	trace = func(op FsOperTracer) {
		op.SetTimeEnd()
		tracer.Trace(op)
	}
	return &ClueFS{shadowDir: shadowDir}, nil
}

func (fs *ClueFS) MountAndServe(mountpoint string, readonly bool) error {
	// Mount the file system
	fs.mountDir = mountpoint
	if IsDebugActive() {
		fuse.Debug = FuseDebug
	}
	mountOpts := []fuse.MountOption{
		fuse.FSName(programName),
		fuse.Subtype(programName),
		fuse.VolumeName(programName),
		fuse.LocalVolume(),
	}
	if readonly {
		mountOpts = append(mountOpts, fuse.ReadOnly())
	}
	conn, err := fuse.Mount(mountpoint, mountOpts...)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Start serving requests
	if err = fusefs.Serve(conn, fs); err != nil {
		return err
	}

	// Check for errors when mounting the file system
	<-conn.Ready
	if err = conn.MountError; err != nil {
		return err
	}

	return nil
}

func (fs *ClueFS) Root() (fusefs.Node, error) {
	if fs.root == nil {
		fs.root = NewDir("", fs.shadowDir, fs)
	}
	return fs.root, nil
}

func (fs *ClueFS) Statfs(ctx context.Context, req *fuse.StatfsRequest, resp *fuse.StatfsResponse) error {
	defer trace(NewStatFsOp(req, fs.mountDir))
	return statfsToFuse(fs.shadowDir, resp)
}

func (fs *ClueFS) Destroy() {
	if fs.root != nil {
		fs.root.doClose()
		fs.root = nil
	}
}
