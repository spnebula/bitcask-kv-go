package index

import (
	"testing"

	"github.com/spnebula/bitcask-kv-go/data"
	"github.com/stretchr/testify/require"
)

func TestAdaptiveRadixTree_Put(t *testing.T) {
	art := NewAdaptiveRadixTree()
	art.Put([]byte("key-1"), &data.LogRecordPos{Fid: 1, Offset: 12})
	art.Put([]byte("key-2"), &data.LogRecordPos{Fid: 1, Offset: 12})
	art.Put([]byte("key-3"), &data.LogRecordPos{Fid: 1, Offset: 12})
}

func TestAdaptiveRadixTree_Get(t *testing.T) {
	art := NewAdaptiveRadixTree()
	art.Put([]byte("key-1"), &data.LogRecordPos{Fid: 1, Offset: 12})
	pos := art.Get([]byte("key-1"))
	require.NotNil(t, pos)

	pos1 := art.Get([]byte("not exist"))
	require.Nil(t, pos1)

	art.Put([]byte("key-1"), &data.LogRecordPos{Fid: 1123, Offset: 990})
	pos2 := art.Get([]byte("key-1"))
	require.NotNil(t, pos2)
}

func TestAdaptiveRadixTree_Delete(t *testing.T) {
	art := NewAdaptiveRadixTree()

	res1 := art.Delete([]byte("not exist"))
	require.False(t, res1)

	art.Put([]byte("key-1"), &data.LogRecordPos{Fid: 1, Offset: 12})
	res2 := art.Delete([]byte("key-1"))
	require.True(t, res2)
	pos := art.Get([]byte("key-1"))
	require.Nil(t, pos)
}

func TestAdaptiveRadixTree_Size(t *testing.T) {
	art := NewAdaptiveRadixTree()

	require.Equal(t, 0, art.Size())

	art.Put([]byte("key-1"), &data.LogRecordPos{Fid: 1, Offset: 12})
	art.Put([]byte("key-2"), &data.LogRecordPos{Fid: 1, Offset: 12})
	art.Put([]byte("key-1"), &data.LogRecordPos{Fid: 1, Offset: 12})
	require.Equal(t, 2, art.Size())
}

func TestAdaptiveRadixTree_Iterator(t *testing.T) {
	art := NewAdaptiveRadixTree()

	art.Put([]byte("ccde"), &data.LogRecordPos{Fid: 1, Offset: 12})
	art.Put([]byte("adse"), &data.LogRecordPos{Fid: 1, Offset: 12})
	art.Put([]byte("bbde"), &data.LogRecordPos{Fid: 1, Offset: 12})
	art.Put([]byte("bade"), &data.LogRecordPos{Fid: 1, Offset: 12})

	iter := art.Iterator(true)
	for iter.Rewind(); iter.Valid(); iter.Next() {
		require.NotNil(t, iter.Key())
		require.NotNil(t, iter.Value())
	}
}
