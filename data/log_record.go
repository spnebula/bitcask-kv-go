package data

import (
	"encoding/binary"
	"hash/crc32"
)

type LogRecordType = byte

const (
	LogRecordNormal  LogRecordType = 0 // Normal record
	LogRecordDeleted LogRecordType = 1 // Deleted record
)

// crc tyoe keysize valuesize
// 4 + 1 + 5(variant) + 5(variant) = 15
const (
	MaxLogRecordHeaderSize = 15
)

type LogRecordPos struct {
	Fid    uint32 // File ID
	Offset int64  // Offset within the file
}

type LogRecordHeader struct {
	crc       uint32        // CRC32 of the key and value
	KeySize   uint32        // Size of the key
	ValueSize uint32        // Size of the value
	Type      LogRecordType // Type of the record
}

type LogRecord struct {
	Key   []byte        // Key of the record
	Value []byte        // Value of the record
	Type  LogRecordType // Type of the record (e.g., normal, deleted)
}

// EncodeLogRecord encodes a log record into a byte slice.
// crc type key_sz value_sz key value
// 4   1    变长(5 变长(5   变长 变长
func EncodeLogRecord(log_record *LogRecord) ([]byte, int64) {
	header := make([]byte, MaxLogRecordHeaderSize)

	header[4] = log_record.Type

	var index = 5
	write_size := binary.PutVarint(header[index:], int64(len(log_record.Key)))
	index += write_size

	write_size = binary.PutVarint(header[index:], int64(len(log_record.Value)))
	index += write_size

	var total_size = index + len(log_record.Key) + len(log_record.Value)
	encBytes := make([]byte, total_size)
	copy(encBytes, header)
	copy(encBytes[index:], log_record.Key)
	copy(encBytes[index+len(log_record.Key):], log_record.Value)

	var crc_res = crc32.ChecksumIEEE(encBytes[4:])
	binary.LittleEndian.PutUint32(encBytes[:4], crc_res)

	return encBytes, int64(total_size)
}

func DecodeLogRecord(data []byte) (*LogRecord, int64) {
	header, header_size := decodeLogRecordHeader(data)
	if header == nil {
		return nil, 0
	}

	var record_size = header_size + int64(header.KeySize) + int64(header.ValueSize)

	log_record := &LogRecord{
		Key:   nil,
		Value: nil,
	}
	if header.KeySize > 0 || header.ValueSize > 0 {
		record_bytes := data[header_size:]

		log_record = &LogRecord{
			Key:   record_bytes[:header.KeySize],
			Value: record_bytes[header.KeySize : header.KeySize+header.ValueSize],
			Type:  header.Type,
		}
	}
	// Verify CRC
	crc := calculateLogRecordCRC(log_record, data[crc32.Size:header_size])
	if crc != header.crc {
		return nil, 0
	}
	return log_record, record_size
}

func decodeLogRecordHeader(data []byte) (*LogRecordHeader, int64) {
	if len(data) < 5 {
		return nil, 0
	}

	header := &LogRecordHeader{
		crc:  binary.LittleEndian.Uint32(data[:4]),
		Type: LogRecordType(data[4]),
	}

	var index = 5
	key_sz, n := binary.Varint(data[index:])
	index += n

	value_sz, n := binary.Varint(data[index:])
	index += n

	header.KeySize = uint32(key_sz)
	header.ValueSize = uint32(value_sz)

	return header, int64(index)

}

// getRecordCRC calculates the CRC of the given log record.
func calculateLogRecordCRC(log_record *LogRecord, log_record_buf []byte) uint32 {
	if len(log_record_buf) == 0 {
		return 0
	}

	crc := crc32.ChecksumIEEE(log_record_buf[:])
	crc = crc32.Update(crc, crc32.IEEETable, log_record.Key)
	crc = crc32.Update(crc, crc32.IEEETable, log_record.Value)
	return crc
}

// log_record.go
func (p *LogRecordPos) Compare(other *LogRecordPos) int {
	if p.Fid != other.Fid {
		if p.Fid < other.Fid {
			return -1
		}
		return 1
	}
	switch {
	case p.Offset < other.Offset:
		return -1
	case p.Offset > other.Offset:
		return 1
	default:
		return 0
	}
}
