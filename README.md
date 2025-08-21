## stacktrace-go

An alternative to Go's native stack trace, that:

- has colorized output
- is [fast](#benchmark)
- is thread-safe
- shows go statement that created the panicking goroutine
- allows to customize the printing of panic values
- supports [NO_COLOR](https://no-color.org) environment variable

Documentation: https://pkg.go.dev/github.com/antoniszymanski/stacktrace-go

### Installation:

```
go get github.com/antoniszymanski/stacktrace-go
```

### Example:

![Example](example.png)

### Benchmark:

```
goos: linux
goarch: amd64
pkg: github.com/antoniszymanski/stacktrace-go
cpu: Intel(R) Core(TM) i7-7700HQ CPU @ 2.80GHz
BenchmarkHandle-8   	   94354	     12849 ns/op	     680 B/op	       7 allocs/op
PASS
```
