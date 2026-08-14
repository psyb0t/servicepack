package main

import (
	"testing"

	"github.com/psyb0t/ctxscope"
	"github.com/stretchr/testify/assert"
)

const (
	testBinary = "test-binary"
	testCommit = "deadbeef"
)

func TestSetProcessScope(t *testing.T) {
	ctxscope.RemoveGlobal(scopeKeyBinary, scopeKeyCommit)
	t.Cleanup(func() {
		ctxscope.RemoveGlobal(scopeKeyBinary, scopeKeyCommit)
	})

	setProcessScope(testBinary, testCommit)

	scope := ctxscope.GetGlobal()
	assert.Equal(t, testBinary, scope[scopeKeyBinary])
	assert.Equal(t, testCommit, scope[scopeKeyCommit])
}
