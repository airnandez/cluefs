package main

import (
	"io"
	"os"

	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

type File struct {
	*Node
	*Handle
}

func NewFile(parent string, name string, fs *ClueFS) *File {
	return &File{
		Node:   NewNode(parent, name, fs),
		Handle: &Handle{},
	}
}

func NewOpenFile(parent string, name string, fs *ClueFS, file *os.File) *File {
	return &File{
		Node:   NewNode(parent, name, fs),
		Handle: &Handle{file: file, handleID: newHandleID()},
	}
}

func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fusefs.Handle, error) {
	defer trace(NewOpenOp(req, f.path))
	newfile := NewFile(f.parent, f.name, f.fs)
	perm := os.FileMode(req.Flags).Perm()
	flags := int(req.Flags & fuse.OpenAccessModeMask)
	if err := newfile.doOpen(f.path, flags, perm); err != nil {
		return nil, err
	}
	resp.Handle = newfile.handleID
	return newfile, nil
}

func (f *File) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	if !f.isOpen() {
		return fuse.ENOTSUP
	}
	defer trace(NewReleaseOp(req, f.path))
	if req.ReleaseFlags&fuse.ReleaseFlush != 0 {
		f.doSync()
	}
	return f.doClose()
}

func (f *File) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	if !f.isOpen() {
		return fuse.ENOTSUP
	}
	defer trace(NewFlushOp(req, f.path))
	return f.doSync()
}

func (f *File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	if !f.isOpen() {
		return fuse.ENOTSUP
	}
	op := NewReadOp(req, f.path, f.size)
	defer trace(op)
	n, err := f.file.ReadAt(resp.Data[0:req.Size], req.Offset)
	resp.Data = resp.Data[0:n]
	op.BytesRead = n
	if err == nil || err == io.EOF {
		return nil
	}
	return osErrorToFuseError(err)
}

func (f *File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	if !f.isOpen() {
		return fuse.ENOTSUP
	}
	op := NewWriteOp(req, f.path)
	defer trace(op)
	var err error
	resp.Size, err = f.file.WriteAt(req.Data, req.Offset)
	op.BytesWritten = resp.Size
	return osErrorToFuseError(err)
}
