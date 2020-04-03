# tt - Simple integration testing for GO code
[![Build Status](https://travis-ci.org/zviadm/tt.svg?branch=master)](https://travis-ci.org/zviadm/tt)

`tt` is a tool to write and run both simple and complex tests for GO projects. `tt` requires almost no configuration and works with standard go module setup.

Using `tt` in your project is simple.
* First setup `tt` and `docker`.
	* `go install github.com/zviadm/tt/tt`
	* install Docker (or Docker Desktop): https://docs.docker.com/install/

* Create `tt.Dockerfile` next to `go.mod` file or any of the parent directories in the same repository. Use [tt.Dockerfile](./tt.Dockerfile) as a starting point. Add version of GO that you care about, and any non-GO third party libraries that you might need. As a general suggestion, try to keep your `tt.Dockerfile` simple.
* Run tests: `$: tt <test args> <packages>`
	* `tt` supports most of the arguments from `go test`

# Testing

```
$: go run ./tt ./...
```