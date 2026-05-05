.PHONY: test smoke demo lint

test:
	go test ./...

smoke:
	cd test/e2e && go test -tags smoke -v ./...

demo:
	@echo "Running M1 vertical slice demo..."
	go run ./cmd/gsxnative emit ios testdata/corpus/swift/counter.swift.gsx > /tmp/Counter.swift
	@echo "Generated /tmp/Counter.swift"

lint:
	go vet ./...
	@unformatted=$$(gofmt -l .); if [ -n "$$unformatted" ]; then echo "Unformatted files:"; echo "$$unformatted"; exit 1; fi
