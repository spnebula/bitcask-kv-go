package utils

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func DirSize(dirPath string) (int64, error) {
	var size int64
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func AvailableDiskSpace() (uint64, error) {
	dir, err := os.Getwd()
	if err != nil {
		return 0, err
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return 0, err
	}
	return stat.Bavail * uint64(stat.Bsize), nil
}

func CopyDir(src, dest string, exclude []string) error {
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		if err := os.MkdirAll(dest, os.ModePerm); err != nil {
			return err
		}
	}

	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		file_name := strings.Replace(path, src, "", 1)
		if file_name == "" {
			return nil
		}

		// exclude 中任意一条 glob 命中即排除（"或" 语义）
		for _, e := range exclude {
			matched, err := filepath.Match(e, info.Name())
			if err != nil {
				return err
			}
			if matched {
				if info.IsDir() {
					// 目录被排除时需连同子树一起跳过，否则子文件会写到未创建的目录
					return filepath.SkipDir
				}
				return nil
			}
		}

		if info.IsDir() {
			return os.MkdirAll(filepath.Join(dest, file_name), info.Mode())
		}

		data, err := os.ReadFile(filepath.Join(src, file_name))
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dest, file_name), data, info.Mode())
	})
}
