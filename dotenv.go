// Package dotenv is a high level wrapper around [godotenv]. It allows to load
// one or multiple .env file(s) according to [original rules]. It searches for
// .env file(s) in current and parent dirs, until it find at least one of them.
//
// [godotenv]: https://github.com/joho/godotenv
// [original rules]: https://github.com/bkeepers/dotenv#what-other-env-files-can-i-use
//
//go:generate mockery
package dotenv

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Load loads .env files using default [Loader]. See [Loader.Load] for details
// about callbacks.
func Load(callbacks ...func() error) error { return New().Load(callbacks...) }

// New creates and returns an instance of .env loader [Loader]. By default it
// searches for .env file(s) until it reaches of the root or any parent dir
// where go.mod file exists.
//
// Creation time options can be changed by opts.
func New() *Loader {
	return &Loader{
		lookup: NewLookup().
			WithRootDir(string(filepath.Separator)).
			WithRootFiles("go.mod"),
	}
}

// Loader is a loader of .env files. Don't create it directly, use [New]
// instead.
type Loader struct {
	lookup *Lookup

	// envSuffix is a suffix of .env files for current environment
	envSuffix string
}

// WithDepth configures [Loader.Load] don't go up deeper and stop searching for
// .env files at n level. Current dir has n == 1, first parent dir has n == 2
// and so on.
func (self *Loader) WithDepth(n int) *Loader {
	self.lookup.WithDepth(n)
	return self
}

// WithEnvVarName reads name of current environment from s environment variable
// and configures [Loader.Load] for searching and loading of .env.CURENV*
// files. For instance with s == "production" it'll search also for
// ".env.production.local" and ".env.production". With s == "test" -
// ".env.test.local" and ".env.test". And so on.
//
// This example configures env to read environment name from "ENV" environment
// variable:
//
//	env := dotenv.New()
//	env.WithEnvVarName("ENV")
//
// So if "ENV" environment variable contains "test", next call to [Loader.Load]
// will try to load ".env.test*" files. See [Loader.Load] for details.
func (self *Loader) WithEnvVarName(s string) *Loader {
	if v, ok := os.LookupEnv(s); ok {
		self.envSuffix = v
	}
	return self
}

// WithEnvSuffix directly sets name of current environment to s. See
// [Loader.WithEnvVarName] above for details.
func (self *Loader) WithEnvSuffix(s string) *Loader {
	self.envSuffix = s
	return self
}

// WithRootDir configures [Loader.Load] to stop at path dir and don't go up.
func (self *Loader) WithRootDir(path string) *Loader {
	self.lookup.WithRootDir(path)
	return self
}

// WithRootFiles configures [Loader.Load] to stop at current dir or any parent
// dir, which contains any of file (or dir) with name from fnames list.
func (self *Loader) WithRootFiles(fnames ...string) *Loader {
	self.lookup.WithRootFiles(fnames...)
	return self
}

// WithRootCallback configures [Loader.Load] to call fn function for every dir
// it visits. It passes absolute path of current dir as path param and expects
// two return values:
//
//  1. true means stop at this dir
//  2. any error
//
// [Loader.FileExistsInDir] may be useful in here.
func (self *Loader) WithRootCallback(fn func(path string) (bool, error),
) *Loader {
	self.lookup.WithRootCallback(fn)
	return self
}

// FileExistsInDir checks if file named fname exists in dir named dirName and
// returns true, if it exists, or false.
//
// May be useful in a callback, configured by [Loader.WithRootCallback].
func (self *Loader) FileExistsInDir(dirName, fname string) (bool, error) {
	return self.lookup.FileExistsInDir(dirName, fname)
}

// Load loads .env files in current dir if any of them exists. If nothing was
// found it tries parent dir and parent of parent dir and so on, until it'll
// find any of .env files or will reach any of configured condition:
//
//  1. Visited dir is at level configured by [Loader.WithDepth], where level 1
//     is current dir, level 2 is parent dir and so on.
//  2. Visited dir is a root dir configured by [Loader.WithRootDir].
//  3. Visited dir has any of file with names configured by
//     [Loader.WithRootFiles].
//  4. A callback function was configured by [Loader.WithRootCallback] and that
//     function returned true for visited dir.
//
// If name of environment wasn't configured by [Loader.WithEnvVarName] or
// [Loader.WithEnvSuffix], Load is looking for:
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
//
// After succesfull loading of .env file(s) it calls functions from cbs one by
// one. It stops calling callbacks after first error. Here an example of using
// [env] to parse env vars into a struct:
//
//	cfg := struct {
//		SomeOpt string `env:"ENV_VAR1"`
//	}{
//		SomeOpt: "some default value, because we don't have .env file(s)",
//	}
//
//	err := dotenv.New().Load(func() error {
//		return env.Parse(&cfg)
//	})
//	if err != nil {
//		log.Fatalf("error loading .env files: %v", err)
//	}
//
// [env]: https://github.com/caarlos0/env
func (self *Loader) Load(callbacks ...func() error) error {
	envs, err := self.lookup.Lookup(self.envFiles()...)
	if err != nil {
		return err
	}

	if len(envs) > 0 {
		if err := godotenv.Load(envs...); err != nil {
			return fmt.Errorf("can't load %v: %w", envs, err)
		}
	}

	for _, cb := range callbacks {
		if err := cb(); err != nil {
			return err
		}
	}
	return nil
}

// envFile returns list of .env files for searching, according to configured
// name of environment. See [Loader.Load] for details.
func (self *Loader) envFiles() []string {
	if self.envSuffix == "" {
		return []string{".env.local", ".env"}
	}

	return []string{
		".env." + self.envSuffix + ".local",
		".env.local",
		".env." + self.envSuffix,
		".env",
	}
}
