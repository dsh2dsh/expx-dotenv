package dotenv

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookup_WithRootCallback(t *testing.T) {
	l := NewLookup()
	assert.Nil(t, l.rootCb)
	assert.False(t, valueNoError[bool](t)(l.stopByRootCb("")))

	var count int
	l.WithRootCallback(func(path string) (bool, error) {
		count++
		return true, nil
	})
	assert.NotNil(t, l.rootCb)
	assert.True(t, valueNoError[bool](t)(l.stopByRootCb("")))
	assert.Equal(t, 1, count)

	l.WithRootCallback(func(path string) (bool, error) {
		return false, os.ErrNotExist
	})
	_, err := l.stopByRootCb("")
	require.ErrorIs(t, err, os.ErrNotExist)

	l.WithRootCallback(func(path string) (bool, error) {
		return path == "/", nil
	})
	assert.False(t, valueNoError[bool](t)(l.stopByRootCb("")))
	assert.True(t, valueNoError[bool](t)(l.stopByRootCb("/")))
}

func TestLookup_FileExistsInDir(t *testing.T) {
	const hasFile = "dotenv_test.go"

	tests := []struct {
		name      string
		dir       string
		file      string
		exists    bool
		newLookup func() *Lookup
		wantErr   error
	}{
		{
			name:   "exists in empty dir",
			dir:    "",
			file:   hasFile,
			exists: true,
		},
		{
			name:   "exists in dot dir",
			dir:    "./",
			file:   hasFile,
			exists: true,
		},
		{
			name:   "exists in testdata",
			dir:    "testdata",
			file:   ".env",
			exists: true,
		},
		{
			name: "doesnt exists",
			dir:  "",
			file: "not exists",
		},
		{
			name: "doesnt exists with error from Stat",
			dir:  "",
			file: hasFile,
			newLookup: func() *Lookup {
				l := NewLookup()
				l.stat = func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				}
				return l
			},
		},
		{
			name: "error from Stat",
			dir:  "",
			file: hasFile,
			newLookup: func() *Lookup {
				l := NewLookup()
				l.stat = func(name string) (os.FileInfo, error) {
					return nil, os.ErrInvalid
				}
				return l
			},
			wantErr: os.ErrInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var l *Lookup
			if tt.newLookup != nil {
				l = tt.newLookup()
			} else {
				l = NewLookup()
			}
			exists, err := l.FileExistsInDir(tt.dir, tt.file)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			if tt.exists {
				assert.True(t, exists, "expected file %v exists in dir %v",
					tt.file, tt.dir)
			} else {
				assert.False(t, exists, "not expected file %v exists in dir %v",
					tt.file, tt.dir)
			}
		})
	}
}

func TestLookup_checkLookupDepth(t *testing.T) {
	tests := []struct {
		name        string
		lookupDepth int
		curDepth    int
		expect      int
	}{
		{
			name:        "with negative lookupDepth",
			lookupDepth: -1,
			expect:      0,
		},
		{
			name:        "with zero lookupDepth",
			lookupDepth: 0,
			expect:      0,
		},
		{
			name:        "with positive lookupDepth and zero curDepth",
			lookupDepth: 2,
			expect:      1,
		},
		{
			name:        "with positive lookupDepth and one curDepth",
			lookupDepth: 2,
			curDepth:    1,
			expect:      -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLookup()
			l.WithDepth(tt.lookupDepth)
			assert.Equal(t, tt.expect, l.checkLookupDepth(tt.curDepth))
		})
	}
}

func TestLookup_nextParentDir_error(t *testing.T) {
	l := NewLookup()
	l.rootFiles = []string{".env.local", ".env"}
	l.stat = func(name string) (os.FileInfo, error) {
		return nil, os.ErrInvalid
	}

	nextDir, err := l.nextParentDir("")
	require.ErrorIs(t, err, os.ErrInvalid)
	assert.Empty(t, nextDir)
}

func TestLoader_lookupDir_error(t *testing.T) {
	l := NewLookup()
	l.stat = func(name string) (os.FileInfo, error) {
		return nil, os.ErrInvalid
	}

	found, dir, err := l.lookupDir([]string{".env.local", ".env"})
	require.ErrorIs(t, err, os.ErrInvalid)
	assert.False(t, found)
	assert.Empty(t, dir)
}

func TestLoader_Lookup_error(t *testing.T) {
	l := NewLookup()
	seen := make(map[string]struct{})
	l.stat = func(name string) (os.FileInfo, error) {
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			return os.Stat(name)
		}
		return nil, os.ErrInvalid
	}

	t.Chdir("testdata")
	envs, err := l.Lookup(".env.local", ".env")
	require.ErrorIs(t, err, os.ErrInvalid)
	assert.Nil(t, envs)

	wantErr := errors.New("oops!")
	l.err = wantErr
	_, err = l.Lookup(".env.local", ".env")
	require.ErrorIs(t, err, wantErr)
}
