package bitcaskkvgo

import (
	"encoding/binary"
	"sync"
	"sync/atomic"

	"github.com/spnebula/bitcask-kv-go/data"
	"github.com/spnebula/bitcask-kv-go/index"
)

const nonTransactionSeqNo = uint64(0)

var txnFinKey = []byte("txn-fin")

// WriteBatch atomic batch of writes
type WriteBatch struct {
	options       WriteBatchOptions
	mu            *sync.Mutex
	db            *DB
	pendingWrites map[string]*data.LogRecord // pending writes
}

func (db *DB) NewWriteBatch(options WriteBatchOptions) *WriteBatch {
	// if the db is first initialized, the seq no file does not exist, can use write batch
	// if the db is not first initialized, the seq no file exists, can not use write batch. It will cause panic, because the db did not close properly last time.
	// BplusTree use write batch only if the correct SeqNoFile exists
	if db.options.MemIndexType == index.BPlusTreeType && !db.seqNoFileExists && !db.isFirstInitial {
		panic("cannot use write batch, seq no file not exists")
	}
	return &WriteBatch{
		options:       options,
		mu:            &sync.Mutex{},
		db:            db,
		pendingWrites: make(map[string]*data.LogRecord),
	}
}

func (wb *WriteBatch) Put(key []byte, value []byte) error {
	if len(key) == 0 { // key cannot be empty
		return ErrKeyIsEmpty
	}

	wb.mu.Lock()
	defer wb.mu.Unlock()

	wb.pendingWrites[string(key)] = &data.LogRecord{
		Key:   key,
		Value: value,
		Type:  data.LogRecordNormal,
	}
	return nil
}

func (wb *WriteBatch) Delete(key []byte) error {
	if len(key) == 0 { // key cannot be empty
		return ErrKeyIsEmpty
	}

	wb.mu.Lock()
	defer wb.mu.Unlock()

	log_pos := wb.db.index.Get(key)
	if log_pos == nil {
		if wb.pendingWrites[string(key)] != nil {
			delete(wb.pendingWrites, string(key))
		}
		return nil
	}

	wb.pendingWrites[string(key)] = &data.LogRecord{
		Key:   key,
		Value: nil,
		Type:  data.LogRecordDeleted,
	}
	return nil
}

func (wb *WriteBatch) Commit() error {

	if len(wb.pendingWrites) == 0 {
		return nil
	}
	if uint(len(wb.pendingWrites)) > wb.options.MaxBatchNum {
		return ErrExceedMaxLogRecordSize
	}

	wb.mu.Lock() // lock protection critical area
	defer wb.mu.Unlock()

	// Get the sequence number of the current transaction
	seq_no := atomic.AddUint64(&wb.db.seqNo, 1)

	// populate the index with the pending writes
	logs_pos := make(map[string]*data.LogRecordPos)
	for _, log_record := range wb.pendingWrites {
		log_record_pos, err := wb.db.appendLogRecord(&data.LogRecord{
			Key:   logRecordKeyWithSeq(log_record.Key, seq_no),
			Value: log_record.Value,
			Type:  log_record.Type,
		})
		if err != nil {
			return err
		}
		logs_pos[string(log_record.Key)] = log_record_pos
	}

	// write a data presenting the transaction to be committed
	finished_record := &data.LogRecord{
		Key:  logRecordKeyWithSeq(txnFinKey, seq_no),
		Type: data.LogRecordFinished,
	}
	_, err := wb.db.appendLogRecord(finished_record)
	if err != nil {
		return err
	}

	// decide whhther to fsync the data file
	if wb.options.SyncWrites && wb.db.activeFile != nil {
		if err := wb.db.activeFile.Sync(); err != nil {
			return err
		}
	}

	for _, record := range wb.pendingWrites {
		pos := logs_pos[string(record.Key)]
		if record.Type == data.LogRecordDeleted {
			wb.db.index.Delete(record.Key)
		} else {
			wb.db.index.Put(record.Key, pos)
		}
	}

	wb.pendingWrites = make(map[string]*data.LogRecord)

	return nil
}

// logRecordKeyWithSeq returns the key with the sequence number.
func logRecordKeyWithSeq(key []byte, seqNo uint64) []byte {
	seq := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(seq[:], seqNo)

	encKey := make([]byte, n+len(key))
	copy(encKey[:n], seq[:n])
	copy(encKey[n:], key)

	return encKey
}

// parseLogRecordKey parses the key with the sequence number.
func parseLogRecordKey(key []byte) ([]byte, uint64) {
	seqNo, n := binary.Uvarint(key)
	realKey := key[n:]
	return realKey, seqNo
}
