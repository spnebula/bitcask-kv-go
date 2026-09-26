package bitcaskkvgo

import (
	"os"
	"sync"
	"testing"

	"github.com/spnebula/bitcask-kv-go/utils"
	"github.com/stretchr/testify/require"
)

// 没有任何数据的情况下进行 merge
func TestDB_Merge(t *testing.T) {
	opts := DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-merge-1")
	opts.DirPath = dir
	db, err := OpenDB(opts)
	defer destroyDB(db)
	require.Nil(t, err)
	require.NotNil(t, db)

	err = db.Merge()
	require.Nil(t, err)
}

// 全部都是有效的数据
func TestDB_Merge2(t *testing.T) {
	opts := DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-merge-2")
	opts.DataFileSize = 32 * 1024 * 1024
	opts.DataFileMergeRatio = 0
	opts.DirPath = dir
	db, err := OpenDB(opts)
	defer destroyDB(db)
	require.Nil(t, err)
	require.NotNil(t, db)

	for i := 0; i < 50000; i++ {
		err := db.Put(utils.GetTestKey(i), utils.RandomValue(1024))
		require.Nil(t, err)
	}

	err = db.Merge()
	require.Nil(t, err)

	// 重启校验
	err = db.Close()
	require.Nil(t, err)

	db2, err := OpenDB(opts)
	defer func() {
		_ = db2.Close()
	}()
	require.Nil(t, err)
	keys := db2.ListKeys()
	require.Equal(t, 50000, len(keys))

	for i := 0; i < 50000; i++ {
		val, err := db2.Get(utils.GetTestKey(i))
		require.Nil(t, err)
		require.NotNil(t, val)
	}
}

// 有失效的数据，和被重复 Put 的数据
func TestDB_Merge3(t *testing.T) {
	opts := DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-merge-3")
	opts.DataFileSize = 32 * 1024 * 1024
	opts.DataFileMergeRatio = 0
	opts.DirPath = dir
	db, err := OpenDB(opts)
	defer destroyDB(db)
	require.Nil(t, err)
	require.NotNil(t, db)

	for i := 0; i < 50000; i++ {
		err := db.Put(utils.GetTestKey(i), utils.RandomValue(1024))
		require.Nil(t, err)
	}
	for i := 0; i < 10000; i++ {
		err := db.Delete(utils.GetTestKey(i))
		require.Nil(t, err)
	}
	for i := 40000; i < 50000; i++ {
		err := db.Put(utils.GetTestKey(i), []byte("new value in merge"))
		require.Nil(t, err)
	}

	err = db.Merge()
	require.Nil(t, err)

	// 重启校验
	err = db.Close()
	require.Nil(t, err)

	db2, err := OpenDB(opts)
	defer func() {
		_ = db2.Close()
	}()
	require.Nil(t, err)
	keys := db2.ListKeys()
	require.Equal(t, 40000, len(keys))

	for i := 0; i < 10000; i++ {
		_, err := db2.Get(utils.GetTestKey(i))
		require.Equal(t, ErrKeyNotFound, err)
	}
	for i := 40000; i < 50000; i++ {
		val, err := db2.Get(utils.GetTestKey(i))
		require.Nil(t, err)
		require.Equal(t, []byte("new value in merge"), val)
	}
}

// 全部是无效的数据
func TestDB_Merge4(t *testing.T) {
	opts := DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-merge-4")
	opts.DataFileSize = 32 * 1024 * 1024
	opts.DataFileMergeRatio = 0
	opts.DirPath = dir
	db, err := OpenDB(opts)
	defer destroyDB(db)
	require.Nil(t, err)
	require.NotNil(t, db)

	for i := 0; i < 50000; i++ {
		err := db.Put(utils.GetTestKey(i), utils.RandomValue(1024))
		require.Nil(t, err)
	}
	for i := 0; i < 50000; i++ {
		err := db.Delete(utils.GetTestKey(i))
		require.Nil(t, err)
	}

	err = db.Merge()
	require.Nil(t, err)

	// 重启校验
	err = db.Close()
	require.Nil(t, err)

	db2, err := OpenDB(opts)
	defer func() {
		_ = db2.Close()
	}()
	require.Nil(t, err)
	keys := db2.ListKeys()
	require.Equal(t, 0, len(keys))
}

// Merge 的过程中有新的数据写入或删除
func TestDB_Merge5(t *testing.T) {
	opts := DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-merge-5")
	opts.DataFileSize = 32 * 1024 * 1024
	opts.DataFileMergeRatio = 0
	opts.DirPath = dir
	db, err := OpenDB(opts)
	defer destroyDB(db)
	require.Nil(t, err)
	require.NotNil(t, db)

	for i := 0; i < 50000; i++ {
		err := db.Put(utils.GetTestKey(i), utils.RandomValue(1024))
		require.Nil(t, err)
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50000; i++ {
			err := db.Delete(utils.GetTestKey(i))
			require.Nil(t, err)
		}
		for i := 60000; i < 70000; i++ {
			err := db.Put(utils.GetTestKey(i), utils.RandomValue(1024))
			require.Nil(t, err)
		}
	}()
	err = db.Merge()
	require.Nil(t, err)
	wg.Wait()

	//重启校验
	err = db.Close()
	require.Nil(t, err)

	db2, err := OpenDB(opts)
	defer func() {
		_ = db2.Close()
	}()
	require.Nil(t, err)
	keys := db2.ListKeys()
	require.Equal(t, 10000, len(keys))

	for i := 60000; i < 70000; i++ {
		val, err := db2.Get(utils.GetTestKey(i))
		require.Nil(t, err)
		require.NotNil(t, val)
	}
}
