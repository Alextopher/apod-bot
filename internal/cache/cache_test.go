package cache

import (
	"testing"
)

type Dummy struct{}

func (d Dummy) GetDate() string {
	return ""
}

// Verify that AppendOnly implements the Cache interface
func TestAppendOnlyImplementsCache(t *testing.T) {
	var _ Cache[Dummy] = &AppendOnly[Dummy]{}
}

// Verify that Empty implements the Cache interface
func TestEmptyImplementsCache(t *testing.T) {
	var _ Cache[Dummy] = &Empty[Dummy]{}
}

// Verify that FSCache implements the Cache interface
func TestFSCacheImplementsCache(t *testing.T) {
	var _ Cache[Dummy] = &FSCache[Dummy]{}
}
