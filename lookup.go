package dotenv

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// NewLookup creates and returns an instance of [Lookup].
func NewLookup() *Lookup { return &Lookup{stat: os.Stat} }

type Lookup struct {
	// lookupDepth defines how many dirs could be checked before stop. It starts
	// at 1 and it means current dir only. 2 and more means check also parent
	// dirs. 0 means not configured.
	lookupDepth int

	// rootCb is a function, which returns should we stop at current dir or go up.
	rootCb func(path string) (bool, error)

	// rootDir is a dir to stop and don't go up
	rootDir string

	// rootFiles contains list of file names for marking root dir. If current or
	// any parent dir has any of file from this list, we'll stop at that dir.
	rootFiles []string

	// stat returns a FileInfo describing the named file, see [os.Stat].
	stat func(name string) (os.FileInfo, error)

	err error
}

// WithDepth configures [Lookup.Lookup] don't go up deeper and stop searching
// for .env files at n level. Current dir has n == 1, first parent dir has n ==
// 2 and so on.
func (self *Lookup) WithDepth(n int) *Lookup {
	self.lookupDepth = n
	return self
}

// WithRootDir configures [Lookup.Lookup] to stop at path dir and don't go up.
func (self *Lookup) WithRootDir(path string) *Lookup {
	if absPath, err := filepath.Abs(path); err != nil {
		self.err = fmt.Errorf("failed absolutize %q: %w", path, err)
	} else {
		self.rootDir = absPath
	}
	return self
}

// WithRootFiles configures [Lookup.Lookup] to stop at current dir or any parent
// dir, which contains any of file (or dir) with name from fnames list.
func (self *Lookup) WithRootFiles(names ...string) *Lookup {
	self.rootFiles = names
	return self
}

// WithRootCallback configures [Lookup.Lookup] to call fn function for every dir
// it visits. It passes absolute path of current dir as path param and expects
// two return values:
//
//  1. true means stop at this dir
//  2. any error
//
// [Loader.FileExistsInDir] may be useful in here.
func (self *Lookup) WithRootCallback(fn func(path string) (bool, error),
) *Lookup {
	self.rootCb = fn
	return self
}

// Lookup is searching for given files, starting from current dir, and returns
// list of found files or nil if nothing found.
//
// If given files were found in one of parent dirs, their names are absolute
// paths. If they are in current dir, returned list will contain just their
// names.
func (self *Lookup) Lookup(files ...string) ([]string, error) {
	if self.err != nil {
		return nil, self.err
	}

	foundAny, dir, err := self.lookupDir(files)
	if err != nil {
		return nil, fmt.Errorf("got error looking for %v: %w", files, err)
	} else if !foundAny {
		return nil, nil
	}

	// foundFiles will overwrite files and it's safe, because we append into
	// foundFiles the same number of items or less.
	foundFiles := files[:0]
	for _, name := range files {
		if exists, err := self.FileExistsInDir(dir, name); err != nil {
			return nil, err
		} else if exists {
			if dir != "" {
				name = filepath.Join(dir, name)
			}
			foundFiles = append(foundFiles, name)
		}
	}

	// At least one file exists, because lookupDir() returned foundAny == true. So
	// here we never return empty slice.
	return foundFiles, nil
}

// lookupDir is searching for a dir, which contains any of files with names from
// files list. It returns:
//
//  1. true, if found any of files, or false.
//  2. dir name if any file was found
//  3. any error
//
// Returned dir name is absolute path or empty string, which means current dir.
//
// It starts searching at current dir, next tries parent dir, parent of parent
// dir and so on, until it reaches configured root.
func (self *Lookup) lookupDir(files []string) (bool, string, error) {
	var curDir string
	var depth int

	for {
		for _, name := range files {
			if exists, err := self.FileExistsInDir(curDir, name); err != nil {
				return false, "", err
			} else if exists {
				return exists, curDir, nil
			}
		}

		if depth = self.checkLookupDepth(depth); depth < 0 {
			break
		}

		newDir, err := self.nextParentDir(curDir)
		if err != nil {
			return false, "", fmt.Errorf("next parent dir of %v: %w", curDir, err)
		} else if newDir == "" {
			break
		}
		curDir = newDir
	}
	return false, "", nil
}

// FileExistsInDir checks if file named fname exists in dir named dirName and
// returns true, if it exists, or false.
//
// May be useful in a callback, configured by [Lookup.WithRootCallback].
func (self *Lookup) FileExistsInDir(dirName, fname string) (bool, error) {
	if dirName != "" {
		fname = filepath.Join(dirName, fname)
	}

	if _, err := self.stat(fname); err == nil {
		return true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("can't stat file '%s': %w", fname, err)
	}
	return false, nil
}

// checkLookupDepth compares current dir level curDir with configured one and
// returns -1, if reached configured limit, or next level. It expects curDir >=
// 0.
//
// It understands the limit is configured if lookupDepth > 0.
func (self *Lookup) checkLookupDepth(curDepth int) int {
	if self.lookupDepth > 0 {
		curDepth++
		if curDepth == self.lookupDepth {
			return -1
		}
	}
	return curDepth
}

// nextParentDir returns parent dir of curDir or empty string, if it configured
// to stop at curDir. It expects curDir is an absolute path or empty string,
// which means current dir.
func (self *Lookup) nextParentDir(curDir string) (string, error) {
	if curDir == "" {
		dir, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("can't get current dir: %w", err)
		}
		curDir = dir
	}

	if stopHere, err := self.stopByRootCb(curDir); err != nil {
		return "", err
	} else if stopHere || curDir == self.rootDir {
		return "", nil
	}

	for _, fname := range self.rootFiles {
		if exists, err := self.FileExistsInDir(curDir, fname); err != nil {
			return "", fmt.Errorf("check existence of file %v in dir %v: %w", fname,
				curDir, err)
		} else if exists {
			return "", nil
		}
	}
	return filepath.Dir(curDir), nil
}

// stopByRootCb calls a function, configured by [Lookup.WithRootCallback], with
// absolute path, and returns its return values. true means stop at this path
// and false means continue to parent dir.
func (self *Lookup) stopByRootCb(path string) (bool, error) {
	if self.rootCb == nil {
		return false, nil
	}
	stopHere, err := self.rootCb(path)
	if err != nil {
		return false, fmt.Errorf("check dir %v using root callback: %w", path, err)
	}
	return stopHere, nil
}
