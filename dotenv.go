// Package dotenv is a high level wrapper around [godotenv]. It allows to load
// one or multiple .env file(s) according to [original rules]. It searches for
// .env file(s) in current and parent dirs, until it find at least one of them.
//
// [godotenv]: https://github.com/joho/godotenv
// [original rules]: https://github.com/bkeepers/dotenv#what-other-env-files-can-i-use
package dotenv

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// New creates and returns an instance of .env loader [Loader]. By default it
// searches for .env file(s) until it reaches of the root or any parent dir
// where go.mod file exists.
func New() Loader {
	return Loader{
		rootDir:   string(filepath.Separator),
		rootFiles: []string{"go.mod"},
	}
}

// Loader is a loader of .env files. Don't create it directly, use [New]
// instead.
type Loader struct {
	// envSuffix is a suffix of .env files for current environment
	envSuffix string

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
}

// WithDepth configures [Load] don't go up deeper and stop searching for .env
// files at n level. Current dir has n == 1, first parent dir has n == 2 and so
// on.
func (self *Loader) WithDepth(n int) *Loader {
	self.lookupDepth = n
	return self
}

// WithEnvVarName reads name of current environment from s environment variable
// and configures [Load] for searching and loading of .env.CURENV* files. For
// instance with s == "production" it'll search also for ".env.production.local"
// and ".env.production". With s == "test" - ".env.test.local" and
// ".env.test". And so on.
func (self *Loader) WithEnvVarName(s string) *Loader {
	if v, ok := os.LookupEnv(s); ok {
		self.envSuffix = v
	}
	return self
}

// WithEnvSuffix directly sets name of current environment to s. See
// [WithEnvVarName] above for details.
func (self *Loader) WithEnvSuffix(s string) *Loader {
	self.envSuffix = s
	return self
}

// WithRootDir configures [Load] to stop at path dir and don't go up.
func (self *Loader) WithRootDir(path string) *Loader {
	if absPath, err := filepath.Abs(path); err == nil {
		self.rootDir = absPath
	}
	return self
}

// WithRootFiles configures [Load] to stop at current dir or any parent dir,
// which contains any of file with name from fnames list.
func (self *Loader) WithRootFiles(fnames ...string) *Loader {
	self.rootFiles = fnames
	return self
}

// WithRootCallback configures [Load] to call fn function for every dir it
// visits. It passes absolute path of current dir as path param and expects two
// return values:
//
//  1. true means stop at this dir
//  2. any error
//
// [FileExistsInDir] may be useful in here.
func (self *Loader) WithRootCallback(fn func(path string) (bool, error),
) *Loader {
	self.rootCb = fn
	return self
}

// Load loads .env files in current dir if any of them exists. If nothing was
// found it tries parent dir and parent of parent dir and so on, until it'll
// find any of .env files or will reach any of configured condition:
//
//  1. Visited dir is at level configured by [WithDepth], where level 1 is
//     current dir, level 2 is parent dir and so on.
//  2. Visited dir is a root dir configured by [WithRootDir].
//  3. Visited dir has any of file with names configured by [WithRootFiles].
//  4. A callback function was configured by [WithRootCallback] and that
//     function returned true for visited dir.
//
// If name of environment wasn't configured by [WithEnvVarName] or
// [WithEnvSuffix], Load is looking for:
//
//  1. env.local
//  2. .env
//
// If name of environment was configured, "production" for instance, it's
// looking for:
//
//  1. .env.production.local
//  2. .env.local
//  3. .env.production
//  4. .env
//
// Load uses [godotenv.Load] and according to how it works any already defined
// env variable can't be redefined by next .env file and has priority. So if
// variable "A" defined in .env.local file, it can't be redefined by variable
// "A" from .env file. Or if env variable "A" somehow defined before calling
// Load, it keeps its value and can't be redefined by .env files.
func (self *Loader) Load() error {
	if envs, err := self.lookupEnvFiles(); err != nil {
		return err
	} else if len(envs) == 0 {
		return nil
	} else if err := godotenv.Load(envs...); err != nil {
		return fmt.Errorf("can't load %v: %w", envs, err)
	}

	return nil
}

// FileExistsInDir checks if file named fname exists in dir named dirName and
// returns true, if it exists, or false.
//
// May be useful in a callback, configured by [WithRootCallback].
func (self *Loader) FileExistsInDir(dirName, fname string) (bool, error) {
	if dirName != "" {
		fname = filepath.Join(dirName, fname)
	}

	if _, err := os.Stat(fname); err == nil {
		return true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("can't stat file '%s': %w", fname, err)
	}

	return false, nil
}

// lookupEnvFiles is searching for .env files, starting from current dir, and
// returns list of found files or nil if nothing found.
//
// If .env files were found in one of parent dirs, their names are absolute
// paths. If they are in current dir, returned list will contain just their
// names.
func (self *Loader) lookupEnvFiles() ([]string, error) {
	envs := self.envFiles()

	found, envDir, err := self.lookupEnvDir(envs)
	if err != nil {
		return nil, fmt.Errorf("got error looking for %v: %w", envs, err)
	} else if !found {
		return nil, nil
	}

	// foundEnvs will overwrite envs and it's safe, because we append into
	// foundEnvs the same number of items or less.
	foundEnvs := envs[:0]
	for _, envFile := range envs {
		if exists, err := self.FileExistsInDir(envDir, envFile); err != nil {
			return nil, err
		} else if exists {
			if envDir != "" {
				envFile = filepath.Join(envDir, envFile)
			}
			foundEnvs = append(foundEnvs, envFile)
		}
	}

	if len(foundEnvs) > 0 {
		return foundEnvs, nil
	}

	return nil, nil // nothing found
}

// envFile returns list of .env files for searching, according to configured
// name of environment. See [Load] for details.
func (self *Loader) envFiles() []string {
	envName := self.envSuffix
	if envName == "" {
		return []string{".env.local", ".env"}
	}

	return []string{
		".env." + envName + ".local", ".env.local",
		".env." + envName, ".env",
	}
}

// lookupEnvDir is searching for a dir, which contains any of files with names
// from envFiles list. It returns:
//
//  1. true, if found any of files, or false.
//  2. dir name if any file was found
//  3. any error
//
// Returned dir name is absolute path or empty string, which means current dir.
//
// It starts searching at current dir, next tries parent dir, parent of parent
// dir and so on, until it reaches configured root.
func (self *Loader) lookupEnvDir(envFiles []string) (bool, string, error) {
	curDir := ""
	depth := 0

	for {
		for _, envFile := range envFiles {
			if exists, err := self.FileExistsInDir(curDir, envFile); err != nil {
				return false, "", err
			} else if exists {
				return exists, curDir, nil
			}
		}

		if depth = self.checkLookupDepth(depth); depth < 0 {
			break
		}

		if newDir, err := self.nextParentDir(curDir); err != nil {
			return false, "", fmt.Errorf("next parent dir of %v: %w", curDir, err)
		} else if newDir == "" {
			break
		} else {
			curDir = newDir
		}
	}

	return false, "", nil
}

// checkLookupDepth compares current dir level curDir with configured one and
// returns -1, if reached configured limit, or next level. It expects curDir >=
// 0.
//
// It understands the limit is configured if lookupDepth > 0.
func (self *Loader) checkLookupDepth(curDepth int) int {
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
func (self *Loader) nextParentDir(curDir string) (string, error) {
	if curDir == "" {
		if dir, err := os.Getwd(); err != nil {
			return "", fmt.Errorf("can't get current dir: %w", err)
		} else {
			curDir = dir
		}
	}

	if stopHere, err := self.stopByRootCb(curDir); err != nil {
		return "", err
	} else if stopHere {
		return "", nil
	} else if curDir == self.rootDir {
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

// stopByRootCb calls a function, configured by [WithRootCallback], with
// absolute path, and returns its return values. true means stop at this path
// and false means continue to parent dir.
func (self *Loader) stopByRootCb(path string) (bool, error) {
	if self.rootCb != nil {
		if stopHere, err := self.rootCb(path); err != nil {
			return false, fmt.Errorf("check dir %v using root callback: %w", path, err)
		} else {
			return stopHere, nil
		}
	}

	return false, nil
}
