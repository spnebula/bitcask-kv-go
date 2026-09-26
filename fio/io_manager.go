package fio

const (
	DataFilePermission = 0644
)

type FileIOType = byte

const (
	// StandardFIO 标准文件 IO
	StandardFIO FileIOType = iota

	// MemoryMap 内存文件映射
	MemoryMap
)

// IOManager is an interface for reading and writing data.
type IOManager interface {
	Read([]byte, int64) (int, error) // Read reads data from the file into the provided byte slice starting at the given offset.
	Write([]byte) (int, error)       // Write writes data from the provided byte slice to the file and returns the number of bytes written.
	Sync() error
	Close() error
	Size() (int64, error)
}

func NewIOManager(fileName string, fileIOType FileIOType) (IOManager, error) {
	switch fileIOType {
	case StandardFIO:
		return NewFileIOManager(fileName)
	case MemoryMap:
		return NewMMapIOManager(fileName)
	default:
		// return nil, ErrInvalidFileIOType
		panic("invalid file io type")
	}
}
