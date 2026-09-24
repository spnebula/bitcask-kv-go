package bitcaskkvgo

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/spnebula/bitcask-kv-go/data"
)

const (
	mergeDirName     = "-merge"
	mergeFinishedKey = "merge.finished"
)

func (db *DB) Merge() error {
	if db.activeFile == nil {
		return ErrDataFileNotFound
	}

	db.mtx.Lock()
	if db.isMerge { // if merge is in process, return error
		db.mtx.Unlock()
		return ErrMergeIsInProcess
	}
	db.isMerge = true

	defer func() {
		db.isMerge = false
	}()

	// persist active file
	if err := db.activeFile.Sync(); err != nil {
		db.mtx.Unlock()
		return err
	}

	// convert active file to data file
	db.olderFiles[db.activeFile.FileID] = db.activeFile

	// create new active file
	err := db.setActiveDataFile()
	if err != nil {
		db.mtx.Unlock()
		return err
	}

	// Record file ids that have not recently participated in merge
	non_merge_file_id := db.activeFile.FileID

	// get all data files need to be merged
	var merge_files []*data.DataFile
	for _, data_file := range db.olderFiles {
		merge_files = append(merge_files, data_file)
	}
	db.mtx.Unlock()

	// order merge files by file id
	sort.Slice(merge_files, func(i, j int) bool {
		return merge_files[i].FileID < merge_files[j].FileID
	})

	merge_path := db.getMergePath()
	if _, err := os.Stat(merge_path); err == nil {
		if err := os.RemoveAll(merge_path); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(merge_path, os.ModePerm); err != nil {
		return err
	}

	// open a nwe temporary db
	merge_options := db.options
	merge_options.DirPath = merge_path
	merge_options.SyncWrites = false
	merge_db, err := OpenDB(merge_options)
	if err != nil {
		return err
	}

	hint_file, err := data.OpenHintFile(merge_path)
	if err != nil {
		return err
	}
	defer hint_file.Close()

	for _, data_file := range merge_files {
		var offset int64 = 0
		for {
			log_record, size, err := data_file.ReadLogRecord(offset)
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			// parse log record key
			real_key, _ := parseLogRecordKey(log_record.Key)
			log_record_pos := db.index.Get(real_key)
			if log_record_pos != nil &&
				log_record_pos.Fid == data_file.FileID &&
				log_record_pos.Offset == offset {

				// clear transaction id
				log_record.Key = logRecordKeyWithSeq(real_key, nonTransactionSeqNo)
				pos, err := merge_db.appendLogRecord(log_record)
				if err != nil {
					return err
				}

				// write current index to hint file
				if err := hint_file.WriteHintRecord(log_record.Key, pos); err != nil {
					return err
				}
			}
			offset += int64(size)
		}
	}

	// sync hint and merge files
	if err := hint_file.Sync(); err != nil {
		return err
	}
	if err := merge_db.Sync(); err != nil {
		return err
	}

	// write merge finished file to indicate that merge is finished
	merge_finished_file, err := data.OpenMergeFinishedFile(merge_path)
	if err != nil {
		return err
	}
	merge_finished_record := &data.LogRecord{
		Key:   []byte(mergeFinishedKey),
		Value: []byte(strconv.Itoa(int(non_merge_file_id))),
	}

	enc_record, _ := data.EncodeLogRecord(merge_finished_record)
	_, err = merge_finished_file.Write(enc_record)
	if err != nil {
		return err
	}
	if err := merge_finished_file.Sync(); err != nil {
		return err
	}
	merge_finished_file.Close()

	return nil
}

func (db *DB) getMergePath() string {
	dir := path.Dir(path.Clean(db.options.DirPath))
	base := path.Base(db.options.DirPath)
	return filepath.Join(dir, base+mergeDirName)
}

// loadMergeFiles loads the merge files from the merge directory.
func (db *DB) loadMergeFiles() error {
	merge_path := db.getMergePath()
	if _, err := os.Stat(merge_path); os.IsNotExist(err) {
		return nil
	}
	defer func() {
		_ = os.RemoveAll(merge_path)
	}()

	dir_entries, err := os.ReadDir(merge_path)
	if err != nil {
		return err
	}

	var merge_finished bool
	var merge_file_names []string
	for _, dir_entry := range dir_entries {
		if dir_entry.Name() == data.MergeFinishedFileName {
			merge_finished = true
		}
		if dir_entry.Name() == data.SeqNoFileName {
			continue
		}
		merge_file_names = append(merge_file_names, dir_entry.Name())
	}

	// return nil if merge is not finished
	if !merge_finished {
		return nil
	}

	non_merge_file_id, err := db.getNonMergeFileID(merge_path)
	if err != nil {
		return nil
	}

	// delete old data files
	var file_id uint32 = 0
	for ; file_id < non_merge_file_id; file_id++ {
		file_name := data.GetDataFileName(db.options.DirPath, file_id)
		if _, err := os.Stat(file_name); err == nil {
			if err := os.Remove(file_name); err != nil {
				return err
			}
		}
	}

	// move new data files to data dir
	for _, merge_file_name := range merge_file_names {
		src_path := filepath.Join(merge_path, merge_file_name)
		dst_path := filepath.Join(db.options.DirPath, merge_file_name)
		if err := os.Rename(src_path, dst_path); err != nil {
			return err
		}
	}

	return nil
}

// getNonMergeFileID returns the file id of the non-merge file
func (db *DB) getNonMergeFileID(dirPath string) (uint32, error) {
	merge_finished_file, err := data.OpenMergeFinishedFile(dirPath)
	if err != nil {
		return 0, err
	}
	defer merge_finished_file.Close()

	var merge_finished_record *data.LogRecord
	merge_finished_record, _, err = merge_finished_file.ReadLogRecord(0)
	if err != nil {
		return 0, err
	}

	non_merge_file_id, err := strconv.Atoi(string(merge_finished_record.Value))
	if err != nil {
		return 0, err
	}
	return uint32(non_merge_file_id), nil
}

// load index from hint files
func (db *DB) loadIndexFromHintFiles() error {
	if len(db.fileIDs) == 0 {
		return nil
	}

	hint_file_name := filepath.Join(db.getMergePath(), data.HintFileName)
	if _, err := os.Stat(hint_file_name); os.IsNotExist(err) {
		return nil
	}

	hint_file, err := data.OpenHintFile(db.getMergePath())
	if err != nil {
		return err
	}
	defer hint_file.Close()

	var offset int64 = 0
	for {
		log_record, size, err := hint_file.ReadLogRecord(offset)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		// parse log record key

		pos, _ := data.DecodeLogRecordPos(log_record.Value)
		db.index.Put(log_record.Key, pos)

		// update read offset
		offset += int64(size)
	}

	return nil
}

// loadIndexFromDataFiles loads the index from the data files.
func (db *DB) loadIndexFromDataFiles() error {
	if len(db.fileIDs) == 0 {
		return nil
	}

	has_merge := false
	non_merge_file_id := uint32(0)
	merge_finished_file := filepath.Join(db.options.DirPath, data.MergeFinishedFileName)
	if _, err := os.Stat(merge_finished_file); err == nil {
		fid, err := db.getNonMergeFileID(db.options.DirPath)
		if err != nil {
			return err
		}
		has_merge = true
		non_merge_file_id = fid
	}

	updateIndex := func(key []byte, typ data.LogRecordType, pos *data.LogRecordPos) {
		var ok bool
		if typ == data.LogRecordNormal {
			ok = db.index.Put(key, pos)
		} else {
			ok = db.index.Delete(key)
		}
		if !ok {
			panic("index update failed")
		}
	}

	var transactionRecords = make(map[uint64][]*data.TransactionRecord)
	var data_file *data.DataFile
	var current_seq_no uint64 = nonTransactionSeqNo
	// iterate over the data files and load the index from each file
	for i, fid := range db.fileIDs {

		if has_merge && fid < int(non_merge_file_id) {
			continue
		}

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

			real_key, seq_no := parseLogRecordKey(log_record.Key)
			if seq_no == nonTransactionSeqNo { // non transaction update index
				updateIndex(real_key, log_record.Type, log_record_pos)
			} else { // transaction update index
				if log_record.Type == data.LogRecordFinished {
					for _, trx_record := range transactionRecords[seq_no] {
						updateIndex(trx_record.Record.Key, trx_record.Record.Type, trx_record.Pos)
					}
					delete(transactionRecords, seq_no)
				} else {
					log_record.Key = real_key
					transactionRecords[seq_no] = append(transactionRecords[seq_no], &data.TransactionRecord{
						Record: log_record,
						Pos:    log_record_pos,
					})
				}
			}

			// update the current sequence number
			if seq_no > current_seq_no {
				current_seq_no = seq_no
			}
			// update the current offset of active file
			offset += int64(size)
		}
		if len(db.fileIDs)-1 == i {
			db.activeFile.WriteOffset = offset
		}
	}

	db.seqNo = current_seq_no

	return nil
}
