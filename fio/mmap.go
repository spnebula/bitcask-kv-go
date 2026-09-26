package fio

import (
	"os"

	"golang.org/x/exp/mmap" // only for reading operation
)

type MMap struct {
	readerAt *mmap.ReaderAt
}

func NewMMapIOManager(fileName string) (*MMap, error) {
	_, err := os.OpenFile(fileName, os.O_CREATE, DataFilePermission)
	if err != nil {
		return nil, err
	}
	readerAt, err := mmap.Open(fileName)
	if err != nil {
		return nil, err
	}
	return &MMap{readerAt: readerAt}, nil
}

func (m *MMap) Read(p0 []byte, p1 int64) (int, error) {
	return m.readerAt.ReadAt(p0, p1)
}

func (m *MMap) Write(p0 []byte) (int, error) {
	panic("TODO: Implement")
}

func (m *MMap) Sync() error {
	panic("TODO: Implement")
}

func (m *MMap) Close() error {
	return m.readerAt.Close()
}

func (m *MMap) Size() (int64, error) {
	return int64(m.readerAt.Len()), nil
}
