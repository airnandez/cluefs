package main

// #cgo LDFLAGS: -lproc

/*
#include <stdlib.h>
#include <libproc.h>

// getProcessPath returns a buffer which contains the full path of the
// process id provided in the argument. The returned buffer is allocated
// in the heap and it is the responsibility of the caller to free the
// returned pointer. getProcessPath may return NULL or an empty string.
char* getProcessPath(int pid)
{
	const int pathLength = PROC_PIDPATHINFO_SIZE + 1; // 1024 + 1
	char* path = malloc(pathLength);
	if (path != NULL) {
		int len = proc_pidpath(pid, (void*)path, pathLength-1);
		path[len] = '\0';
	}
	return path;
}
*/
import "C"
import "unsafe"

// osProcessPath returns the full path of the executable program of a
// process given its id. If the path cannot be retrieved, it returns the
// empty string.
func osProcessPath(pid uint32) string {
	path := C.getProcessPath(C.int(pid))
	defer C.free(unsafe.Pointer(path))
	return C.GoString(path)
}
