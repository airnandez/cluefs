package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"bazil.org/fuse"
)

// Information about the process which requested the file I/O operation
type ProcessInfo struct {
	Uid uint32
	Gid uint32
	Pid uint32
}

func (proc ProcessInfo) String() string {
	return fmt.Sprintf("%s(%d) %s(%d) '%s'(%d)", userName(proc.Uid), proc.Uid, groupName(proc.Gid), proc.Gid, processPath(proc.Pid), proc.Pid)
}

// Information shared by all traceable file system operations
type Header struct {
	ProcessInfo
	OperType FSOperType
	Start    time.Time
	End      time.Time
	Path     string
	IsDir    bool
}

func NewHeader(h fuse.Header, path string, isDir bool, op FSOperType) Header {
	return NewHeaderProcessInfo(ProcessInfo{h.Uid, h.Gid, h.Pid}, path, isDir, op)
}

func NewHeaderFile(h fuse.Header, path string, op FSOperType) Header {
	return NewHeaderProcessInfo(ProcessInfo{h.Uid, h.Gid, h.Pid}, path, false, op)
}

func NewHeaderDir(h fuse.Header, path string, op FSOperType) Header {
	return NewHeaderProcessInfo(ProcessInfo{h.Uid, h.Gid, h.Pid}, path, true, op)
}

func NewHeaderProcessInfo(proc ProcessInfo, path string, isDir bool, op FSOperType) Header {
	now := time.Now()
	return Header{
		ProcessInfo: proc,
		OperType:    op,
		Start:       now,
		End:         now,
		Path:        path,
		IsDir:       isDir,
	}
}

func (h *Header) String() string {
	return fmt.Sprintf("[%s %d] %s", h.ProcessInfo, h.Duration().Nanoseconds(), h.OperType)
}

func (h *Header) MarshalJSON() ([]byte, error) {
	var jhdr = map[string]interface{}{
		"uid":     h.Uid,
		"usr":     userName(h.Uid),
		"gid":     h.Gid,
		"grp":     groupName(h.Gid),
		"pid":     h.Pid,
		"proc":    processPath(h.Pid),
		"start":   h.Start.UTC().Format(time.RFC3339Nano),
		"end":     h.End.UTC().Format(time.RFC3339Nano),
		"nselaps": h.Duration().Nanoseconds(),
	}
	return json.Marshal(jhdr)
}

func (h *Header) MarshalCSV() []string {
	// Pre-allocate so that operation-specific marshallers don't need
	// to re-allocate to extend the serialized version of each operation
	res := make([]string, 0, 16)
	return append(
		res,
		h.Start.UTC().Format(time.RFC3339Nano),
		h.End.UTC().Format(time.RFC3339Nano),
		fmt.Sprintf("%d", h.Duration().Nanoseconds()),
		userName(h.Uid),
		fmt.Sprintf("%d", h.Uid),
		groupName(h.Gid),
		fmt.Sprintf("%d", h.Gid),
		processPath(h.Pid),
		fmt.Sprintf("%d", h.Pid),
		h.Path,
		isDirMap[h.IsDir],
		h.OperType.String(),
	)
}

func (h *Header) Duration() time.Duration {
	return h.End.Sub(h.Start)
}

func (h *Header) SetTimeStart() {
	h.Start = time.Now()
}

func (h *Header) SetTimeEnd() {
	h.End = time.Now()
}

func (h *Header) SetIsDir(isDir bool) {
	h.IsDir = isDir
}

type FSOperType uint32

const (
	FsOpen FSOperType = iota
	FsRead
	FsWrite
	FsFlush
	FsRelease
	FsMkdir
	FsRemove
	FsCreate
	FsSymlink
	FsStat
	FsReadDir
	FsStatfs
	FsRename
	FsReadLink
	FsAccess
	FsSetAttr
	FsListXattr
	FsGetXattr
	FsRemoveXattr
	FsSetXattr
)

var opNames = map[FSOperType]string{
	FsOpen:        "open",
	FsRead:        "read",
	FsWrite:       "write",
	FsFlush:       "flush",
	FsRelease:     "close",
	FsMkdir:       "mkdir",
	FsRemove:      "unlink",
	FsCreate:      "creat",
	FsSymlink:     "symlink",
	FsStat:        "stat",
	FsReadDir:     "readdir",
	FsStatfs:      "statfs",
	FsRename:      "rename",
	FsReadLink:    "readlink",
	FsAccess:      "access",
	FsSetAttr:     "setattr",
	FsListXattr:   "listxattr",
	FsGetXattr:    "getxattr",
	FsRemoveXattr: "removexattr",
	FsSetXattr:    "setxattr",
}

func (t FSOperType) String() string {
	if n, ok := opNames[t]; ok {
		return n
	}
	return "unknown"
}

func permString(mode os.FileMode) string {
	return fmt.Sprintf("%0#4o", mode&os.ModePerm)
}

var openModeMap = map[fuse.OpenFlags]string{
	fuse.OpenReadOnly:  "O_RDONLY",
	fuse.OpenWriteOnly: "O_WRONLY",
	fuse.OpenReadWrite: "O_RDWR",
}

func openModeString(flags fuse.OpenFlags) string {
	mode := flags & fuse.OpenAccessModeMask
	if s, ok := openModeMap[mode]; ok {
		return s
	}
	return fmt.Sprintf("unknown mode [%x]", mode)
}

// Adapted from bazil.org/fuse
type flagName struct {
	bit  uint32
	name string
}

var openFlagNames = []flagName{
	{uint32(fuse.OpenCreate), "O_CREAT"},
	{uint32(fuse.OpenExclusive), "O_EXCL"},
	{uint32(fuse.OpenTruncate), "O_TRUNC"},
	{uint32(fuse.OpenAppend), "O_APPEND"},
	{uint32(fuse.OpenSync), "O_SYNC"},
}

func openFlagsString(flags fuse.OpenFlags) []string {
	mask := uint32(flags &^ fuse.OpenAccessModeMask)
	if mask == 0 {
		return []string{}
	}
	res := make([]string, 0, len(openFlagNames))
	for _, n := range openFlagNames {
		if mask&n.bit != 0 {
			res = append(res, n.name)
			mask &^= n.bit
		}
	}
	return res
}

func flagsString(flags fuse.OpenFlags) string {
	res := make([]string, 0, 8)
	res = append(res, openModeString(flags))
	res = append(res, openFlagsString(flags)...)
	return strings.Join(res, "|")
}

// ------------------------------------------------------------------
// Open

type OpenOp struct {
	Header
	Flags fuse.OpenFlags
	Perm  os.FileMode
}

func NewOpenOp(req *fuse.OpenRequest, path string) *OpenOp {
	return &OpenOp{
		Header: NewHeader(req.Header, path, req.Dir, FsOpen),
		Flags:  req.Flags,
		Perm:   os.FileMode(req.Flags).Perm(),
	}
}

func (op *OpenOp) String() string {
	return fmt.Sprintf("%s '%s' %s %s %s", &op.Header, op.Path, isDirMap[op.IsDir], flagsString(op.Flags), permString(op.Perm))
}

var isDirMap = map[bool]string{
	true:  "dir",
	false: "file",
}

func (op *OpenOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":  op.OperType.String(),
			"path":  op.Path,
			"isdir": op.IsDir,
			"flags": flagsString(op.Flags),
			"perm":  permString(op.Perm),
		},
	})
}

func (op *OpenOp) MarshalCSV() []string {
	return append(
		op.Header.MarshalCSV(),
		flagsString(op.Flags),
		permString(op.Perm),
	)
}

// ------------------------------------------------------------------
// Read

type ReadOp struct {
	Header
	FileSize  uint64
	Offset    int64
	Size      int
	BytesRead int
}

func NewReadOp(req *fuse.ReadRequest, path string, fileSize uint64) *ReadOp {
	return &ReadOp{
		Header:    NewHeaderFile(req.Header, path, FsRead),
		FileSize:  fileSize,
		Offset:    req.Offset,
		Size:      req.Size,
		BytesRead: -1,
	}
}

func (op *ReadOp) String() string {
	return fmt.Sprintf("%s '%s' %s %d %d %d %d", &op.Header, op.FileSize, op.Path, isDirMap[op.IsDir], op.Offset, op.Size, op.BytesRead)
}

func (op *ReadOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":      op.OperType.String(),
			"path":      op.Path,
			"isdir":     op.IsDir,
			"filesize":  op.FileSize,
			"position":  op.Offset,
			"bytesreq":  op.Size,
			"bytesread": op.BytesRead,
		},
	})
}

func (op *ReadOp) MarshalCSV() []string {
	return append(
		op.Header.MarshalCSV(),
		fmt.Sprintf("%d", op.FileSize),
		fmt.Sprintf("%d", op.Offset),
		fmt.Sprintf("%d", op.Size),
		fmt.Sprintf("%d", op.BytesRead),
	)
}

// ------------------------------------------------------------------
// Write

type WriteOp struct {
	Header
	Offset       int64
	Size         int
	BytesWritten int
}

func NewWriteOp(req *fuse.WriteRequest, path string) *WriteOp {
	return &WriteOp{
		Header:       NewHeaderFile(req.Header, path, FsWrite),
		Offset:       req.Offset,
		Size:         len(req.Data),
		BytesWritten: -1,
	}
}

func (op *WriteOp) String() string {
	return fmt.Sprintf("%s '%s' %s %d %d %d", &op.Header, op.Path, isDirMap[op.IsDir], op.Offset, op.Size, op.BytesWritten)
}

func (op *WriteOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":         op.OperType.String(),
			"path":         op.Path,
			"isdir":        op.IsDir,
			"position":     op.Offset,
			"bytesreq":     op.Size,
			"byteswritten": op.BytesWritten,
		},
	})
}

func (op *WriteOp) MarshalCSV() []string {
	return append(
		op.Header.MarshalCSV(),
		fmt.Sprintf("%d", op.Offset),
		fmt.Sprintf("%d", op.Size),
		fmt.Sprintf("%d", op.BytesWritten),
	)
}

// ------------------------------------------------------------------
// Flush

type FlushOp struct {
	Header
}

func NewFlushOp(req *fuse.FlushRequest, path string) *FlushOp {
	return &FlushOp{
		Header: NewHeaderFile(req.Header, path, FsFlush),
	}
}

func (op *FlushOp) String() string {
	return fmt.Sprintf("%s '%s' %s", &op.Header, op.Path, isDirMap[op.IsDir])
}

func (op *FlushOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":  op.OperType.String(),
			"path":  op.Path,
			"isdir": op.IsDir,
		},
	})
}

func (op *FlushOp) MarshalCSV() []string {
	return op.Header.MarshalCSV()
}

// ------------------------------------------------------------------
// Release

type ReleaseOp struct {
	Header
}

func NewReleaseOp(req *fuse.ReleaseRequest, path string) *ReleaseOp {
	return &ReleaseOp{
		Header: NewHeader(req.Header, path, req.Dir, FsRelease),
	}
}

func (op *ReleaseOp) String() string {
	return fmt.Sprintf("%s '%s' %s", &op.Header, op.Path, isDirMap[op.IsDir])
}

func (op *ReleaseOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":  op.OperType.String(),
			"path":  op.Path,
			"isdir": op.IsDir,
		},
	})
}

func (op *ReleaseOp) MarshalCSV() []string {
	return op.Header.MarshalCSV()
}

// ------------------------------------------------------------------
// Mkdir

type MkdirOp struct {
	Header
	Mode os.FileMode
}

func NewMkdirOp(req *fuse.MkdirRequest, path string, mode os.FileMode) *MkdirOp {
	return &MkdirOp{
		Header: NewHeaderDir(req.Header, path, FsMkdir),
		Mode:   mode,
	}
}

func (op *MkdirOp) String() string {
	return fmt.Sprintf("%s '%s' %s %s", &op.Header, op.Path, isDirMap[op.IsDir], permString(op.Mode))
}

func (op *MkdirOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":  op.OperType.String(),
			"path":  op.Path,
			"mode":  permString(op.Mode),
			"isdir": op.IsDir,
		},
	})
}

func (op *MkdirOp) MarshalCSV() []string {
	return append(
		op.Header.MarshalCSV(),
		permString(op.Mode),
	)
}

// ------------------------------------------------------------------
// Remove

type RemoveOp struct {
	Header
}

func NewRemoveOp(req *fuse.RemoveRequest, path string) *RemoveOp {
	return &RemoveOp{
		Header: NewHeader(req.Header, path, req.Dir, FsRemove),
	}
}

func (op *RemoveOp) String() string {
	return fmt.Sprintf("%s '%s' %s", &op.Header, op.Path, isDirMap[op.IsDir])
}

func (op *RemoveOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":  op.OperType.String(),
			"path":  op.Path,
			"isdir": op.IsDir,
		},
	})
}

func (op *RemoveOp) MarshalCSV() []string {
	return op.Header.MarshalCSV()
}

// ------------------------------------------------------------------
// Create

type CreateOp struct {
	Header
	Flags fuse.OpenFlags
	Mode  os.FileMode
}

func NewCreateOp(req *fuse.CreateRequest, path string) *CreateOp {
	return &CreateOp{
		Header: NewHeaderFile(req.Header, path, FsCreate),
		Flags:  req.Flags,
		Mode:   req.Mode,
	}
}

func (op *CreateOp) String() string {
	return fmt.Sprintf("%s '%s' %s %s %s", &op.Header, op.Path, isDirMap[op.IsDir], flagsString(op.Flags), permString(op.Mode))
}

func (op *CreateOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":  op.OperType.String(),
			"path":  op.Path,
			"isdir": op.IsDir,
			"flags": flagsString(op.Flags),
			"perm":  permString(op.Mode),
		},
	})
}

func (op *CreateOp) MarshalCSV() []string {
	return append(
		op.Header.MarshalCSV(),
		flagsString(op.Flags),
		permString(op.Mode),
	)
}

// ------------------------------------------------------------------
// Symlink

type SymlinkOp struct {
	Header
	Target string
}

func NewSymlinkOp(req *fuse.SymlinkRequest, path, target string, isDir bool) *SymlinkOp {
	return &SymlinkOp{
		Header: NewHeader(req.Header, path, isDir, FsSymlink),
		Target: target,
	}
}

func (op *SymlinkOp) String() string {
	return fmt.Sprintf("%s '%s' %s %s", &op.Header, op.Path, isDirMap[op.IsDir], op.Target)
}

func (op *SymlinkOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":   op.OperType.String(),
			"path":   op.Path,
			"isdir":  op.IsDir,
			"target": op.Target,
		},
	})
}

func (op *SymlinkOp) MarshalCSV() []string {
	return append(
		op.Header.MarshalCSV(),
		op.Target,
	)
}

// ------------------------------------------------------------------
// Lookup

type LookupOp struct {
	Header
}

func NewLookupOp(req *fuse.LookupRequest, path string, isDir bool) *LookupOp {
	return &LookupOp{
		Header: NewHeader(req.Header, path, isDir, FsStat),
	}
}

func (op *LookupOp) String() string {
	return fmt.Sprintf("%s '%s' %s", &op.Header, op.Path, isDirMap[op.IsDir])
}

func (op *LookupOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":  op.OperType.String(),
			"path":  op.Path,
			"isdir": op.IsDir,
		},
	})
}

func (op *LookupOp) MarshalCSV() []string {
	return op.Header.MarshalCSV()
}

// ------------------------------------------------------------------
// ReadDir

type ReadDirOp struct {
	Header
}

func NewReadDirOp(path string, id ProcessInfo) *ReadDirOp {
	return &ReadDirOp{
		Header: NewHeaderProcessInfo(id, path, true, FsReadDir),
	}
}

func (op *ReadDirOp) String() string {
	return fmt.Sprintf("%s '%s' %s", &op.Header, op.Path, isDirMap[op.IsDir])
}

func (op *ReadDirOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":  op.OperType.String(),
			"path":  op.Path,
			"isdir": op.IsDir,
		},
	})
}

func (op *ReadDirOp) MarshalCSV() []string {
	return op.Header.MarshalCSV()
}

// ------------------------------------------------------------------
// Statfs

type StatFsOp struct {
	Header
}

func NewStatFsOp(req *fuse.StatfsRequest, path string) *StatFsOp {
	return &StatFsOp{
		Header: NewHeaderDir(req.Header, path, FsStatfs),
	}
}

func (op *StatFsOp) String() string {
	return fmt.Sprintf("%s '%s' %s", &op.Header, op.Path, isDirMap[op.IsDir])
}

func (op *StatFsOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":  op.OperType.String(),
			"path":  op.Path,
			"isdir": op.IsDir,
		},
	})
}

func (op *StatFsOp) MarshalCSV() []string {
	return op.Header.MarshalCSV()
}

// ------------------------------------------------------------------
// Rename

type RenameOp struct {
	Header
	NewPath string
}

func NewRenameOp(req *fuse.RenameRequest, oldpath, newpath string) *RenameOp {
	return &RenameOp{
		Header:  NewHeaderFile(req.Header, oldpath, FsRename),
		NewPath: newpath,
	}
}

func (op *RenameOp) String() string {
	return fmt.Sprintf("%s '%s' %s '%s'", &op.Header, op.Path, op.NewPath, isDirMap[op.IsDir])
}

func (op *RenameOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":  op.OperType.String(),
			"isdir": op.IsDir,
			"old":   op.Path,
			"new":   op.NewPath,
		},
	})
}

func (op *RenameOp) MarshalCSV() []string {
	return append(
		op.Header.MarshalCSV(),
		op.NewPath,
	)
}

// ------------------------------------------------------------------
// Readlink

type ReadlinkOp struct {
	Header
}

func NewReadlinkOp(req *fuse.ReadlinkRequest, path string) *ReadlinkOp {
	return &ReadlinkOp{
		Header: NewHeaderFile(req.Header, path, FsReadLink),
	}
}

func (op *ReadlinkOp) String() string {
	return fmt.Sprintf("%s '%s' %s", &op.Header, op.Path, isDirMap[op.IsDir])
}

func (op *ReadlinkOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":  op.OperType.String(),
			"path":  op.Path,
			"isdir": op.IsDir,
		},
	})
}

func (op *ReadlinkOp) MarshalCSV() []string {
	return op.Header.MarshalCSV()
}

// ------------------------------------------------------------------
// Access

type AccessOp struct {
	Header
	Mask uint32
}

func NewAccessOp(req *fuse.AccessRequest, path string, isDir bool) *AccessOp {
	return &AccessOp{
		Header: NewHeader(req.Header, path, isDir, FsAccess),
		Mask:   req.Mask,
	}
}

func (op *AccessOp) String() string {
	return fmt.Sprintf("%s '%s' %s %x", &op.Header, op.Path, isDirMap[op.IsDir], op.Mask)
}

func (op *AccessOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":  op.OperType.String(),
			"path":  op.Path,
			"isdir": op.IsDir,
			"mode":  accessModeString(op.Mask),
		},
	})
}

func (op *AccessOp) MarshalCSV() []string {
	return append(
		op.Header.MarshalCSV(),
		accessModeString(op.Mask),
	)
}

// ------------------------------------------------------------------
// Setattr

type SetattrOp struct {
	Header
	AttrValid fuse.SetattrValid
}

func NewSetattrOp(req *fuse.SetattrRequest, path string) *SetattrOp {
	return &SetattrOp{
		Header:    NewHeaderFile(req.Header, path, FsSetAttr),
		AttrValid: req.Valid,
	}
}

func (op *SetattrOp) String() string {
	// TODO: improve printing of attribute to decouple from fuse
	return fmt.Sprintf("%s '%s' %s %s", &op.Header, op.Path, isDirMap[op.IsDir], op.AttrValid)
}

func (op *SetattrOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":  op.OperType.String(),
			"path":  op.Path,
			"isdir": op.IsDir,
			// TODO: marshal attribute
		},
	})
}

func (op *SetattrOp) MarshalCSV() []string {
	return append(
		op.Header.MarshalCSV(),
		// TODO: marshal attribute
		// op.AttrValid,
	)
}

// ------------------------------------------------------------------
// Getxattr

type GetxattrOp struct {
	Header
	AttrName string
}

func NewGetxattrOp(req *fuse.GetxattrRequest, path string) *GetxattrOp {
	return &GetxattrOp{
		Header:   NewHeaderFile(req.Header, path, FsGetXattr),
		AttrName: req.Name,
	}
}

func (op *GetxattrOp) String() string {
	return fmt.Sprintf("%s '%s' %s %s", &op.Header, op.Path, isDirMap[op.IsDir], op.AttrName)
}

func (op *GetxattrOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":  op.OperType.String(),
			"path":  op.Path,
			"isdir": op.IsDir,
			"name":  op.AttrName,
		},
	})
}

func (op *GetxattrOp) MarshalCSV() []string {
	return append(
		op.Header.MarshalCSV(),
		op.AttrName,
	)
}

// ------------------------------------------------------------------
// Listxattr

type ListxattrOp struct {
	Header
	Size uint32
}

func NewListxattrOp(req *fuse.ListxattrRequest, path string) *ListxattrOp {
	return &ListxattrOp{
		Header: NewHeaderFile(req.Header, path, FsListXattr),
		Size:   req.Size,
	}
}

func (op *ListxattrOp) String() string {
	return fmt.Sprintf("%s '%s' %s %d", &op.Header, op.Path, isDirMap[op.IsDir], op.Size)
}

func (op *ListxattrOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":  op.OperType.String(),
			"path":  op.Path,
			"isdir": op.IsDir,
			"size":  op.Size,
		},
	})
}

func (op *ListxattrOp) MarshalCSV() []string {
	return append(
		op.Header.MarshalCSV(),
		fmt.Sprintf("%d", op.Size),
	)
}

// ------------------------------------------------------------------
// Setxattr

type SetxattrOp struct {
	Header
	AttrName string
}

func NewSetxattrOp(req *fuse.SetxattrRequest, path string) *SetxattrOp {
	return &SetxattrOp{
		Header:   NewHeaderFile(req.Header, path, FsSetXattr),
		AttrName: req.Name,
	}
}

func (op *SetxattrOp) String() string {
	// TODO: improve printing of attribute to decouple from fuse
	return fmt.Sprintf("%s '%s' %s '%s'", &op.Header, op.Path, isDirMap[op.IsDir], op.AttrName)
}

func (op *SetxattrOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":  op.OperType.String(),
			"path":  op.Path,
			"isdir": op.IsDir,
			"name":  op.AttrName,
		},
	})
}

func (op *SetxattrOp) MarshalCSV() []string {
	return append(
		op.Header.MarshalCSV(),
		op.AttrName,
	)
}

// ------------------------------------------------------------------
// Removexattr

type RemovexattrOp struct {
	Header
	AttrName string
}

func NewRemovexattrOp(req *fuse.RemovexattrRequest, path string) *RemovexattrOp {
	return &RemovexattrOp{
		Header:   NewHeaderFile(req.Header, path, FsRemoveXattr),
		AttrName: req.Name,
	}
}

func (op *RemovexattrOp) String() string {
	return fmt.Sprintf("%s '%s' %s '%s'", &op.Header, op.Path, isDirMap[op.IsDir], op.AttrName)
}

func (op *RemovexattrOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hdr": &op.Header,
		"op": map[string]interface{}{
			"type":  op.OperType.String(),
			"path":  op.Path,
			"isdir": op.IsDir,
			"name":  op.AttrName,
		},
	})
}

func (op *RemovexattrOp) MarshalCSV() []string {
	return append(
		op.Header.MarshalCSV(),
		op.AttrName,
	)
}
