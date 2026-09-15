package bitcaskkvgo

import "github.com/spnebula/bitcask-kv-go/index"

type Options struct {
	DirPath      string // DirPath 表示数据文件目录
	DataFileSize int64  // DataFileSize 表示数据文件大小

	SyncWrites   bool
	MemIndexType index.IndexerType
}

var DefaultOptions = &Options{
	DirPath:      "/tmp",
	DataFileSize: 64 * 1024 * 1024,
	SyncWrites:   false,
	MemIndexType: index.BTreeType,
}

type IteratorOptions struct {
	Prefix  []byte // Prefix presents the prefix of the keys to iterate over.
	Reverse bool   // Reverse indicates whether to iterate over the keys in reverse order.
}

var DefaultIteratorOptions = IteratorOptions{
	Prefix:  nil,
	Reverse: false,
}
