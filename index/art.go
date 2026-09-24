package index

import (
	"bytes"
	"sort"
	"sync"

	goart "github.com/plar/go-adaptive-radix-tree"
	"github.com/spnebula/bitcask-kv-go/data"
)

type AdaptiveRadixTree struct {
	lock *sync.RWMutex
	tree goart.Tree
}

func NewAdaptiveRadixTree() *AdaptiveRadixTree {
	return &AdaptiveRadixTree{
		lock: new(sync.RWMutex),
		tree: goart.New(),
	}
}

func (a *AdaptiveRadixTree) Get(key []byte) *data.LogRecordPos {
	a.lock.RLock()
	defer a.lock.RUnlock()
	val, ok := a.tree.Search(key)
	if !ok {
		return nil
	}

	return val.(*data.LogRecordPos)
}

func (a *AdaptiveRadixTree) Put(key []byte, pos *data.LogRecordPos) bool {
	a.lock.Lock()
	a.tree.Insert(key, pos)
	a.lock.Unlock()

	return true
}

func (a *AdaptiveRadixTree) Delete(key []byte) bool {
	a.lock.Lock()
	defer a.lock.Unlock()
	_, ok := a.tree.Delete(key)
	if !ok {
		return false
	}
	return true
}

func (a *AdaptiveRadixTree) Iterator(reverse bool) Iterator {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return newARTIterator(a.tree, reverse)
}

func (a *AdaptiveRadixTree) Size() int {
	return a.tree.Size()
}

// Art 索引迭代器
type artIterator struct {
	currIndex int     // 当前遍历的下标位置
	reverse   bool    // 是否是反向遍历
	values    []*Item // key+位置索引信息
}

func newARTIterator(tree goart.Tree, reverse bool) *artIterator {
	var idx int
	if reverse {
		idx = tree.Size() - 1
	}
	values := make([]*Item, tree.Size())
	saveValues := func(node goart.Node) bool {
		item := &Item{
			key: node.Key(),
			pos: node.Value().(*data.LogRecordPos),
		}
		values[idx] = item
		if reverse {
			idx--
		} else {
			idx++
		}
		return true
	}

	tree.ForEach(saveValues)

	return &artIterator{
		currIndex: 0,
		reverse:   reverse,
		values:    values,
	}
}

func (ai *artIterator) Rewind() {
	ai.currIndex = 0
}

func (ai *artIterator) Seek(key []byte) {
	if ai.reverse {
		ai.currIndex = sort.Search(len(ai.values), func(i int) bool {
			return bytes.Compare(ai.values[i].key, key) <= 0
		})
	} else {
		ai.currIndex = sort.Search(len(ai.values), func(i int) bool {
			return bytes.Compare(ai.values[i].key, key) >= 0
		})
	}
}

func (ai *artIterator) Next() {
	ai.currIndex += 1
}

func (ai *artIterator) Valid() bool {
	return ai.currIndex < len(ai.values)
}

func (ai *artIterator) Key() []byte {
	return ai.values[ai.currIndex].key
}

func (ai *artIterator) Value() *data.LogRecordPos {
	return ai.values[ai.currIndex].pos
}

func (ai *artIterator) Close() {
	ai.values = nil
}
