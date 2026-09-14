package data

import (
	"hash/crc32"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeLogRecord(t *testing.T) {
	log_record := &LogRecord{
		Key:   []byte("name"),
		Value: []byte("Hello, World!"),
		Type:  LogRecordNormal,
	}
	encRecord, size := EncodeLogRecord(log_record)
	require.NotNil(t, encRecord)
	require.Greater(t, size, int64(5))

	// test delete record
	log_record = &LogRecord{
		Key:   []byte("name"),
		Value: []byte{},
		Type:  LogRecordDeleted,
	}
	encRecord, size = EncodeLogRecord(log_record)
	require.NotNil(t, encRecord)
	require.Greater(t, size, int64(5))

	decRecord, size := DecodeLogRecord(encRecord)
	require.NotNil(t, decRecord)
	require.NotEqual(t, size, 0)
	require.EqualValues(t, log_record, decRecord)
}

func TestDecodeLogRecord(t *testing.T) {
	log_record := &LogRecord{
		Key:   []byte("Hello, World!"),
		Value: []byte("Hello, World!"),
		Type:  LogRecordNormal,
	}
	encRecord, size := EncodeLogRecord(log_record)
	require.NotNil(t, encRecord)
	require.Greater(t, size, int64(5))

	decRecord, size := DecodeLogRecord(encRecord)
	require.NotNil(t, decRecord)
	require.NotEqual(t, size, 0)
	require.EqualValues(t, log_record, decRecord)
}

func TestGetLogRecordCRC(t *testing.T) {
	log_record := &LogRecord{
		Key:   []byte("name"),
		Value: []byte("Hello, World!"),
		Type:  LogRecordNormal,
	}
	encRecord, size := EncodeLogRecord(log_record)
	require.NotNil(t, encRecord)
	require.Greater(t, size, int64(5))

	header, header_size := decodeLogRecordHeader(encRecord)
	require.NotNil(t, header)
	require.EqualValues(t, header.crc, crc32.ChecksumIEEE(encRecord[4:]))
	require.EqualValues(t, header.KeySize, uint32(len(log_record.Key)))
	require.EqualValues(t, header.ValueSize, uint32(len(log_record.Value)))

	crc_prev := crc32.ChecksumIEEE(encRecord[4:])
	crc := calculateLogRecordCRC(log_record, encRecord[4:header_size])

	require.EqualValues(t, crc, crc_prev)
}
