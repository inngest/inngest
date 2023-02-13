This package tests every config variation against every function definition,
in an N*N fashion, automatically.

## Writing new functions

You can use any inngest function to test.

1. Run `inngest init` within the `fns` dir
2. Create a `run.go` file which asserts that the services write specific logs
   according to the function's steps
3. Register the function's test DSL via `testdsl.Register`
4. Import the function as a package within configs.go.

## Running tests

```
go run . -test.v
```

Filtering:

```
go run . -test.v -test.run ${regex}
```

For example, to run `async-timeout` against immemory:

```
go run . -test.v -test.run inmemory-async
```

Or for all inmemory tests only:

```
go run . -test.v -test.run inmemory
```
