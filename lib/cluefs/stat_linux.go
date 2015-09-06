package cluefs

import (
	"os"
	"syscall"
	"time"

	"bazil.org/fuse"
)

func statToFuseAttr(st syscall.Stat_t) fuse.Attr {
	var mode os.FileMode
	switch st.Mode & syscall.S_IFMT {
	case syscall.S_IFBLK:
		mode = os.ModeDevice
	case syscall.S_IFCHR:
		mode = os.ModeDevice | os.ModeCharDevice
	case syscall.S_IFDIR:
		mode = os.ModeDir
	case syscall.S_IFIFO:
		mode = os.ModeNamedPipe
	case syscall.S_IFLNK:
		mode = os.ModeSymlink
	case syscall.S_IFREG:
		// nothing to do
	case syscall.S_IFSOCK:
		mode = os.ModeSocket
	}
	if st.Mode&syscall.S_ISGID != 0 {
		mode |= os.ModeSetgid
	}
	if st.Mode&syscall.S_ISUID != 0 {
		mode |= os.ModeSetuid
	}
	if st.Mode&syscall.S_ISVTX != 0 {
		mode |= os.ModeSticky
	}
	perm := os.FileMode(st.Mode) & os.ModePerm
	return fuse.Attr{
		Inode:     uint64(st.Ino),
		Size:      uint64(st.Size),
		Blocks:    uint64(st.Blocks),
		Atime:     timespecToTime(st.Atim),
		Mtime:     timespecToTime(st.Mtim),
		Ctime:     timespecToTime(st.Ctim),
		Crtime:    timespecToTime(st.Ctim),
		Mode:      perm | mode,
		Nlink:     uint32(st.Nlink),
		Uid:       st.Uid,
		Gid:       st.Gid,
		Rdev:      uint32(st.Rdev), //TODO: how to correctly convert from Stat_t (64bits) to RDev (32bits)
		BlockSize: uint32(st.Blksize),
	}
}

// Source: os.Syscall
func timespecToTime(ts syscall.Timespec) time.Time {
	return time.Unix(int64(ts.Sec), int64(ts.Nsec))
}

func fuseTypeFromStatMode(mode uint32) fuse.DirentType {
	t := fuse.DT_Unknown
	switch mode & syscall.S_IFMT {
	case syscall.S_IFBLK:
		t = fuse.DT_Block
	case syscall.S_IFCHR:
		t = fuse.DT_Char
	case syscall.S_IFDIR:
		t = fuse.DT_Dir
	case syscall.S_IFIFO:
		t = fuse.DT_FIFO
	case syscall.S_IFLNK:
		t = fuse.DT_Link
	case syscall.S_IFREG:
		t = fuse.DT_File
	case syscall.S_IFSOCK:
		t = fuse.DT_Socket
	}
	return t
}

func statAtimeMtime(fullpath string) (time.Time, time.Time, error) {
	var st syscall.Stat_t
	if err := syscall.Lstat(fullpath, &st); err != nil {
		var nulltime time.Time
		return nulltime, nulltime, err
	}
	return timespecToTime(st.Atim), timespecToTime(st.Mtim), nil
}

func statfsToFuse(path string, resp *fuse.StatfsResponse) error {
	var buf syscall.Statfs_t
	if err := syscall.Statfs(path, &buf); err != nil {
		return fuse.ENOTSUP
	}
	resp.Blocks = uint64(buf.Blocks) // Total data blocks in file system.
	resp.Bfree = uint64(buf.Bfree)   // Free blocks in file system.
	resp.Bavail = uint64(buf.Bavail) // Free blocks in file system if you're not root.
	resp.Files = uint64(buf.Files)   // Total files in file system.
	resp.Ffree = uint64(buf.Ffree)   // Free files in file system.
	resp.Bsize = uint32(buf.Bsize)   // Block size
	resp.Namelen = uint32(buf.Namelen)
	resp.Frsize = uint32(buf.Frsize)
	return nil
}
