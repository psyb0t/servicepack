package main

import (
	"testing"

	"github.com/psyb0t/ctxscope"
	"github.com/stretchr/testify/assert"
)

const (
	testBinary  = "test-binary"
	testCommit  = "deadbeef"
	testVersion = "v1.2.3"
)

func TestSetProcessScope(t *testing.T) {
	ctxscope.RemoveGlobal(scopeKeyBinary, scopeKeyCommit, scopeKeyVersion)
	t.Cleanup(func() {
		ctxscope.RemoveGlobal(scopeKeyBinary, scopeKeyCommit, scopeKeyVersion)
	})

	setProcessScope(testBinary, testCommit, testVersion)

	scope := ctxscope.GetGlobal()
	assert.Equal(t, testBinary, scope[scopeKeyBinary])
	assert.Equal(t, testCommit, scope[scopeKeyCommit])
	assert.Equal(t, testVersion, scope[scopeKeyVersion])
}
