package bitcaskkvgo

import (
	"os"
	"testing"

	"github.com/spnebula/bitcask-kv-go/utils"
	"github.com/stretchr/testify/require"
)

func TestDB_WriteBatch1(t *testing.T) {
	opts := DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-batch-1")
	opts.DirPath = dir
	db, err := OpenDB(opts)
	defer destroyDB(db)
	require.Nil(t, err)
	require.NotNil(t, db)

	// 写数据之后并不提交
	wb := db.NewWriteBatch(DefaultWriteBatchOptions)
	err = wb.Put(utils.GetTestKey(1), utils.RandomValue(10))
	require.Nil(t, err)
	err = wb.Delete(utils.GetTestKey(2))
	require.Nil(t, err)

	_, err = db.Get(utils.GetTestKey(1))
	require.Equal(t, ErrKeyNotFound, err)

	// 正常提交数据
	err = wb.Commit()
	require.Nil(t, err)

	val1, err := db.Get(utils.GetTestKey(1))
	require.NotNil(t, val1)
	require.Nil(t, err)

	// 删除有效的数据
	wb2 := db.NewWriteBatch(DefaultWriteBatchOptions)
	err = wb2.Delete(utils.GetTestKey(1))
	require.Nil(t, err)
	err = wb2.Commit()
	require.Nil(t, err)

	_, err = db.Get(utils.GetTestKey(1))
	require.Equal(t, ErrKeyNotFound, err)
}

func TestDB_WriteBatch2(t *testing.T) {
	opts := DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-batch-2")
	opts.DirPath = dir
	db, err := OpenDB(opts)
	defer destroyDB(db)
	require.Nil(t, err)
	require.NotNil(t, db)

	err = db.Put(utils.GetTestKey(1), utils.RandomValue(10))
	require.Nil(t, err)

	wb := db.NewWriteBatch(DefaultWriteBatchOptions)
	err = wb.Put(utils.GetTestKey(2), utils.RandomValue(10))
	require.Nil(t, err)
	err = wb.Delete(utils.GetTestKey(1))
	require.Nil(t, err)

	err = wb.Commit()
	require.Nil(t, err)

	err = wb.Put(utils.GetTestKey(11), utils.RandomValue(10))
	require.Nil(t, err)
	err = wb.Commit()
	require.Nil(t, err)

	// 重启
	err = db.Close()
	require.Nil(t, err)

	db2, err := OpenDB(opts)
	require.Nil(t, err)

	_, err = db2.Get(utils.GetTestKey(1))
	require.Equal(t, ErrKeyNotFound, err)

	// 校验序列号
	require.Equal(t, uint64(2), db.seqNo)
}

//func TestDB_WriteBatch3(t *testing.T) {
//	opts := DefaultOptions
//	//dir, _ := os.MkdirTemp("", "bitcask-go-batch-3")
//	dir := "/tmp/bitcask-go-batch-3"
//	opts.DirPath = dir
//	db, err := OpenDB(opts)
//	//defer destroyDB(db)
//	require.Nil(t, err)
//	require.NotNil(t, db)
//
//	keys := db.ListKeys()
//	t.Log(len(keys))
//	//
//	//wbOpts := DefaultWriteBatchOptions
//	//wbOpts.MaxBatchNum = 10000000
//	//wb := db.NewWriteBatch(wbOpts)
//	//for i := 0; i < 500000; i++ {
//	//	err := wb.Put(utils.GetTestKey(i), utils.RandomValue(1024))
//	//	require.Nil(t, err)
//	//}
//	//err = wb.Commit()
//	//require.Nil(t, err)
//}
