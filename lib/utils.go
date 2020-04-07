package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/deckarep/golang-set"
)

// IsDirOrInvalidFile whether the file is dir or os.Stat failed
func IsDirOrInvalidFile(file, srcBaseDir string) bool {
	if file == srcBaseDir {
		return true
	}
	info, err := os.Stat(file)
	if err != nil || info == nil {
		fmt.Printf("%s exists? skipped\n", file)
		return true
	}
	return false
}

// ToStringArray mapset.Set -> []string
func ToStringArray(s mapset.Set) (result []string) {
	for _, v := range s.ToSlice() {
		if str, ok := v.(string); ok {
			result = append(result, str)
		}
	}
	return
}

// DoneWaitGroups wait groups n times
func DoneWaitGroups(wg *sync.WaitGroup, n int) {
	for i := 0; i < n; i++ {
		wg.Done()
	}
}

// CopyFile copy file src to dest
func CopyFile(src, dest string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := source.Close(); err != nil {
			fmt.Printf("%s file close failed\n", src)
		}
	}()

	destination, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() {
		if err := destination.Close(); err != nil {
			fmt.Printf("%s file close failed\n", dest)
		}
	}()
	_, err = io.Copy(destination, source);
	return err
}

// RemoveAll under dir
func RemoveAll(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() {
		if err := d.Close(); err != nil {
			fmt.Printf("%s file close failed\n", dir)
		}
	}()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}
