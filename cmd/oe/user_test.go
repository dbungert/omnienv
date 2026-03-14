package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUserIDs(t *testing.T) {
	info := CurrentUserInfo()

	assert.Equal(t, info.uid, os.Getuid())
	assert.Equal(t, info.gid, os.Getgid())
}
