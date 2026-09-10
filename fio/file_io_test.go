package fio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func RemoveFile(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return err
	}

	return nil
}

func TestNewFileIOManager(t *testing.T) {
	fio, err := NewFileIOManager(filepath.Join("/tmp", "data"))
	assert.Nil(t, err)
	assert.NotNil(t, fio)
	defer RemoveFile(filepath.Join("/tmp", "data"))

	err = fio.Close()
	assert.Nil(t, err)
}

func TestFileIO_ReadWrite(t *testing.T) {
	fio, err := NewFileIOManager(filepath.Join("/tmp", "data"))
	assert.Nil(t, err)
	assert.NotNil(t, fio)
	defer RemoveFile(filepath.Join("/tmp", "data"))

	data := []byte("Hello, World!")
	n, err := fio.Write(data)
	assert.Nil(t, err)
	assert.Equal(t, len(data), n)

	readBuffer := make([]byte, len(data))
	n, err = fio.Read(readBuffer, 0)
	assert.Nil(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, readBuffer)
	t.Logf("Read %d bytes, content: %s, with error: %v", n, string(readBuffer), err)
	// fmt.Print(n, err)

	err = fio.Close()
	assert.Nil(t, err)
}

func TestFileIO_Sync(t *testing.T) {
	fio, err := NewFileIOManager(filepath.Join("/tmp", "data"))
	assert.Nil(t, err)
	assert.NotNil(t, fio)
	defer RemoveFile(filepath.Join("/tmp", "data"))

	data := []byte("Hello, World!")
	n, err := fio.Write(data)
	assert.Nil(t, err)
	assert.Equal(t, len(data), n)

	err = fio.Sync()
	assert.Nil(t, err)

	err = fio.Close()
	assert.Nil(t, err)
}
