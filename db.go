package bitcaskkvgo

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gofrs/flock"
	"github.com/spnebula/bitcask-kv-go/data"
	"github.com/spnebula/bitcask-kv-go/fio"
	"github.com/spnebula/bitcask-kv-go/index"
	"github.com/spnebula/bitcask-kv-go/utils"
)

const (
	DataFileNameSuffix = ".data"
)

const (
	seqNoKey     = "seq.no"
	fileLockName = "flock"
)

// DB represents the Bitcask key-value store.
type DB struct {
	options *Options

	mtx        *sync.RWMutex
	fileIDs    []int                     // fileIDs holds the file IDs of the data files in the Bitcask key-value store.
	activeFile *data.DataFile            // activeFile is the current data file where new log records are appended.
	olderFiles map[uint32]*data.DataFile // olderFiles holds references to older data files for reading existing log records.

	index index.Indexer

	seqNo           uint64 // transaction sequence number
	seqNoFileExists bool   // whether the seq.no file exists
	isFirstInitial  bool   // whether the db is first initialized

	isMerge bool

	fileLock   *flock.Flock // file lock for the database
	bytesWrite uint         // number of bytes written to the database

	reclaminSize uint64 // reclamation size
}

type Stat struct {
	KeyNum          uint  // key total number
	DataFileNum     uint  // data file number
	ReclaimableSize int64 // can merge data size
	DiskSize        int64 // data dir size in disk
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
		db.isFirstInitial = true
		if err := os.Mkdir(options.DirPath, os.ModePerm); err != nil {
			return nil, err
		}
	}

	fileLock := flock.New(filepath.Join(options.DirPath, fileLockName))
	hold, err := fileLock.TryLock()
	if err != nil {
		return nil, err
	}
	if !hold {
		return nil, ErrDatabaseIsUsing
	}
	db.fileLock = fileLock

	entries, err := os.ReadDir(options.DirPath)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		db.isFirstInitial = true
	}

	db.options = options
	db.index = index.NewIndex(options.MemIndexType, db.options.DirPath, db.options.SyncWrites)

	// load merge files
	if err := db.loadMergeFiles(); err != nil {
		return nil, err
	}

	// load data files
	if err := db.loadDataFiles(); err != nil {
		return nil, err
	}

	if db.options.MemIndexType != index.BPlusTreeType {
		// load index from hint files
		if err := db.loadIndexFromHintFiles(); err != nil {
			return nil, err
		}

		// load index from data files
		if err := db.loadIndexFromDataFiles(); err != nil {
			return nil, err
		}

		// reset io type to standard io
		if db.options.MMapAtStartup {
			if err := db.resetIoType(); err != nil {
				return nil, err
			}
		}
	}

	if db.options.MemIndexType == index.BPlusTreeType {
		if err := db.loadSeqNo(); err != nil {
			return nil, err
		}
		if db.activeFile != nil {
			size, err := db.activeFile.IoManager.Size()
			if err != nil {
				return nil, err
			}
			db.activeFile.WriteOffset = size
		}
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

	var io_type fio.FileIOType
	for i, fid := range fids {
		io_type = fio.StandardFIO
		if db.options.MMapAtStartup && i != len(fids)-1 {
			io_type = fio.MemoryMap
		}

		data_file, err := data.OpenDataFile(db.options.DirPath, uint32(fid), io_type)
		if err != nil {
			return err
		}
		if i == len(fids)-1 {
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

func (db *DB) checkOptions(options *Options) error {
	if options.DirPath == "" {
		return ErrDirPathEmpty
	}
	if options.DataFileSize <= 0 {
		return ErrDataFileSizeEmpty
	}
	if options.DataFileMergeRatio < 0 || options.DataFileMergeRatio > 1 {
		return ErrDataFileMergeRatioInvalid
	}

	return nil
}

func (db *DB) Stat() *Stat {
	db.mtx.RLock()
	defer db.mtx.RUnlock()

	var data_files_num = uint(len(db.olderFiles))
	if db.activeFile != nil {
		data_files_num++
	}

	dirSize, err := utils.DirSize(db.options.DirPath)
	if err != nil {
		panic(fmt.Sprintf("failed to get dir size: %v", err))
	}

	return &Stat{
		KeyNum:          uint(db.index.Size()),
		DataFileNum:     data_files_num,
		ReclaimableSize: int64(db.reclaminSize),
		DiskSize:        dirSize,
	}
}

// Put inserts a new log record into the Bitcask key-value store.
func (db *DB) Put(key []byte, value []byte) error {
	if len(key) == 0 { // key cannot be empty
		return ErrKeyIsEmpty
	}

	var log_record = &data.LogRecord{
		Key:   logRecordKeyWithSeq(key, nonTransactionSeqNo),
		Value: value,
		Type:  data.LogRecordNormal,
	}
	db.mtx.Lock()
	defer db.mtx.Unlock()

	pos, err := db.appendLogRecord(log_record)
	if err != nil {
		return err
	}

	if old_pos := db.index.Put(key, pos); old_pos != nil {
		db.reclaminSize += uint64(old_pos.Size)
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
	iterator.Close()
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
	iterator.Close()
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
		Key:   logRecordKeyWithSeq(key, nonTransactionSeqNo),
		Value: nil,
		Type:  data.LogRecordDeleted,
	}

	pos, err := db.appendLogRecord(log_record)
	if err != nil {
		return err
	}
	db.reclaminSize += uint64(pos.Size)

	old_pos, ok := db.index.Delete(key)
	if !ok {
		return ErrIndexUpdateFailed
	}
	if old_pos != nil {
		db.reclaminSize += uint64(old_pos.Size)
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
	defer func() {
		if err := db.fileLock.Unlock(); err != nil {
			panic(fmt.Sprintf("failed to unlock the directory, %v", err))
		}
	}()
	if db.activeFile == nil {
		return nil
	}

	db.mtx.Lock()
	defer db.mtx.Unlock()

	// save current trx sequence number
	seqNoFile, err := data.OpenSeqNoFile(db.options.DirPath)
	if err != nil {
		return err
	}
	record := &data.LogRecord{
		Key:   []byte(seqNoKey),
		Value: []byte(strconv.FormatUint(db.seqNo, 10)),
	}
	encRecord, _ := data.EncodeLogRecord(record)
	if _, err := seqNoFile.Write(encRecord); err != nil {
		return err
	}
	if err := seqNoFile.Sync(); err != nil {
		return err
	}

	// Close the data files
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
	// bytes written num
	db.bytesWrite += uint(len(encRecord))

	// whether to sync according to user config
	var needSync = db.options.SyncWrites
	if !needSync && db.options.BytesPerSync > 0 && db.bytesWrite >= db.options.BytesPerSync {
		needSync = true
	}
	if needSync {
		if err := db.activeFile.Sync(); err != nil {
			return nil, err
		}
		// 清空累计值
		if db.bytesWrite > 0 {
			db.bytesWrite = 0
		}
	}

	pos := &data.LogRecordPos{
		Fid:    db.activeFile.FileID,
		Offset: writeOffset,
		Size:   uint32(size),
	}
	return pos, nil
}

func (db *DB) setActiveDataFile() error {
	var initFileID uint32 = 0
	// Implement the logic to append the log record to the data file and update the index accordingly.
	if db.activeFile != nil {
		initFileID = db.activeFile.FileID + 1
	}

	dataFile, err := data.OpenDataFile(db.options.DirPath, initFileID, fio.StandardFIO)
	if err != nil {
		return err
	}

	db.activeFile = dataFile

	return nil
}

func (db *DB) loadSeqNo() error {
	fileName := filepath.Join(db.options.DirPath, data.SeqNoFileName)
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return nil
	}

	seqNoFile, err := data.OpenSeqNoFile(db.options.DirPath)
	if err != nil {
		return err
	}
	record, _, err := seqNoFile.ReadLogRecord(0)
	seqNo, err := strconv.ParseUint(string(record.Value), 10, 64)
	if err != nil {
		return err
	}
	db.seqNo = seqNo
	db.seqNoFileExists = true

	return os.Remove(fileName)
}

func (db *DB) resetIoType() error {
	if db.activeFile == nil {
		return nil
	}

	if err := db.activeFile.SetIOManager(db.options.DirPath, fio.StandardFIO); err != nil {
		return err
	}
	for _, dataFile := range db.olderFiles {
		if err := dataFile.SetIOManager(db.options.DirPath, fio.StandardFIO); err != nil {
			return err
		}
	}
	return nil
}
