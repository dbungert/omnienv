package omnienv

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUserIDs(t *testing.T) {
	info := CurrentUserInfo()

	assert.Equal(t, info.UID, os.Getuid())
	assert.Equal(t, info.GID, os.Getgid())
}
