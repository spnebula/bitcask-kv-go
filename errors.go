package bitcaskkvgo

import "errors"

var (
	ErrKeyEmpty          = errors.New("key is empty")
	ErrIndexUpdateFailed = errors.New("index update failed")
	ErrKeyNotFound       = errors.New("not found by key")
	ErrDataFileNotFound  = errors.New("data file not found")
)
