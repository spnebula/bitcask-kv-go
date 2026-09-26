package bitcaskkvgo

import "errors"

var (
	ErrKeyIsEmpty        = errors.New("key is empty")
	ErrIndexUpdateFailed = errors.New("index update failed")
	ErrKeyNotFound       = errors.New("not found by key")
	ErrDataFileNotFound  = errors.New("data file not found")
	ErrDirPathEmpty      = errors.New("dir path is empty")
	ErrDataFileSizeEmpty = errors.New("data file size is empty")
	ErrDataDirctCorrupt  = errors.New("data directory is corrupt")

	ErrExceedMaxLogRecordSize = errors.New("exceed max log record size")

	ErrMergeIsInProcess          = errors.New("merge is in process")
	ErrDatabaseIsUsing           = errors.New("the database directory is used by another process")
	ErrDataFileMergeRatioInvalid = errors.New("data file merge ratio is invalid")
	ErrMergeRatioUnReachable     = errors.New("merge ratio is unreachable")
	ErrNoEnoughSpaceForMerge     = errors.New("no enough disk space for merge")
)
