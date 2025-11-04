package main

// This secondary entrypoint is unused by the container build, which builds the root package.
// It exists so that `go build ./...` succeeds locally and in CI.
func main() {}
