package dotenv_test

import (
	"log"
	"os"

	dotenv "github.com/dsh2dsh/expx-dotenv"
)

func Example() {
	if err := dotenv.New().Load(); err != nil {
		log.Fatalf("error loading .env files: %v", err)
	}
}

func Example_chainedCalls() {
	if err := dotenv.New().WithDepth(1).WithEnvSuffix("test").Load(); err != nil {
		log.Fatalf("error loading .env files: %v", err)
	}
}

// func Example_withParse() {
// 	cfg := struct {
// 		SomeOpt string `env:"ENV_VAR1"`
// 	}{
// 		SomeOpt: "some default value, because we don't have .env file(s)",
// 	}

// 	err := dotenv.New().Load(func() error {
// 		return env.Parse(&cfg) //nolint:wrapcheck
// 	})
// 	if err != nil {
// 		log.Fatalf("error loading .env files: %v", err)
// 	}

// 	fmt.Println(cfg.SomeOpt)
// 	// Output: some default value, because we don't have .env file(s)
// }

func ExampleLoader_WithRootDir() {
	// "ENV_ROOT" environment variable contains name of current environment.
	dotenv.New().WithRootDir(os.Getenv("ENV_ROOT"))
}

func ExampleLoader_WithRootFiles() {
	// stop at dir, which contains ".git"
	dotenv.New().WithRootFiles(".git")
}

func ExampleLoader_WithRootCallback() {
	env := dotenv.New()
	env.WithRootCallback(func(path string) (bool, error) {
		return env.FileExistsInDir(path, ".git") //nolint: wrapcheck
	})
}
