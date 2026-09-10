package index

import (
	"bytes"

	"github.com/google/btree"
	"github.com/spnebula/bitcask-kv-go/data"
)

type Index interface {
	Get(key []byte) *data.LogRecordPos           // 根据 key 获取数据位置信息
	Put(key []byte, pos *data.LogRecordPos) bool // 向索引中存储 key 对应的数据位置信息
	Delete(key []byte) bool                      // 从索引中删除 key 对应的数据位置信息
}

type Item struct {
	key []byte
	pos *data.LogRecordPos
}

func (i *Item) Less(b btree.Item) bool {
	return bytes.Compare(i.key, b.(*Item).key) == -1
}
