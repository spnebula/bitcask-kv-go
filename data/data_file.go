package data

import (
	"github.com/spnebula/bitcask-kv-go/fio"
)

// DataFile represents a data file in the Bitcask key-value store.
type DataFile struct {
	FileID      uint32        // FileID is the unique identifier for the data file.
	WriteOffset int64         // WriteOffset is the current write offset in the file.
	IoManager   fio.IOManager // IoManager is the I/O manager for the data file.
}

// OpenDataFile opens a data file with the given file ID.
func OpenDataFile(dirPath string, fileID uint32) (*DataFile, error) {

	dataFile := &DataFile{
		FileID: fileID,
	}

	return dataFile, nil
}

func (df *DataFile) ReadLogRecord(offset int64) (log_record *LogRecord, size int, err error) {
	return nil, 0, nil
}

func (df *DataFile) Write([]byte) error {
	return nil
}

func (df *DataFile) Sync() error {

	return nil
}

func (df *DataFile) Close() error {
	return df.IoManager.Close()
}
