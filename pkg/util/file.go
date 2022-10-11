package util

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	readWriteMode     = 0o666 // -rw-rw-rw- or drw-rw-rw-
	dirPermissionBits = 0o755 // -rwxr-xr-x or drwxr-xr-x
	logDestKey        = "dest"
)

var log = ctrl.Log.WithName("util")

func CreateDirectory(dirPath string) error {
	log.Info("Creating directory", "path", dirPath)

	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err = os.MkdirAll(dirPath, dirPermissionBits); err != nil {
			return fmt.Errorf("failed to make directory: %w", err)
		}
	}

	log.Info("Directory has been created", "path", dirPath)

	return nil
}

func CopyFiles(src, dest string) error {
	log.Info("Start copying files", "src", src, logDestKey, dest)

	files, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read a content of directory %q: %w", src, err)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		fp := path.Join(src, f.Name())

		var input []byte

		input, err = os.ReadFile(fp)
		if err != nil {
			return fmt.Errorf("failed to read a file %q: %w", fp, err)
		}

		destFp := path.Join(dest, f.Name())

		err = os.WriteFile(destFp, input, dirPermissionBits)
		if err != nil {
			return fmt.Errorf("failed to write to file %q: %w", destFp, err)
		}
	}

	log.Info("Files have been copied", logDestKey, dest)

	return nil
}

func CopyFile(src, dest string) error {
	log.Info("Start copying file", "src", src, logDestKey, dest)

	input, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read a file %q: %w", src, err)
	}

	err = os.WriteFile(dest, input, dirPermissionBits)
	if err != nil {
		return fmt.Errorf("failed to write to file %q: %w", dest, err)
	}

	log.Info("File has been copied", logDestKey, dest)

	return nil
}

func DoesDirectoryExist(dirPath string) bool {
	if _, err := os.Stat(dirPath); err != nil {
		if os.IsNotExist(err) {
			return false
		}

		log.Error(err, "unable to check directory")

		return false
	}

	return true
}

func RemoveDirectory(dirPath string) error {
	if err := os.RemoveAll(dirPath); err != nil {
		return errors.Wrapf(err, "couldn't remove directory %q", dirPath)
	}

	log.Info("directory has been cleaned", "directory", dirPath)

	return nil
}

func IsDirectoryEmpty(dirPath string) bool {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		log.Error(err, "unable to check directory")

		return false
	}

	return len(files) == 0
}

func ReplaceStringInFile(file, oldLine, newLine string) error {
	input, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read a file %q: %w", file, err)
	}

	output := bytes.ReplaceAll(input, []byte(oldLine), []byte(newLine))

	err = os.WriteFile(file, output, readWriteMode)
	if err != nil {
		return fmt.Errorf("failed to write to file %q: %w", file, err)
	}

	return nil
}

func GetListFilesInDirectory(src string) ([]fs.DirEntry, error) {
	files, err := os.ReadDir(src)
	if err != nil {
		return nil, fmt.Errorf("failed to read a content of directory %q: %w", src, err)
	}

	return files, nil
}
