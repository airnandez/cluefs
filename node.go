package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
	"bazil.org/fuse/syscallx"
	"golang.org/x/net/context"
)

type Node struct {
	parent string
	name   string
	path   string
	fs     *ClueFS
}

func NewNode(parent string, name string, fs *ClueFS) *Node {
	path := filepath.Join(parent, name)
	if len(parent) == 0 {
		path = name
	}
	return &Node{parent: parent, name: name, path: path, fs: fs}
}

func (n Node) String() string {
	return fmt.Sprintf("%s %s", n.parent, n.name)
}

func (n *Node) Attr(ctx context.Context, attr *fuse.Attr) error {
	var st syscall.Stat_t
	syscall.Lstat(n.path, &st)
	*attr = statToFuseAttr(st)
	return nil
}

func (n *Node) Access(ctx context.Context, req *fuse.AccessRequest) error {
	isDir, err := isDir(n.path)
	defer trace(NewAccessOp(req, n.path, isDir))
	if err != nil {
		return err
	}
	if access(n.path, req.Mask) {
		return nil
	}
	return fuse.Errno(syscall.EACCES)
}

func (n *Node) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	defer trace(NewSetattrOp(req, n.path))
	var err error
	if req.Valid.Atime() {
		var mtime time.Time
		_, mtime, err = statAtimeMtime(n.path)
		if err == nil {
			err = os.Chtimes(n.path, req.Atime, mtime)
		}
	} else if req.Valid.Mtime() {
		var atime time.Time
		atime, _, err = statAtimeMtime(n.path)
		if err == nil {
			err = os.Chtimes(n.path, atime, req.Mtime)
		}
	} else if req.Valid.Bkuptime() {
		// TODO: set backup time
	} else if req.Valid.Chgtime() {
		// TODO: set change time
	} else if req.Valid.Crtime() {
		// TODO: set creation time
	} else if req.Valid.Flags() {
		// TODO: set flags
	} else if req.Valid.Uid() {
		var gid uint32
		_, gid, err = getUidGid(n.path)
		if err == nil {
			err = os.Chown(n.path, int(req.Uid), int(gid))
		}
	} else if req.Valid.Gid() {
		var uid uint32
		uid, _, err = getUidGid(n.path)
		if err == nil {
			err = os.Chown(n.path, int(uid), int(req.Gid))
		}
	} else if req.Valid.Size() {
		err = os.Truncate(n.path, int64(req.Size))
	} else if req.Valid.Mode() {
		err = os.Chmod(n.path, req.Mode.Perm())
	}
	if err != nil {
		return osErrorToFuseError(err)
	}
	return n.Attr(ctx, &resp.Attr)
}

func (n *Node) Rename(ctx context.Context, req *fuse.RenameRequest, newDir fusefs.Node) error {
	destDir, ok := newDir.(*Dir)
	if !ok {
		return fuse.EIO
	}
	oldpath := filepath.Join(n.path, req.OldName)
	newpath := filepath.Join(destDir.path, req.NewName)
	defer trace(NewRenameOp(req, oldpath, newpath))
	return osErrorToFuseError(os.Rename(oldpath, newpath))
}

func (n *Node) Readlink(ctx context.Context, req *fuse.ReadlinkRequest) (string, error) {
	defer trace(NewReadlinkOp(req, n.path))
	dest, err := os.Readlink(n.path)
	if err != nil {
		return "", osErrorToFuseError(err)
	}
	return dest, nil
}

func (n *Node) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	defer trace(NewGetxattrOp(req, n.path))
	size, err := syscallx.Getxattr(n.path, req.Name, []byte{})
	if err != nil || size <= 0 {
		return fuse.ErrNoXattr
	}
	buffer := make([]byte, size)
	size, err = syscallx.Getxattr(n.path, req.Name, buffer)
	if err != nil {
		return osErrorToFuseError(err)
	}
	resp.Xattr = buffer
	return nil
}

func (n *Node) Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	defer trace(NewListxattrOp(req, n.path))
	size, err := syscallx.Listxattr(n.path, []byte{})
	if err != nil || size <= 0 {
		return nil
	}
	buffer := make([]byte, size)
	size, err = syscallx.Listxattr(n.path, buffer)
	if err != nil {
		return osErrorToFuseError(err)
	}
	resp.Xattr = buffer
	return nil
}

func (n *Node) Setxattr(ctx context.Context, req *fuse.SetxattrRequest) error {
	defer trace(NewSetxattrOp(req, n.path))
	err := syscallx.Setxattr(n.path, req.Name, req.Xattr, int(req.Flags))
	return osErrorToFuseError(err)
}

func (n *Node) Removexattr(ctx context.Context, req *fuse.RemovexattrRequest) error {
	defer trace(NewRemovexattrOp(req, n.path))
	// TODO: this needs to be improved, since the behavior of Removexattr depends
	// on the previous existance of the attribute. The return code of the operation
	// is governed by the flags. See bazil.org/fuse/syscallx.Removexattr comments.
	_, err := syscallx.Getxattr(n.path, req.Name, []byte{})
	if err == nil {
		// TODO: There is already an attribute with that name. Should return
		// the expected error code according to the request's flags
		err = syscallx.Removexattr(n.path, req.Name)
		return osErrorToFuseError(err)
	}
	return nil
}

func isDir(fullpath string) (bool, error) {
	var st syscall.Stat_t
	if err := syscall.Stat(fullpath, &st); err != nil {
		return false, osErrorToFuseError(err)
	}
	return st.Mode&syscall.S_IFMT == syscall.S_IFDIR, nil
}

func getFuseDirent(fullpath string, name string) fuse.Dirent {
	var st syscall.Stat_t
	syscall.Lstat(fullpath, &st)
	return fuse.Dirent{
		Inode: st.Ino,
		Type:  fuseTypeFromStatMode(st.Mode),
		Name:  name,
	}
}

func getUidGid(path string) (uint32, uint32, error) {
	var st syscall.Stat_t
	if err := syscall.Lstat(path, &st); err != nil {
		return 0, 0, err
	}
	return st.Uid, st.Gid, nil
}
