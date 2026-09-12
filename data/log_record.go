package data

import "hash/crc32"

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
func EncodeLogRecord(log_record *LogRecord) ([]byte, int64) {
	return nil, 0
}

func DecodeLogRecord(data []byte) (*LogRecord, int64) {
	return nil, 0
}

func decodeLogRecordHeader(data []byte) (*LogRecordHeader, int64) {
	header := &LogRecordHeader{}
	header.crc = uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
	// header.KeySize = data[4]
	// header.ValueSize = data[5]
	// header.Type = LogRecordType(data[6])
	return header, 7

}

func getLogRecordHeaderCRC(log_record *LogRecord, log_record_buf []byte) uint32 {
	crc := crc32.NewIEEE()
	crc.Write(log_record_buf)
	crc.Write([]byte{byte(log_record.Type)})
	crc.Write(log_record.Key)
	crc.Write(log_record.Value)
	return crc.Sum32()
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
