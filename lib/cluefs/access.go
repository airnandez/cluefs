// +build linux darwin
package cluefs

/*
#include <unistd.h>
*/
import "C"

func access(path string, mode uint32) bool {
	return C.access(C.CString(path), C.int(mode)) == 0
}

var accessModeMap = map[uint32]string{
	// See: <unistd.h> for these values
	0:        "F_OK",
	(1 << 0): "X_OK",
	(1 << 1): "W_OK",
	(1 << 2): "R_OK",
}

func accessModeString(mode uint32) string {
	if s, ok := accessModeMap[mode&0x7]; ok {
		return s
	}
	return "unknown"
}
