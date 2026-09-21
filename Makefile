.PHONY: test test-race bench bench-all lint vet build clean cover profile

# Run all tests
test:
	go test ./...

# Run all tests with race detector
test-race:
	go test -race -count=1 ./...

# Run benchmarks (algorithms only)
bench:
	go test -bench=. -benchmem -benchtime=3s ./algorithms/

# Run all benchmarks across all packages
bench-all:
	go test -bench=. -benchmem ./...

# Run go vet
vet:
	go vet ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run ./...

# Build all packages
build:
	go build ./...

# Generate coverage report
cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# CPU profiling
profile:
	go test -cpuprofile=cpu.pprof -memprofile=mem.pprof -bench=. -benchtime=5s ./algorithms/
	@echo "CPU profile: cpu.pprof"
	@echo "Memory profile: mem.pprof"
	@echo "View with: go tool pprof cpu.pprof"

# Clean generated files
clean:
	rm -f cpu.pprof mem.pprof coverage.out coverage.html
	rm -f *.test
