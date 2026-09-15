package index

import (
	"bytes"
	"sort"
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

func (bt *BTree) Iterator(reverse bool) Iterator {
	if bt.tree == nil {
		return nil
	}

	// lock avoid concurrent conflicts
	bt.lock.RLock()
	defer bt.lock.RUnlock()
	return NewBreeIterator(bt, reverse)
}

func (bt *BTree) Get(key []byte) *data.LogRecordPos {
	item := Item{
		key: key,
	}
	// 加锁，防止 BTree 的 Get 操作并发冲突
	bt.lock.RLock()
	defer bt.lock.RUnlock()
	result := bt.tree.Get(&item)
	if result == nil {
		return nil
	}
	return result.(*Item).pos
}

func (bt *BTree) Put(key []byte, pos *data.LogRecordPos) bool {
	item := Item{
		key: key,
		pos: pos,
	}
	bt.lock.Lock()
	bt.tree.ReplaceOrInsert(&item)
	bt.lock.Unlock()

	return true
}

func (bt *BTree) Delete(key []byte) bool {
	item := Item{
		key: key,
	}
	bt.lock.Lock()
	oldItem := bt.tree.Delete(&item)
	if oldItem == nil {
		bt.lock.Unlock()
		return false
	}
	bt.lock.Unlock()

	return true
}

func (bt *BTree) Size() int {
	return bt.tree.Len()
}

type btreeIterator struct {
	currIndex int     // currIndex position in values
	reverse   bool    // reverse iterator
	values    []*Item // values key and index info in the tree
}

func NewBreeIterator(bt *BTree, reverse bool) Iterator {
	var idx int
	var values = make([]*Item, bt.tree.Len())

	saveValues := func(i btree.Item) bool {
		values[idx] = i.(*Item)
		idx++
		return true
	}

	if reverse {
		bt.tree.Descend(saveValues)
	} else {
		bt.tree.Ascend(saveValues)
	}
	return &btreeIterator{
		currIndex: 0,
		reverse:   reverse,
		values:    values,
	}

}

func (bi *btreeIterator) Rewind() {
	bi.currIndex = 0
}

func (bi *btreeIterator) Seek(key []byte) {
	if bi.reverse {
		bi.currIndex = sort.Search(len(bi.values), func(i int) bool {
			return bytes.Compare(bi.values[i].key, key) <= 0 // grab the first key that is less than or equal to the key
		})
	} else {
		bi.currIndex = sort.Search(len(bi.values), func(i int) bool {
			return bytes.Compare(bi.values[i].key, key) >= 0 // grab the first key that is greater than or equal to the key
		})
	}
}

func (bti *btreeIterator) Next() {
	bti.currIndex += 1
}

func (bti *btreeIterator) Valid() bool {
	return bti.currIndex < len(bti.values)
}

func (bti *btreeIterator) Key() []byte {
	return bti.values[bti.currIndex].key
}

func (bti *btreeIterator) Value() *data.LogRecordPos {
	return bti.values[bti.currIndex].pos
}

func (bti *btreeIterator) Close() {
	bti.values = nil
}
