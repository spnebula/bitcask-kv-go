package index

import (
	"testing"

	"github.com/spnebula/bitcask-kv-go/data"
	"github.com/stretchr/testify/assert"
)

func TestBTree_Put(t *testing.T) {
	bt := NewBTree(2)

	res1 := bt.Put(nil, &data.LogRecordPos{Fid: 1, Offset: 100})
	assert.True(t, res1)

	res2 := bt.Put([]byte("key1"), &data.LogRecordPos{Fid: 2, Offset: 200})
	assert.True(t, res2)
}

func TestBTree_Get(t *testing.T) {
	bt := NewBTree(2)

	bt.Put([]byte("key1"), &data.LogRecordPos{Fid: 2, Offset: 200})

	pos := bt.Get([]byte("key1"))
	assert.NotNil(t, pos)
	assert.Equal(t, uint32(2), pos.Fid)
	assert.Equal(t, int64(200), pos.Offset)

	posNil := bt.Get([]byte("nonexistent"))
	assert.Nil(t, posNil)

	res := bt.Put([]byte("a"), &data.LogRecordPos{Fid: 3, Offset: 300})
	assert.True(t, res)

	posA := bt.Get([]byte("a"))
	assert.NotNil(t, posA)
	assert.Equal(t, uint32(3), posA.Fid)
	assert.Equal(t, int64(300), posA.Offset)

	t.Log("BTree Get test passed")
}

func TestBTree_Delete(t *testing.T) {
	bt := NewBTree(2)

	bt.Put([]byte("key1"), &data.LogRecordPos{Fid: 2, Offset: 200})

	res := bt.Delete([]byte("key1"))
	assert.True(t, res)

	pos := bt.Get([]byte("key1"))
	assert.Nil(t, pos)

	var resNonExistent bool = bt.Delete([]byte("nonexistent"))
	assert.False(t, resNonExistent)

	t.Log("BTree Delete test passed")
}
