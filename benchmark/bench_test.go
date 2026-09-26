package benchmark

import (
	"os"
	"testing"

	bitcask "github.com/spnebula/bitcask-kv-go"

	"github.com/spnebula/bitcask-kv-go/utils"
	"github.com/stretchr/testify/require"
)

var db *bitcask.DB

func init() {
	// 初始化用于基准测试的存储引擎
	options := bitcask.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-bench")
	options.DirPath = dir

	var err error
	db, err = bitcask.OpenDB(options)
	if err != nil {
		panic(err)
	}
}

func Benchmark_Put(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := db.Put(utils.GetTestKey(i), utils.RandomValue(1024))
		require.Nil(b, err)
	}
}

func Benchmark_Get(b *testing.B) {
	for i := 0; i < 10000; i++ {
		err := db.Put(utils.GetTestKey(i), utils.RandomValue(1024))
		require.Nil(b, err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := db.Get(utils.GetTestKey(i))
		if err != nil && err != bitcask.ErrKeyNotFound {
			b.Fatal(err)
		}
	}
}

func Benchmark_Delete(b *testing.B) {
	// 预写入 b.N 个 key，保证待删除的 key 一定存在。
	// 注意：该准备阶段不计入基准测试的计时。
	for i := 0; i < b.N; i++ {
		require.Nil(b, db.Put(utils.GetTestKey(i), utils.RandomValue(1024)))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := db.Delete(utils.GetTestKey(i))
		require.Nil(b, err)
	}
}
