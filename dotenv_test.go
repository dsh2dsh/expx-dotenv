package dotenv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dsh2dsh/expx-dotenv/internal/mocks"
)

var allEnvVars = []string{"TEST_VAR1", "TEST_VAR2"}

func valueNoError[V any](t *testing.T) func(val V, err error) V {
	return func(val V, err error) V {
		require.NoError(t, err)
		return val
	}
}

func TestWithDepth(t *testing.T) {
	env := New()
	assert.Equal(t, 0, env.lookupDepth)
	assert.Same(t, env, env.WithDepth(env.lookupDepth+1))
	assert.Equal(t, 1, env.lookupDepth)
}

func TestWithEnvVarName(t *testing.T) {
	env := New()
	assert.Equal(t, "", env.envSuffix)
	t.Setenv("ENV", "123")
	assert.Same(t, env, env.WithEnvVarName("ENV"))
	assert.Equal(t, "123", env.envSuffix)
}

func TestWithEnvSuffix(t *testing.T) {
	env := New()
	assert.Equal(t, "", env.envSuffix)
	assert.Same(t, env, env.WithEnvSuffix("123"))
	assert.Equal(t, "123", env.envSuffix)
}

func TestWithRootDir(t *testing.T) {
	env := New()
	assert.Equal(t, string(filepath.Separator), env.rootDir)

	curDir := valueNoError[string](t)(os.Getwd())
	parentDir := valueNoError[string](t)(filepath.Abs("../"))

	tests := []struct {
		withRootDir string
		expect      string
	}{
		{
			withRootDir: curDir,
			expect:      curDir,
		},
		{
			withRootDir: "./",
			expect:      curDir,
		},
		{
			withRootDir: "./abc",
			expect:      filepath.Join(curDir, "abc"),
		},
		{
			withRootDir: "../",
			expect:      parentDir,
		},
		{
			withRootDir: "testdata/a",
			expect:      filepath.Join(curDir, "testdata", "a"),
		},
	}

	for _, tt := range tests {
		assert.Same(t, env, env.WithRootDir(tt.withRootDir))
		assert.Equal(t, tt.expect, env.rootDir)
	}
}

func TestWithRootFiles(t *testing.T) {
	env := New()
	assert.Equal(t, []string{"go.mod"}, env.rootFiles)
	env.WithRootFiles(".git", "go.mod")
	assert.Equal(t, []string{".git", "go.mod"}, env.rootFiles)
}

func TestWithRootCallback(t *testing.T) {
	env := New()
	assert.Nil(t, env.rootCb)
	assert.False(t, valueNoError[bool](t)(env.stopByRootCb("")))

	var count int
	env.WithRootCallback(func(path string) (bool, error) {
		count++
		return true, nil
	})
	assert.NotNil(t, env.rootCb)
	assert.True(t, valueNoError[bool](t)(env.stopByRootCb("")))
	assert.Equal(t, 1, count)

	env.WithRootCallback(func(path string) (bool, error) {
		return false, os.ErrNotExist
	})
	_, err := env.stopByRootCb("")
	require.ErrorIs(t, err, os.ErrNotExist)

	env.WithRootCallback(func(path string) (bool, error) {
		return path == "/", nil
	})
	assert.False(t, valueNoError[bool](t)(env.stopByRootCb("")))
	assert.True(t, valueNoError[bool](t)(env.stopByRootCb("/")))
}

func TestFileExistsInDir(t *testing.T) {
	hasFile := "dotenv_test.go"

	tests := []struct {
		name      string
		dir       string
		file      string
		exists    bool
		newLoader func() *Loader
		cfgLoader func(l *Loader) *Loader
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
			newLoader: func() *Loader {
				filer := mocks.NewMockFiler(t)
				filer.EXPECT().Stat(hasFile).Return(nil, os.ErrNotExist)
				return New(WithFiler(filer))
			},
		},
		{
			name: "error from Stat",
			dir:  "",
			file: hasFile,
			newLoader: func() *Loader {
				filer := mocks.NewMockFiler(t)
				filer.EXPECT().Stat(hasFile).Return(nil, os.ErrInvalid)
				return New(WithFiler(filer))
			},
			wantErr: os.ErrInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var env *Loader
			if tt.newLoader != nil {
				env = tt.newLoader()
			} else {
				env = New()
			}
			exists, err := env.FileExistsInDir(tt.dir, tt.file)
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

func TestLoader_envFiles(t *testing.T) {
	env := New()
	assert.Equal(t, []string{".env.local", ".env"}, env.envFiles())

	env.WithEnvSuffix("test")
	assert.Equal(t,
		[]string{".env.test.local", ".env.local", ".env.test", ".env"},
		env.envFiles())
}

func TestLoader_checkLookupDepth(t *testing.T) {
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
			env := New()
			env.WithDepth(tt.lookupDepth)
			assert.Equal(t, tt.expect, env.checkLookupDepth(tt.curDepth))
		})
	}
}

func TestLoader_Load(t *testing.T) {
	tests := []struct {
		name       string
		dir        string
		envVarName string
		expect     string
		expectErr  bool
		before     func(*testing.T, *Loader)
	}{
		{
			name:       "in root dir",
			dir:        "testdata",
			envVarName: allEnvVars[0],
			expect:     "testdata",
		},
		{
			name:       "in subdir",
			dir:        "testdata/a",
			envVarName: allEnvVars[0],
			expect:     "testdata",
		},
		{
			name:       "with test suffix",
			dir:        "testdata/a",
			envVarName: allEnvVars[0],
			expect:     "testdata-test",
			before: func(t *testing.T, env *Loader) {
				env.WithEnvSuffix("test")
			},
		},
		{
			name:       "with test suffix but from .env",
			dir:        "testdata/a",
			envVarName: allEnvVars[1],
			expect:     "testdata2",
			before: func(t *testing.T, env *Loader) {
				env.WithEnvSuffix("test")
			},
		},
		{
			name:       "with test ENV",
			dir:        "testdata/a",
			envVarName: allEnvVars[0],
			expect:     "testdata-test",
			before: func(t *testing.T, env *Loader) {
				t.Setenv("ENV", "test")
				env.WithEnvVarName("ENV")
			},
		},
		{
			name:       "WithRootDir empty dir",
			dir:        "testdata/a",
			envVarName: allEnvVars[0],
			expect:     "",
			before: func(t *testing.T, env *Loader) {
				curDir := valueNoError[string](t)(os.Getwd())
				env.WithRootDir(curDir)
			},
		},
		{
			name:       "WithRootDir testdata",
			dir:        "testdata/a",
			envVarName: allEnvVars[0],
			expect:     "testdata",
			before: func(t *testing.T, env *Loader) {
				curDir := valueNoError[string](t)(os.Getwd())
				env.WithRootDir(filepath.Dir(curDir))
			},
		},
		{
			name:       "WithDepth 1",
			dir:        "testdata/a",
			envVarName: allEnvVars[0],
			expect:     "",
			before: func(t *testing.T, env *Loader) {
				env.WithDepth(1)
			},
		},
		{
			name:       "WithDepth 2",
			dir:        "testdata/a",
			envVarName: allEnvVars[0],
			expect:     "testdata",
			before: func(t *testing.T, env *Loader) {
				env.WithDepth(2)
			},
		},
		{
			name:       "with error from Load",
			dir:        "testdata",
			envVarName: allEnvVars[0],
			expectErr:  true,
			before: func(t *testing.T, env *Loader) {
				env.WithEnvSuffix("error")
			},
		},
		{
			name:       "WithRootCb stop at a",
			dir:        "testdata/a",
			envVarName: allEnvVars[0],
			expect:     "",
			before: func(t *testing.T, env *Loader) {
				env.WithRootCallback(func(path string) (bool, error) {
					return true, nil
				})
			},
		},
		{
			name:       "WithRootCb stop at testdata",
			dir:        "testdata/a",
			envVarName: allEnvVars[0],
			expect:     "testdata",
			before: func(t *testing.T, env *Loader) {
				curDir := valueNoError[string](t)(os.Getwd())
				env.WithRootCallback(func(path string) (bool, error) {
					return path == filepath.Dir(curDir), nil
				})
			},
		},
		{
			name:       "WithRootCb error",
			dir:        "testdata/a",
			envVarName: allEnvVars[0],
			expectErr:  true,
			before: func(t *testing.T, env *Loader) {
				env.WithRootCallback(func(path string) (bool, error) {
					return false, os.ErrNotExist
				})
			},
		},
		{
			name:       "WithRootFiles stop at go.mod",
			dir:        "testdata/b",
			envVarName: allEnvVars[0],
			expect:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changeDir(t, tt.dir)
			restoreEnvVars(t)
			env := New()
			if tt.before != nil {
				tt.before(t, env)
			}
			if tt.expectErr {
				require.Error(t, env.Load())
			} else {
				require.NoError(t, env.Load())
				assert.Equal(t, tt.expect, os.Getenv(tt.envVarName),
					"unexpected value of %v in %v dir", tt.envVarName, tt.dir)
			}
		})
	}
}

func changeDir(t *testing.T, path string) {
	curDir := valueNoError[string](t)(os.Getwd())
	require.NoError(t, os.Chdir(path))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(curDir))
	})
}

func restoreEnvVars(t *testing.T) {
	for _, v := range allEnvVars {
		t.Setenv(v, "")
		require.NoError(t, os.Unsetenv(v))
	}
}

func TestLoader_Load_errorGetwd(t *testing.T) {
	tmpDir := valueNoError[string](t)(os.MkdirTemp("", "expx-dotenv-"))
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(tmpDir))
	})

	curDir := valueNoError[string](t)(os.Getwd())
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(curDir))
	})

	env := New()
	require.NoError(t, os.RemoveAll(tmpDir))
	require.Error(t, env.Load())
}

func TestLoader_nextParentDir_error(t *testing.T) {
	filer := mocks.NewMockFiler(t)
	filer.EXPECT().Stat(mock.Anything).Return(nil, os.ErrInvalid)
	l := New(WithFiler(filer))

	nextDir, err := l.nextParentDir("")
	require.ErrorIs(t, err, os.ErrInvalid)
	assert.Equal(t, "", nextDir)
}

func TestLoader_lookupEnvDir_error(t *testing.T) {
	filer := mocks.NewMockFiler(t)
	filer.EXPECT().Stat(mock.Anything).Return(nil, os.ErrInvalid)
	l := New(WithFiler(filer))

	found, envDir, err := l.lookupEnvDir(l.envFiles())
	require.ErrorIs(t, err, os.ErrInvalid)
	assert.False(t, found)
	assert.Equal(t, "", envDir)
}

func TestLoader_lookupEnvFiles_error(t *testing.T) {
	filer := mocks.NewMockFiler(t)
	seen := make(map[string]struct{})
	filer.EXPECT().Stat(mock.Anything).RunAndReturn(
		func(name string) (os.FileInfo, error) {
			if _, ok := seen[name]; !ok {
				seen[name] = struct{}{}
				return os.Stat(name)
			}
			return nil, os.ErrInvalid
		})

	l := New(WithFiler(filer))
	changeDir(t, "testdata")
	envs, err := l.lookupEnvFiles()
	require.ErrorIs(t, err, os.ErrInvalid)
	assert.Nil(t, envs)
}

func TestLoader_Load_withCallbacks(t *testing.T) {
	var callCnt int

	tests := []struct {
		name      string
		callbacks []func() error
		wantErr   error
		assert    func(t *testing.T)
	}{
		{
			name: "called without error",
			callbacks: []func() error{
				func() error {
					callCnt++
					return nil
				},
			},
			assert: func(t *testing.T) {
				assert.Equal(t, 1, callCnt)
			},
		},
		{
			name: "called all without error",
			callbacks: []func() error{
				func() error {
					callCnt++
					return nil
				},
				func() error {
					callCnt++
					return nil
				},
			},
			assert: func(t *testing.T) {
				assert.Equal(t, 2, callCnt)
			},
		},
		{
			name: "with first error",
			callbacks: []func() error{
				func() error {
					callCnt++
					return os.ErrInvalid
				},
				func() error {
					callCnt++
					return nil
				},
			},
			wantErr: os.ErrInvalid,
			assert: func(t *testing.T) {
				assert.Equal(t, 1, callCnt)
			},
		},
		{
			name: "with last error",
			callbacks: []func() error{
				func() error {
					callCnt++
					return nil
				},
				func() error {
					callCnt++
					return os.ErrInvalid
				},
			},
			wantErr: os.ErrInvalid,
			assert: func(t *testing.T) {
				assert.Equal(t, 2, callCnt)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCnt = 0
			err := New().WithDepth(1).Load(tt.callbacks...)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			tt.assert(t)
		})
	}
}

func TestLoad(t *testing.T) {
	changeDir(t, "testdata")
	restoreEnvVars(t)
	require.NoError(t, Load())
	assert.Equal(t, "testdata", os.Getenv(allEnvVars[0]))
}
