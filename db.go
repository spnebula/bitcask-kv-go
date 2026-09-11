package bitcaskkvgo

import (
	"sync"

	"github.com/spnebula/bitcask-kv-go/data"
	"github.com/spnebula/bitcask-kv-go/index"
)

// DB represents the Bitcask key-value store.
type DB struct {
	mtx        *sync.RWMutex
	activeFile *data.DataFile            // activeFile is the current data file where new log records are appended.
	olderFiles map[uint32]*data.DataFile // olderFiles holds references to older data files for reading existing log records.

	index index.Index

	options *Options
}

func NewDB() *DB {
	return &DB{
		mtx:        &sync.RWMutex{},
		olderFiles: make(map[uint32]*data.DataFile),
	}
}

// Put inserts a new log record into the Bitcask key-value store.
func (db *DB) Put(key []byte, value []byte) error {
	if len(key) == 0 { // key cannot be empty
		return ErrKeyEmpty
	}

	var log_record = &data.LogRecord{
		Key:   key,
		Value: value,
		Type:  data.LogRecordNormal,
	}

	pos, err := db.appendLogRecord(log_record)
	if err != nil {
		return err
	}

	if ok := db.index.Put(key, pos); !ok {
		return ErrIndexUpdateFailed
	}

	return nil
}

// Get retrieves the value associated with the given key from the Bitcask key-value store.
func (db *DB) Get(key []byte) ([]byte, error) {
	if len(key) == 0 { // key cannot be empty
		return nil, ErrKeyEmpty
	}

	db.mtx.RLock()
	defer db.mtx.RUnlock()

	pos := db.index.Get(key)
	if pos == nil {
		return nil, ErrKeyNotFound
	}

	// acording to file id, read the data from the corresponding data file
	data_file := db.getLogRecord(pos.Fid)
	if data_file == nil {
		return nil, ErrDataFileNotFound
	}

	log_record, err := data_file.ReadLogRecord(pos.Offset)
	if err != nil {
		return nil, err
	}
	if log_record.Type == data.LogRecordDeleted {
		return nil, ErrKeyNotFound
	}

	return log_record.Value, nil
}

func (db *DB) getLogRecord(fid uint32) *data.DataFile {
	if db.activeFile.FileID == fid {
		return db.activeFile
	} else {
		return db.olderFiles[fid]
	}
}

// Delete removes the log record associated with the given key from the Bitcask key-value store.
func (db *DB) Delete(key []byte) error {
	return nil
}

// Close closes the Bitcask key-value store.
func (db *DB) Close() error {
	db.mtx.Lock()
	defer db.mtx.Unlock()
	for _, dataFile := range db.olderFiles {
		if err := dataFile.Close(); err != nil {
			return err
		}
	}

	err := db.activeFile.Close()
	return err
}

// appendLogRecord appends a new log record to the active data file.
func (db *DB) appendLogRecord(log_record *data.LogRecord) (*data.LogRecordPos, error) {
	encRecord, size := data.EncodeLogRecord(log_record)

	db.mtx.Lock()
	defer db.mtx.Unlock()

	if db.activeFile == nil {
		if err := db.setActiveDataFile(); err != nil {
			return nil, err
		}
	}

	//
	if db.activeFile.WriteOffset+size > db.options.DataFileSize {
		// sync active file
		if err := db.activeFile.Sync(); err != nil {
			return nil, err
		}

		// convert active file to old file
		db.olderFiles[db.activeFile.FileID] = db.activeFile

		// open new active file
		if err := db.setActiveDataFile(); err != nil {
			return nil, err
		}
	}

	writeOffset := db.activeFile.WriteOffset
	err := db.activeFile.Write(encRecord)
	if err != nil {
		return nil, err
	}

	// acording to user config
	if db.options.SyncWrites {
		if err := db.activeFile.Sync(); err != nil {
			return nil, err
		}
	}

	pos := &data.LogRecordPos{
		Fid:    db.activeFile.FileID,
		Offset: writeOffset,
	}
	return pos, nil
}

func (db *DB) setActiveDataFile() error {
	var initFileID uint32 = 0
	// Implement the logic to append the log record to the data file and update the index accordingly.
	if db.activeFile == nil {
		initFileID = db.activeFile.FileID + 1
	}

	dataFile, err := data.OpenDataFile(db.options.DirPath, initFileID)
	if err != nil {
		return err
	}

	db.activeFile = dataFile

	return nil
}
