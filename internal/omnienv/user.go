package omnienv

import "os"

type UserInfo struct {
	UID int
	GID int
}

func CurrentUserInfo() UserInfo {
	return UserInfo{
		UID: os.Getuid(),
		GID: os.Getgid(),
	}
}
