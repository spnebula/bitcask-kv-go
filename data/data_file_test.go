package data

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func RemoveFile(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return err
	}

	return nil
}

func TestOpenDataFile(t *testing.T) {
	t.Log("file path:", os.TempDir())
	df, err := OpenDataFile(os.TempDir(), 0)
	if err != nil {
		t.Fatal(err)
	}
	require.NotNil(t, df)
	require.Nil(t, err)
	defer func() {
		RemoveFile(filepath.Join(os.TempDir(), fmt.Sprintf("%d.data", 0)))
	}()
	defer df.Close()

	df2, err := OpenDataFile(os.TempDir(), 111)
	if err != nil {
		t.Fatal(err)
	}
	require.NotNil(t, df2)
	df3, err := OpenDataFile(os.TempDir(), 111)
	if err != nil {
		t.Fatal(err)
	}
	require.NotNil(t, df3)
	df.Close()
	df2.Close()
}

func TestDataFileWrite(t *testing.T) {
	t.Log("file path:", os.TempDir())
	df, err := OpenDataFile(os.TempDir(), 0)
	if err != nil {
		t.Fatal(err)
	}
	require.NotNil(t, df)
	require.Nil(t, err)
	defer func() {
		RemoveFile(filepath.Join(os.TempDir(), fmt.Sprintf("%d.data", 0)))
	}()
	defer df.Close()

	data := []byte("Hello, World!")
	n, err := df.Write(data)
	require.Nil(t, err)
	require.Equal(t, len(data), n)

	data = []byte("\nHello, World!2")
	n, err = df.Write(data)
	require.Nil(t, err)
	require.Equal(t, len(data), n)
	err = df.Close()
	require.Nil(t, err)
}

func TestDataFileReadLogRecord(t *testing.T) {
	t.Log("file path:", os.TempDir())
	log_record := &LogRecord{
		Key:   []byte("name"),
		Value: []byte("Hello, World!"),
		Type:  LogRecordNormal,
	}
	encRecord, size := EncodeLogRecord(log_record)
	require.NotNil(t, encRecord)
	require.Greater(t, size, int64(5))

	decRecord, dec_size := DecodeLogRecord(encRecord)
	require.NotNil(t, decRecord)
	require.Equal(t, size, dec_size)

	// write log record to data file
	df, err := OpenDataFile(os.TempDir(), 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		RemoveFile(filepath.Join(os.TempDir(), fmt.Sprintf("%d.data", 0)))
	}()
	defer df.Close()

	require.NotNil(t, df)
	require.Nil(t, err)

	n, err := df.Write(encRecord)
	require.Nil(t, err)
	require.Equal(t, size, int64(n))

	// read log record from data file
	offset := int64(0)
	log_record, size, err = df.ReadLogRecord(offset)
	require.Nil(t, err)
	require.NotNil(t, log_record)
	require.Equal(t, size, int64(len(encRecord)))
	require.Equal(t, log_record.Key, log_record.Key)
	require.Equal(t, log_record.Value, log_record.Value)
	require.Equal(t, log_record.Type, log_record.Type)

	log_record2 := &LogRecord{
		Key:   []byte("name2"),
		Value: []byte("Hello, World2!"),
		Type:  LogRecordNormal,
	}
	encRecord2, size2 := EncodeLogRecord(log_record2)
	require.NotNil(t, encRecord2)
	require.Greater(t, size2, int64(5))

	n, err = df.Write(encRecord2)
	require.Nil(t, err)
	require.Equal(t, size2, int64(n))

	// read log record from data file
	offset = int64(size)
	log_record, size, err = df.ReadLogRecord(offset)
	require.Nil(t, err)
	require.NotNil(t, log_record)
	require.Equal(t, size, int64(len(encRecord2)))
	require.Equal(t, log_record.Key, log_record2.Key)
	require.Equal(t, log_record.Value, log_record2.Value)
	require.Equal(t, log_record.Type, log_record2.Type)

}
