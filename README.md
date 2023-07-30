# Go high level wrapper around [godotenv](https://github.com/joho/godotenv)

[![Go](https://github.com/dsh2dsh/expx-dotenv/actions/workflows/go.yml/badge.svg)](https://github.com/dsh2dsh/expx-dotenv/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/dsh2dsh/expx-dotenv.svg)](https://pkg.go.dev/github.com/dsh2dsh/expx-dotenv)

It allows to load one or multiple .env file(s) according to [original
rules](https://github.com/bkeepers/dotenv#what-other-env-files-can-i-use). It
searches for .env file(s) in current and parent dirs, until it find at least one
of them.

## Instalation
```shell
go get https://github.com/dsh2dsh/expx-dotenv
```

## Usage
```go
env := dotenv.New()
if err := env.Load(); err != nil {
	log.Fatalf("error loading .env files: %v", err)
}
```

or with chained calls:
```go
env := dotenv.New()
if err := env.WithDepth(1).WithEnvSuffix("test").Load(); err != nil {
	log.Fatalf("error loading .env files: %v", err)
}
```

Load environment variables and parse them into a struct:
```go
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
```
