package cluefs

import (
	"fmt"
	"os"
	"syscall"

	"bazil.org/fuse"
)

var (
	handleIDGeneratorChan <-chan fuse.HandleID
)

func handleIDGenerator() <-chan fuse.HandleID {
	outChan := make(chan fuse.HandleID)
	go func() {
		for nextId := fuse.HandleID(1); ; nextId++ {
			outChan <- nextId
		}
	}()
	return outChan
}

func newHandleID() fuse.HandleID {
	return <-handleIDGeneratorChan
}

func init() {
	handleIDGeneratorChan = handleIDGenerator()
}

type Handle struct {
	file     *os.File
	handleID fuse.HandleID
	flags    fuse.OpenFlags
	blksize  uint32
}

func NewHandle() *Handle {
	return &Handle{}
}

func (h Handle) String() string {
	return fmt.Sprintf("%d", h.handleID)
}

func (h *Handle) isOpen() bool {
	return h.file != nil
}

func (h *Handle) doOpen(path string, flags fuse.OpenFlags) (uint64, error) {
	if h.isOpen() {
		return 0, nil
	}
	mode := int(flags & fuse.OpenAccessModeMask)
	perm := os.FileMode(flags).Perm()
	file, err := os.OpenFile(path, mode, perm)
	if err != nil {
		return 0, osErrorToFuseError(err)
	}
	blksize, err := getBlkSize(file)
	if err != nil {
		return 0, err
	}
	h.file, h.flags, h.handleID, h.blksize = file, flags, newHandleID(), blksize
	return h.getFileSize()
}

func (h *Handle) doCreate(path string, flags fuse.OpenFlags, mode os.FileMode) error {
	if h.isOpen() {
		return nil
	}
	file, err := os.OpenFile(path, int(flags), mode)
	if err != nil {
		return osErrorToFuseError(err)
	}
	blksize, err := getBlkSize(file)
	if err != nil {
		return err
	}
	h.file, h.flags, h.handleID, h.blksize = file, flags, newHandleID(), blksize
	return nil
}

func (h *Handle) getFileSize() (uint64, error) {
	var stat syscall.Stat_t
	if err := syscall.Fstat(int(h.file.Fd()), &stat); err != nil {
		return 0, osErrorToFuseError(err)
	}
	return uint64(stat.Size), nil
}

func (h *Handle) doClose() error {
	if h.isOpen() {
		file := h.file
		h.file = nil
		h.handleID = 0
		return osErrorToFuseError(file.Close())
	}
	return nil
}

func (h *Handle) doSync() (uint64, error) {
	if !h.isOpen() {
		return 0, nil
	}
	if err := h.file.Sync(); err != nil {
		return 0, osErrorToFuseError(err)
	}
	return h.getFileSize()
}

func getBlkSize(f *os.File) (uint32, error) {
	var stat syscall.Stat_t
	if err := syscall.Fstat(int(f.Fd()), &stat); err != nil {
		return 0, osErrorToFuseError(err)
	}
	return uint32(stat.Blksize), nil
}
