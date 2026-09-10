package index

import (
	"sync"

	"github.com/google/btree"
	"github.com/spnebula/bitcask-kv-go/data"
)

// BTree 索引，封装 Google btree
type BTree struct {
	tree *btree.BTree // BTree 更新操作需要加锁
	lock *sync.RWMutex
}

// NewBTree 创建 BTree 索引
func NewBTree(degree int) *BTree {
	return &BTree{
		tree: btree.New(degree),
		lock: new(sync.RWMutex),
	}
}

func (b *BTree) Get(key []byte) *data.LogRecordPos {
	item := Item{
		key: key,
	}
	result := b.tree.Get(&item)
	if result == nil {
		return nil
	}
	return result.(*Item).pos
}

func (b *BTree) Put(key []byte, pos *data.LogRecordPos) bool {
	item := Item{
		key: key,
		pos: pos,
	}
	b.lock.Lock()
	b.tree.ReplaceOrInsert(&item)
	b.lock.Unlock()

	return true
}

func (b *BTree) Delete(key []byte) bool {
	item := Item{
		key: key,
	}
	b.lock.Lock()
	oldItem := b.tree.Delete(&item)
	if oldItem == nil {
		b.lock.Unlock()
		return false
	}
	b.lock.Unlock()

	return true
}
