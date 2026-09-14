package data

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"path/filepath"

	"github.com/spnebula/bitcask-kv-go/fio"
)

var (
	ErrInvalidLogRecordCRC = errors.New("invalid log record CRC")
)

// DataFile represents a data file in the Bitcask key-value store.
type DataFile struct {
	FileID      uint32        // FileID is the unique identifier for the data file.
	WriteOffset int64         // WriteOffset is the current write offset in the file.
	IoManager   fio.IOManager // IoManager is the I/O manager for the data file.
}

// OpenDataFile opens a data file with the given file ID.
func OpenDataFile(dirPath string, fileID uint32) (*DataFile, error) {

	file_path := filepath.Join(dirPath, fmt.Sprintf("%d.data", fileID))

	ioManager, err := fio.NewIOManager(file_path)
	if err != nil {
		return nil, err
	}
	dataFile := &DataFile{
		FileID:    fileID,
		IoManager: ioManager,
	}

	return dataFile, nil
}

// ReadLogRecord reads a log record from the data file at the given offset.
func (df *DataFile) ReadLogRecord(offset int64) (log_record *LogRecord, size int64, err error) {
	file_size, err := df.IoManager.Size()
	if err != nil {
		return nil, 0, err
	}

	var header_read_size int64 = MaxLogRecordHeaderSize
	if offset+MaxLogRecordHeaderSize > file_size {
		header_read_size = file_size - offset
	}

	data_bytes, err := df.ReadNBytes(offset, header_read_size)
	if err != nil {
		return nil, 0, err
	}

	log_record_header, header_size := decodeLogRecordHeader(data_bytes)
	if log_record_header == nil {
		return nil, 0, io.EOF
	}
	if log_record_header.ValueSize == 0 && log_record_header.KeySize == 0 {
		return nil, 0, io.EOF
	}

	var record_size int64 = int64(log_record_header.KeySize) + int64(log_record_header.ValueSize)
	size = record_size + header_size

	if log_record_header.KeySize > 0 || log_record_header.ValueSize > 0 {
		record_bytes, err := df.ReadNBytes(offset+header_size, record_size)
		if err != nil {
			return nil, 0, err
		}
		log_record = &LogRecord{
			Key:   record_bytes[:log_record_header.KeySize],
			Value: record_bytes[log_record_header.KeySize:],
			Type:  log_record_header.Type,
		}
	}
	// Verify CRC
	crc := calculateLogRecordCRC(log_record, data_bytes[crc32.Size:header_size])
	if crc != log_record_header.crc {
		return nil, 0, ErrInvalidLogRecordCRC
	}

	return log_record, size, nil
}

func (df *DataFile) ReadNBytes(offset int64, n int64) ([]byte, error) {
	b := make([]byte, n)
	_, err := df.IoManager.Read(b, offset)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (df *DataFile) Write(data []byte) (int, error) {
	szie, err := df.IoManager.Write(data)
	if err != nil {
		return 0, err
	}
	df.WriteOffset += int64(szie)
	return szie, err
}

func (df *DataFile) Sync() error {
	return df.IoManager.Sync()
}

func (df *DataFile) Close() error {
	return df.IoManager.Close()
}
