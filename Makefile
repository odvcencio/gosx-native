.PHONY: test smoke android-smoke android-connected demo lint

test:
	go test ./...

smoke:
	cd test/e2e && go test -tags smoke -v ./...

android-smoke:
	go run ./cmd/gsxnative emit android testdata/corpus/swift/counter.swift.gsx > examples/counter-android/app/src/main/kotlin/generated/Counter.kt
	cd examples/counter-android && gradle --no-daemon :gsx-nativekit:assembleRelease :app:assembleDebug

android-connected:
	go run ./cmd/gsxnative emit android testdata/corpus/swift/counter.swift.gsx > examples/counter-android/app/src/main/kotlin/generated/Counter.kt
	cd examples/counter-android && gradle --no-daemon :app:connectedDebugAndroidTest

demo:
	@echo "Running M1 vertical slice demo..."
	go run ./cmd/gsxnative emit ios testdata/corpus/swift/counter.swift.gsx > /tmp/Counter.swift
	go run ./cmd/gsxnative emit android testdata/corpus/swift/counter.swift.gsx > /tmp/Counter.kt
	@echo "Generated /tmp/Counter.swift"
	@echo "Generated /tmp/Counter.kt"

lint:
	go vet ./...
	@unformatted=$$(gofmt -l .); if [ -n "$$unformatted" ]; then echo "Unformatted files:"; echo "$$unformatted"; exit 1; fi
