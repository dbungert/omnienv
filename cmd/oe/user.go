package main

import "os"

type UserInfo struct {
	uid int
	gid int
}

func CurrentUserInfo() UserInfo {
	return UserInfo{
		uid: os.Getuid(),
		gid: os.Getgid(),
	}
}
