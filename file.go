package main

import (
	"io"

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

func NewFileWithHandle(parent string, name string, fs *ClueFS, handle *Handle) *File {
	return &File{
		Node:   NewNode(parent, name, fs),
		Handle: handle,
	}
}

func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fusefs.Handle, error) {
	op := NewOpenOp(req, f.path)
	defer trace(op)
	newfile := NewFile(f.parent, f.name, f.fs)
	size, err := newfile.doOpen(f.path, req.Flags)
	if err != nil {
		return nil, err
	}
	resp.Handle = fuse.HandleID(newfile.handleID)
	op.FileSize = size
	op.BlockSize = newfile.blksize
	op.OpenID = newfile.handleID
	return newfile, nil
}

func (f *File) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	if !f.isOpen() {
		return fuse.ENOTSUP
	}
	defer trace(NewReleaseOp(req, f.path, f.handleID))
	if req.ReleaseFlags&fuse.ReleaseFlush != 0 {
		f.doSync()
	}
	return f.doClose()
}

func (f *File) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	if !f.isOpen() {
		return fuse.ENOTSUP
	}
	op := NewFlushOp(req, f.path, f.handleID)
	defer trace(op)
	size, err := f.doSync()
	if err != nil {
		return err
	}
	op.FileSize = size
	op.Flags = fuse.OpenFlags(f.flags)
	return nil
}

func (f *File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	if !f.isOpen() {
		return fuse.ENOTSUP
	}
	op := NewReadOp(req, f.path, f.handleID)
	defer trace(op)
	size, err := f.getFileSize()
	if err != nil {
		return err
	}
	op.FileSize = size
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
	op := NewWriteOp(req, f.path, f.handleID)
	defer trace(op)
	var err error
	resp.Size, err = f.file.WriteAt(req.Data, req.Offset)
	op.BytesWritten = resp.Size
	return osErrorToFuseError(err)
}
