package data

type LogRecordType = byte

const (
	LogRecordNormal  LogRecordType = 0 // Normal record
	LogRecordDeleted LogRecordType = 1 // Deleted record
)

type LogRecordPos struct {
	Fid    uint32 // File ID
	Offset int64  // Offset within the file
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
