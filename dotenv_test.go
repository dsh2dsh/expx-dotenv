package dotenv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var allEnvVars = []string{"TEST_VAR1", "TEST_VAR2"}

func valueNoError[V any](t *testing.T) func(val V, err error) V {
	return func(val V, err error) V {
		require.NoError(t, err)
		return val
	}
}

func TestLoader_WithDepth(t *testing.T) {
	env := New()
	assert.Same(t, env, env.WithDepth(env.lookup.lookupDepth+1))
	assert.Equal(t, 1, env.lookup.lookupDepth)
}

func TestLoader_WithEnvVarName(t *testing.T) {
	env := New()
	assert.Empty(t, env.envSuffix)
	t.Setenv("ENV", "123")
	assert.Same(t, env, env.WithEnvVarName("ENV"))
	assert.Equal(t, "123", env.envSuffix)
}

func TestLoader_WithEnvSuffix(t *testing.T) {
	env := New()
	assert.Empty(t, env.envSuffix)
	assert.Same(t, env, env.WithEnvSuffix("123"))
	assert.Equal(t, "123", env.envSuffix)
}

func TestLoader_WithRootDir(t *testing.T) {
	env := New()
	assert.Equal(t, string(filepath.Separator), env.lookup.rootDir)

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
		assert.Equal(t, tt.expect, env.lookup.rootDir)
	}
}

func TestLoader_WithRootFiles(t *testing.T) {
	env := New()
	assert.Equal(t, []string{"go.mod"}, env.lookup.rootFiles)
	env.WithRootFiles(".git", "go.mod")
	assert.Equal(t, []string{".git", "go.mod"}, env.lookup.rootFiles)
}

func TestLoader_FileExistsInDir(t *testing.T) {
	env := New()
	exists, err := env.FileExistsInDir("", "dotenv_test.go")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = env.FileExistsInDir("", "not exists")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestLoader_envFiles(t *testing.T) {
	env := New()
	assert.Equal(t, []string{".env.local", ".env"}, env.envFiles())

	env.WithEnvSuffix("test")
	assert.Equal(t,
		[]string{".env.test.local", ".env.local", ".env.test", ".env"},
		env.envFiles())
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
			t.Chdir(tt.dir)
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

func restoreEnvVars(t *testing.T) {
	for _, v := range allEnvVars {
		t.Setenv(v, "")
		require.NoError(t, os.Unsetenv(v))
	}
}

func TestLoader_Load_errorGetwd(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	env := New()
	require.NoError(t, os.RemoveAll(tmpDir))
	require.Error(t, env.Load())
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
	t.Chdir("testdata")
	restoreEnvVars(t)
	require.NoError(t, Load())
	assert.Equal(t, "testdata", os.Getenv(allEnvVars[0]))
}
