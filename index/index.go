package index

import (
	"bytes"

	"github.com/google/btree"
	"github.com/spnebula/bitcask-kv-go/data"
)

type Indexer interface {
	Get(key []byte) *data.LogRecordPos                         // 根据 key 获取数据位置信息
	Put(key []byte, pos *data.LogRecordPos) *data.LogRecordPos // 向索引中存储 key 对应的数据位置信息
	Delete(key []byte) (*data.LogRecordPos, bool)              // 从索引中删除 key 对应的数据位置信息
	Iterator(reverse bool) Iterator
	Size() int
}

type IndexerType = uint8

const (
	BTreeType IndexerType = iota
	ArtTreeType
	BPlusTreeType
)

func NewIndex(indexType IndexerType, dirPath string, syncWriter bool) Indexer {
	switch indexType {
	case BTreeType:
		return NewBTree(16)
	case ArtTreeType:
		return NewAdaptiveRadixTree()
	case BPlusTreeType:
		return NewBPlusTree(dirPath, syncWriter)
	default:
		return nil
	}
}

type Item struct {
	key []byte
	pos *data.LogRecordPos
}

func (i *Item) Less(b btree.Item) bool {
	return bytes.Compare(i.key, b.(*Item).key) == -1
}

// Iterator 通用索引迭代器
type Iterator interface {
	// Rewind 重新回到迭代器的起点，即第一个数据
	Rewind()

	// Seek 根据传入的 key 查找到第一个大于（或小于）等于的目标 key，根据从这个 key 开始遍历
	Seek(key []byte)

	// Next 跳转到下一个 key
	Next()

	// Valid 是否有效，即是否已经遍历完了所有的 key，用于退出遍历
	Valid() bool

	// Key 当前遍历位置的 Key 数据
	Key() []byte

	// Value 当前遍历位置的 Value 数据
	Value() *data.LogRecordPos

	// Close 关闭迭代器，释放相应资源
	Close()
}
