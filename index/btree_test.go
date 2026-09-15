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

func TestBTree_Iterator(t *testing.T) {
	bt1 := NewBTree(32)
	// 1.BTree 为空的情况
	iter1 := bt1.Iterator(false)
	assert.Equal(t, false, iter1.Valid())

	//	2.BTree haa data
	bt1.Put([]byte("ccde"), &data.LogRecordPos{Fid: 1, Offset: 10})
	iter2 := bt1.Iterator(false)
	assert.Equal(t, true, iter2.Valid())
	assert.NotNil(t, iter2.Key())
	assert.NotNil(t, iter2.Value())
	iter2.Next()
	assert.Equal(t, false, iter2.Valid())

	// 3. BTree has multiple data
	bt1.Put([]byte("acee"), &data.LogRecordPos{Fid: 1, Offset: 10})
	bt1.Put([]byte("eede"), &data.LogRecordPos{Fid: 1, Offset: 10})
	bt1.Put([]byte("bbcd"), &data.LogRecordPos{Fid: 1, Offset: 10})
	iter3 := bt1.Iterator(false)
	for iter3.Rewind(); iter3.Valid(); iter3.Next() {
		assert.NotNil(t, iter3.Key())
		t.Log(string(iter3.Key()))
	}
	t.Log("\n")

	iter4 := bt1.Iterator(true)
	for iter4.Rewind(); iter4.Valid(); iter4.Next() {
		assert.NotNil(t, iter4.Key())
		t.Log(string(iter4.Key()))
	}
	t.Log("\n")

	// 4.test seek
	iter5 := bt1.Iterator(false)
	for iter5.Seek([]byte("cc")); iter5.Valid(); iter5.Next() {
		assert.NotNil(t, iter5.Key())
		t.Log(string(iter5.Key()))
	}
	t.Log("\n")

	// 5.reverse seek
	iter6 := bt1.Iterator(true)
	for iter6.Seek([]byte("zz")); iter6.Valid(); iter6.Next() {
		assert.NotNil(t, iter6.Key())
		t.Log(string(iter6.Key()))
	}
}
