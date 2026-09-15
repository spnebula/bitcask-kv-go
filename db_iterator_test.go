package bitcaskkvgo

import (
	"os"
	"testing"

	"github.com/spnebula/bitcask-kv-go/utils"
	"github.com/stretchr/testify/require"
)

func TestDB_NewIterator(t *testing.T) {
	opts := DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-iterator-1")
	opts.DirPath = dir
	db, err := OpenDB(opts)
	defer destroyDB(db)
	require.Nil(t, err)
	require.NotNil(t, db)

	iterator := db.NewIterator(DefaultIteratorOptions)
	require.NotNil(t, iterator)
	require.Equal(t, false, iterator.Valid())
}

func TestDB_Iterator_One_Value(t *testing.T) {
	opts := DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-iterator-2")
	opts.DirPath = dir
	db, err := OpenDB(opts)
	defer destroyDB(db)
	require.Nil(t, err)
	require.NotNil(t, db)

	err = db.Put(utils.GetTestKey(10), utils.GetTestKey(10))
	require.Nil(t, err)

	iterator := db.NewIterator(DefaultIteratorOptions)
	require.NotNil(t, iterator)
	require.Equal(t, true, iterator.Valid())
	require.Equal(t, utils.GetTestKey(10), iterator.Key())
	val, err := iterator.Value()
	require.Nil(t, err)
	require.Equal(t, utils.GetTestKey(10), val)
}

func TestDB_Iterator_Multi_Values(t *testing.T) {
	opts := DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-iterator-3")
	opts.DirPath = dir
	db, err := OpenDB(opts)
	defer destroyDB(db)
	require.Nil(t, err)
	require.NotNil(t, db)

	err = db.Put([]byte("annde"), utils.RandomValue(10))
	require.Nil(t, err)
	err = db.Put([]byte("cnedc"), utils.RandomValue(10))
	require.Nil(t, err)
	err = db.Put([]byte("aeeue"), utils.RandomValue(10))
	require.Nil(t, err)
	err = db.Put([]byte("esnue"), utils.RandomValue(10))
	require.Nil(t, err)
	err = db.Put([]byte("bnede"), utils.RandomValue(10))
	require.Nil(t, err)

	// 正向迭代
	iter1 := db.NewIterator(DefaultIteratorOptions)
	for iter1.Rewind(); iter1.Valid(); iter1.Next() {
		require.NotNil(t, iter1.Key())
	}
	iter1.Rewind()
	for iter1.Seek([]byte("c")); iter1.Valid(); iter1.Next() {
		require.NotNil(t, iter1.Key())
	}

	// 反向迭代
	iterOpts1 := DefaultIteratorOptions
	iterOpts1.Reverse = true
	iter2 := db.NewIterator(iterOpts1)
	for iter2.Rewind(); iter2.Valid(); iter2.Next() {
		require.NotNil(t, iter2.Key())
	}
	iter2.Rewind()
	for iter2.Seek([]byte("c")); iter2.Valid(); iter2.Next() {
		require.NotNil(t, iter2.Key())
	}

	// 指定了 prefix
	iterOpts2 := DefaultIteratorOptions
	iterOpts2.Prefix = []byte("aee")
	iter3 := db.NewIterator(iterOpts2)
	for iter3.Rewind(); iter3.Valid(); iter3.Next() {
		require.NotNil(t, iter3.Key())
	}
}
