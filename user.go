package main

import (
	"fmt"
	"os/user"
	"sync"
)

// userStore is a map from user ids to user name
var (
	userMutex  sync.RWMutex
	userStore  map[uint32]string
	groupMutex sync.RWMutex
	groupStore map[uint32]string
)

func init() {
	userStore = make(map[uint32]string, 512)
	groupStore = make(map[uint32]string, 256)
}

func userName(uid uint32) string {
	userMutex.RLock()
	name, ok := userStore[uid]
	userMutex.RUnlock()
	if !ok {
		name = uidToUserName(uid)
		userMutex.Lock()
		userStore[uid] = name
		userMutex.Unlock()
	}
	return name
}

func uidToUserName(uid uint32) string {
	u, err := user.LookupId(fmt.Sprintf("%d", uid))
	if err != nil {
		return ""
	}
	return u.Username
}

func groupName(uid uint32) string {
	groupMutex.RLock()
	name, ok := groupStore[uid]
	groupMutex.RUnlock()
	if !ok {
		name = gidToGroupName(uid)
		groupMutex.Lock()
		groupStore[uid] = name
		groupMutex.Unlock()
	}
	return name
}
