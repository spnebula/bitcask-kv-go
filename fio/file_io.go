package fio

import "os"

type FileIO struct {
	fd   *os.File // fd is the file descriptor for the underlying file.
	size int64
}

func NewFileIOManager(fileName string) (*FileIO, error) {
	fd, err := os.OpenFile(fileName,
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		DataFilePermission)
	if err != nil {
		return nil, err
	}

	file_io := &FileIO{fd: fd}
	size, err := file_io.FileSize()
	if err != nil {
		return nil, err
	}
	file_io.size = size

	return file_io, nil
}

func (fio *FileIO) Read(b []byte, p1 int64) (int, error) {
	return fio.fd.ReadAt(b, p1)
}

func (fio *FileIO) Write(b []byte) (int, error) {
	size, err := fio.fd.Write(b)
	if err != nil {
		return 0, err
	}
	fio.size += int64(size)
	return size, nil
}

func (fio *FileIO) Sync() error {
	return fio.fd.Sync()
}

func (fio *FileIO) Close() error {
	return fio.fd.Close()
}

func (fio *FileIO) Size() (int64, error) {
	// stat, err := fio.fd.Stat()
	// if err != nil {
	// 	return 0, err
	// }
	// size := stat.Size()
	return fio.size, nil
	// return size, nil
}

func (fio *FileIO) FileSize() (int64, error) {
	stat, err := fio.fd.Stat()
	if err != nil {
		return 0, err
	}
	size := stat.Size()
	return size, nil
}
