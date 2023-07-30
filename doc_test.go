package dotenv_test

import (
	"fmt"
	"log"
	"os"

	dotenv "github.com/dsh2dsh/expx-dotenv"
)

func Example() {
	env := dotenv.New()
	if err := env.Load(); err != nil {
		log.Fatalf("error loading .env files: %v", err)
	}
}

func Example_chainedCalls() {
	env := dotenv.New()
	if err := env.WithDepth(1).WithEnvSuffix("test").Load(); err != nil {
		log.Fatalf("error loading .env files: %v", err)
	}
}

func Example_withParse() {
	env := dotenv.New()
	cfg := struct {
		SomeOpt string `env:"ENV_VAR1"`
	}{
		SomeOpt: "some default value, because we don't have .env file(s)",
	}
	if err := env.LoadTo(&cfg); err != nil {
		log.Fatalf("error loading .env files: %v", err)
	}
	fmt.Println(cfg.SomeOpt)
	// Output: some default value, because we don't have .env file(s)
}

func ExampleLoader_WithRootDir() {
	env := dotenv.New()
	// "ENV_ROOT" environment variable contains name of current environment.
	env.WithRootDir(os.Getenv("ENV_ROOT"))
}

func ExampleLoader_WithRootFiles() {
	env := dotenv.New()
	// stop at dir, which contains ".git"
	env.WithRootFiles(".git")
}

func ExampleLoader_WithRootCallback() {
	env := dotenv.New()
	env.WithRootCallback(func(path string) (bool, error) {
		return env.FileExistsInDir(path, ".git") //nolint: wrapcheck
	})
}
