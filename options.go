package bitcaskkvgo

import "github.com/spnebula/bitcask-kv-go/index"

type Options struct {
	DirPath      string // DirPath 表示数据文件目录
	DataFileSize int64  // DataFileSize 表示数据文件大小

	SyncWrites   bool
	BytesPerSync uint // BytesPerSync 累计写到多少字节进行持久化
	MemIndexType index.IndexerType

	MMapAtStartup bool // MMapAtStartup 是否在启动时将数据文件映射到内存

	DataFileMergeRatio float32 // DataFileMergeRatio 表示数据文件合并比例
}

var DefaultOptions = &Options{
	DirPath:            "/tmp",
	DataFileSize:       64 * 1024 * 1024,
	SyncWrites:         false,
	MemIndexType:       index.BTreeType,
	MMapAtStartup:      true,
	DataFileMergeRatio: 0.5,
}

type IteratorOptions struct {
	Prefix  []byte // Prefix presents the prefix of the keys to iterate over.
	Reverse bool   // Reverse indicates whether to iterate over the keys in reverse order.
}

var DefaultIteratorOptions = IteratorOptions{
	Prefix:  nil,
	Reverse: false,
}

// WriteBatchOptions declares the options for WriteBatch
type WriteBatchOptions struct {
	MaxBatchNum uint // MaxBatchNum presents the maximum number of writes in a batch.
	SyncWrites  bool // SyncWrites indicates whether to sync the writes to disk.
}

var DefaultWriteBatchOptions = WriteBatchOptions{
	MaxBatchNum: 10000,
	SyncWrites:  true,
}
