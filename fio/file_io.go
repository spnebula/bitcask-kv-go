package fio

import "os"

type FileIO struct {
	fd *os.File // fd is the file descriptor for the underlying file.

}

func NewFileIOManager(fileName string) (*FileIO, error) {
	fd, err := os.OpenFile(fileName,
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		DataFilePerm)
	if err != nil {
		return nil, err
	}

	return &FileIO{fd: fd}, nil
}

func (fio *FileIO) Read(b []byte, p1 int64) (int, error) {
	return fio.fd.ReadAt(b, p1)
}

func (fio *FileIO) Write(b []byte) (int, error) {
	// TODO: We can use WriteAt here to avoid the lock in the underlying file descriptor. But we need to maintain the offset ourselves.
	return fio.fd.Write(b)
}

func (fio *FileIO) Sync() error {
	return fio.fd.Sync()
}

func (fio *FileIO) Close() error {
	return fio.fd.Close()
}
