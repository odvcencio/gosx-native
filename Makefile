.PHONY: test smoke android-smoke android-connected android-managed demo lint

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

android-managed:
	go run ./cmd/gsxnative emit android testdata/corpus/swift/counter.swift.gsx > examples/counter-android/app/src/main/kotlin/generated/Counter.kt
	cd examples/counter-android && gradle --no-daemon :app:ciApi30DebugAndroidTest -Pandroid.testoptions.manageddevices.emulator.gpu=swiftshader_indirect

demo:
	@echo "Running native parity demo..."
	go run ./cmd/gsxnative emit ios testdata/corpus/swift/counter.swift.gsx > /tmp/Counter.swift
	go run ./cmd/gsxnative emit android testdata/corpus/swift/counter.swift.gsx > /tmp/Counter.kt
	go run ./cmd/gsxnative emit ios testdata/corpus/go/counter.gsx > /tmp/Counter.gosx.swift
	go run ./cmd/gsxnative emit android testdata/corpus/go/counter.gsx > /tmp/Counter.gosx.kt
	go run ./cmd/gsxnative emit ios testdata/corpus/go/panel.gsx > /tmp/Panel.gosx.swift
	go run ./cmd/gsxnative emit android testdata/corpus/go/panel.gsx > /tmp/Panel.gosx.kt
	go run ./cmd/gsxnative emit ios testdata/corpus/go/greeter.gsx > /tmp/Greeter.gosx.swift
	go run ./cmd/gsxnative emit android testdata/corpus/go/greeter.gsx > /tmp/Greeter.gosx.kt
	go run ./cmd/gsxnative emit ios testdata/corpus/go/derived.gsx > /tmp/Derived.gosx.swift
	go run ./cmd/gsxnative emit android testdata/corpus/go/derived.gsx > /tmp/Derived.gosx.kt
	@echo "Generated /tmp/Counter.swift"
	@echo "Generated /tmp/Counter.kt"
	@echo "Generated /tmp/Counter.gosx.swift"
	@echo "Generated /tmp/Counter.gosx.kt"
	@echo "Generated /tmp/Panel.gosx.swift"
	@echo "Generated /tmp/Panel.gosx.kt"
	@echo "Generated /tmp/Greeter.gosx.swift"
	@echo "Generated /tmp/Greeter.gosx.kt"
	@echo "Generated /tmp/Derived.gosx.swift"
	@echo "Generated /tmp/Derived.gosx.kt"

lint:
	go vet ./...
	@unformatted=$$(gofmt -l .); if [ -n "$$unformatted" ]; then echo "Unformatted files:"; echo "$$unformatted"; exit 1; fi
