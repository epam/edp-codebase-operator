package util

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("util")

func CreateDirectory(path string) error {
	log.Info("Creating directory for oc templates", "path", path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}
	log.Info("Directory has been created", "path", path)
	return nil
}

func CopyFiles(src, dest string) error {
	log.Info("Start copying files", "src", src, "dest", dest)

	files, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, f := range files {
		input, err := ioutil.ReadFile(src + "/" + f.Name())
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(fmt.Sprintf("%s/%s", dest, f.Name()), input, 0755)
		if err != nil {
			return err
		}
	}

	log.Info("Files have been copied", "dest", dest)

	return nil
}

func CopyFile(src, dest string) error {
	log.Info("Start copying file", "src", src, "dest", dest)

	input, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(dest, input, 0755); err != nil {
		return err
	}
	log.Info("File has been copied", "dest", dest)
	return nil
}

func DoesDirectoryExist(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
		log.Error(err, "unable to check directory")
		return false
	}
	return true
}

func RemoveDirectory(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return errors.Wrapf(err, "couldn't remove directory %v", path)
	}
	log.Info("directory has been cleaned", "directory", path)
	return nil
}

func IsDirectoryEmpty(path string) bool {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Error(err, "unable to check directory")
		return false
	}
	return len(files) == 0
}
