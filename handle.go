package main

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
	size     uint64
	blksize  uint32
}

func (h Handle) String() string {
	return fmt.Sprintf("%d", h.handleID)
}

func (h *Handle) isOpen() bool {
	return h.file != nil
}

func (h *Handle) doOpen(path string, flags int, perm os.FileMode) error {
	if h.isOpen() {
		return nil
	}
	file, err := os.OpenFile(path, flags, perm)
	if err != nil {
		return osErrorToFuseError(err)
	}
	var stat syscall.Stat_t
	if err := syscall.Fstat(int(file.Fd()), &stat); err != nil {
		return osErrorToFuseError(err)
	}
	h.size = uint64(stat.Size)
	h.blksize = uint32(stat.Blksize)
	h.file = file
	h.handleID = newHandleID()
	return nil
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

func (h *Handle) doSync() error {
	if !h.isOpen() {
		return nil
	}
	if err := h.file.Sync(); err != nil {
		return osErrorToFuseError(err)
	}
	var stat syscall.Stat_t
	if err := syscall.Fstat(int(h.file.Fd()), &stat); err != nil {
		return osErrorToFuseError(err)
	}
	h.size = uint64(stat.Size)
	return nil
}
