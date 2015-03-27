// +build linux darwin

package main

/*
#include <unistd.h>
#include <stdlib.h>
#include <string.h>
#include <grp.h>

// getGroupName retrieves the group name given a group id or NULL if the
// specified group id does not exist or there was an error retrieving it.
// It is the responsibility of the caller to free the returned memory.
char* getGroupName(int gid)
{
	long bufsize = sysconf(_SC_GETGR_R_SIZE_MAX);
	if (bufsize <= 0) {
		bufsize = 1024;
	}

	char* buffer = malloc((int)bufsize * sizeof(char));
	if (buffer == NULL) {
		return NULL;
	}

	struct group grp;
	struct group* result;
	if (getgrgid_r(gid, &grp, buffer, (size_t)bufsize, &result) != 0) {
		free(buffer);
		return NULL;
	}

	char* groupName = NULL;
	if (result != NULL) {
		groupName = strdup(result->gr_name);
	}
	free(buffer);
	return groupName;
}
*/
import "C"
import "unsafe"

func gidToGroupName(gid uint32) string {
	gname := C.getGroupName(C.int(gid))
	defer C.free(unsafe.Pointer(gname))
	return C.GoString(gname)
}
