package bitcaskkvgo

import (
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/spnebula/bitcask-kv-go/data"
	"github.com/spnebula/bitcask-kv-go/index"
)

const (
	DataFileNameSuffix = ".data"
)

// DB represents the Bitcask key-value store.
type DB struct {
	mtx        *sync.RWMutex
	fileIDs    []int                     // fileIDs holds the file IDs of the data files in the Bitcask key-value store.
	activeFile *data.DataFile            // activeFile is the current data file where new log records are appended.
	olderFiles map[uint32]*data.DataFile // olderFiles holds references to older data files for reading existing log records.

	index index.Indexer

	options *Options
}

// OpenDB opens a new Bitcask key-value store with the given options.
func OpenDB(options *Options) (*DB, error) {
	db := &DB{
		mtx:        &sync.RWMutex{},
		olderFiles: make(map[uint32]*data.DataFile),
	}

	if err := db.checkOptions(options); err != nil {
		return nil, err
	}

	_, err := os.Stat(options.DirPath)
	if os.IsNotExist(err) {
		if err := os.Mkdir(options.DirPath, os.ModePerm); err != nil {
			return nil, err
		}
	}

	db.options = options
	db.index = index.NewIndex(options.MemIndexType)

	if err := db.loadDataFiles(); err != nil {
		return nil, err
	}

	if err := db.loadIndexFromDataFiles(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) loadDataFiles() error {
	dir_entries, err := os.ReadDir(db.options.DirPath)
	if err != nil {
		return err
	}

	fids := []int{}
	for _, entry := range dir_entries {
		if strings.HasSuffix(entry.Name(), DataFileNameSuffix) {
			fid, err := strconv.ParseUint(strings.TrimSuffix(entry.Name(), DataFileNameSuffix), 10, 32)
			if err != nil {
				return ErrDataDirctCorrupt
			}
			fids = append(fids, int(fid))
		}
	}

	sort.Ints(fids)
	db.fileIDs = fids

	for i, fid := range fids {
		data_file, err := data.OpenDataFile(db.options.DirPath, uint32(fid))
		if err != nil {
			return err
		}
		db.olderFiles[data_file.FileID] = data_file
		if i == len(db.olderFiles)-1 {
			db.activeFile = data_file
		} else {
			db.olderFiles[uint32(fid)] = data_file
		}
	}
	if db.activeFile == nil {
		if err := db.setActiveDataFile(); err != nil {
			return err
		}
	}

	return nil
}

// loadIndexFromDataFiles loads the index from the data files.
func (db *DB) loadIndexFromDataFiles() error {
	if len(db.fileIDs) == 0 {
		return nil
	}

	var data_file *data.DataFile
	// iterate over the data files and load the index from each file
	for i, fid := range db.fileIDs {
		if i == len(db.fileIDs)-1 {
			data_file = db.activeFile
		} else {
			data_file = db.olderFiles[uint32(fid)]
		}

		var offset int64 = 0
		for {
			log_record, size, err := data_file.ReadLogRecord(offset)
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}

			var log_record_pos = &data.LogRecordPos{
				Fid:    data_file.FileID,
				Offset: offset,
			}
			if log_record.Type == data.LogRecordDeleted {
				db.index.Delete(log_record.Key)
			} else {
				db.index.Put(log_record.Key, log_record_pos)
			}
			offset += int64(size)
		}
		if len(db.fileIDs)-1 == i {
			db.activeFile.WriteOffset = offset
		}
	}

	return nil
}

func (db *DB) checkOptions(options *Options) error {
	if options.DirPath == "" {
		return ErrDirPathEmpty
	}
	if options.DataFileSize <= 0 {
		return ErrDataFileSizeEmpty
	}

	return nil
}

// Put inserts a new log record into the Bitcask key-value store.
func (db *DB) Put(key []byte, value []byte) error {
	if len(key) == 0 { // key cannot be empty
		return ErrKeyIsEmpty
	}

	var log_record = &data.LogRecord{
		Key:   key,
		Value: value,
		Type:  data.LogRecordNormal,
	}
	db.mtx.Lock()
	defer db.mtx.Unlock()

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
		return nil, ErrKeyIsEmpty
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

	log_record, _, err := data_file.ReadLogRecord(pos.Offset)
	if err != nil {
		return nil, err
	}
	if log_record.Type == data.LogRecordDeleted {
		return nil, ErrKeyNotFound
	}

	return log_record.Value, nil
}

// ListKeys returns all keys in the Bitcask key-value store.
func (db *DB) ListKeys() [][]byte {
	iterator := db.index.Iterator(false)
	keys := make([][]byte, 0, db.index.Size())
	for iterator.Rewind(); iterator.Valid(); iterator.Next() {
		keys = append(keys, iterator.Key())
	}
	return keys
}

// Fold iterates over all keys in the Bitcask key-value store and calls the given function for each key-value pair.
func (db *DB) Fold(fn func(key []byte, value []byte) bool) error {
	db.mtx.RLock()
	defer db.mtx.RUnlock()

	iterator := db.index.Iterator(false)
	for iterator.Rewind(); iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		value, err := db.getValueByPosition(iterator.Value())
		if err != nil {
			return err
		}
		if !fn(key, value) {
			return nil
		}
	}
	return nil
}

func (db *DB) getValueByPosition(pos *data.LogRecordPos) ([]byte, error) {
	// acording to file id, read the data from the corresponding data file
	data_file := db.getLogRecord(pos.Fid)
	if data_file == nil {
		return nil, ErrDataFileNotFound
	}

	log_record, _, err := data_file.ReadLogRecord(pos.Offset)
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
	if len(key) == 0 { // key cannot be empty
		return ErrKeyIsEmpty
	}
	db.mtx.Lock()
	defer db.mtx.Unlock()

	pos := db.index.Get(key)
	if pos == nil {
		return ErrKeyNotFound
	}

	log_record := &data.LogRecord{
		Key:   key,
		Value: nil,
		Type:  data.LogRecordDeleted,
	}

	_, err := db.appendLogRecord(log_record)
	if err != nil {
		return err
	}

	if ok := db.index.Delete(key); !ok {
		return ErrIndexUpdateFailed
	}
	return nil
}

func (db *DB) Sync() error {
	if db.activeFile == nil {
		return nil
	}
	db.mtx.Lock()
	defer db.mtx.Unlock()
	return db.activeFile.Sync()
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

	if db.activeFile != nil {
		err := db.activeFile.Close()
		return err
	}
	return nil
}

// appendLogRecord appends a new log record to the active data file.
func (db *DB) appendLogRecord(log_record *data.LogRecord) (*data.LogRecordPos, error) {
	encRecord, size := data.EncodeLogRecord(log_record)

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
	_, err := db.activeFile.Write(encRecord)
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
	if db.activeFile != nil {
		initFileID = db.activeFile.FileID + 1
	}

	dataFile, err := data.OpenDataFile(db.options.DirPath, initFileID)
	if err != nil {
		return err
	}

	db.activeFile = dataFile

	return nil
}
