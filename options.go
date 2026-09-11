package bitcaskkvgo

import "github.com/spnebula/bitcask-kv-go/index"

type Options struct {
	DirPath      string // DirPath 表示数据文件目录
	DataFileSize int64  // DataFileSize 表示数据文件大小

	SyncWrites   bool
	MemIndexType index.IndexerType
}
