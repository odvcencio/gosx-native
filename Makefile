SCENE3D_NATIVE_FIXTURES := scene3d_instancing scene3d_postfx scene3d_compute scene3d_html scene3d_canvas scene3d_spread

.PHONY: test smoke scene-conform build-ios build-android build-all build-scene3d-ios build-scene3d-android build-scene3d-fixtures-ios build-scene3d-fixtures-android android-smoke android-connected android-managed demo lint

test:
	go test ./...

smoke:
	cd test/e2e && go test -tags smoke -v ./...

scene-conform:
	go run ./cmd/gsxnative scene-conform

build-ios:
	go run ./cmd/gsxnative build ios

build-android:
	go run ./cmd/gsxnative build android

build-all:
	go run ./cmd/gsxnative build all

build-scene3d-ios:
	go run ./cmd/gsxnative build ios --source testdata/corpus/go/scene3d.gsx --output examples/counter-ios/CounterDemo/Generated/SceneDemo.swift

build-scene3d-android:
	go run ./cmd/gsxnative build android --source testdata/corpus/go/scene3d.gsx --output examples/counter-android/app/src/main/kotlin/generated/SceneDemo.kt --task :app:compileDebugKotlin

build-scene3d-fixtures-ios:
	@set -eu; \
	derived_data="$$(mktemp -d)"; \
	cleanup() { rm -f examples/counter-ios/CounterDemo/Generated/SceneFixture_*.swift; rm -rf "$$derived_data"; }; \
	trap cleanup EXIT; \
	for fixture in $(SCENE3D_NATIVE_FIXTURES); do \
		go run ./cmd/gsxnative emit ios "testdata/corpus/go/$$fixture.gsx" > "examples/counter-ios/CounterDemo/Generated/SceneFixture_$$fixture.swift"; \
	done; \
	(cd examples/counter-ios && xcodegen generate); \
	destination="$${IOS_SIMULATOR_DESTINATION:-}"; \
	if [ -z "$$destination" ]; then \
		if [ -n "$${IOS_SIMULATOR_NAME:-}" ]; then \
			destination="platform=iOS Simulator,name=$$IOS_SIMULATOR_NAME"; \
		else \
			destination="generic/platform=iOS Simulator"; \
		fi; \
	fi; \
	xcodebuild \
		-project examples/counter-ios/CounterDemo.xcodeproj \
		-scheme CounterDemo \
		-destination "$$destination" \
		-derivedDataPath "$$derived_data" \
		build

build-scene3d-fixtures-android:
	@set -eu; \
	cleanup() { rm -f examples/counter-android/app/src/main/kotlin/generated/SceneFixture_*.kt; }; \
	trap cleanup EXIT; \
	for fixture in $(SCENE3D_NATIVE_FIXTURES); do \
		go run ./cmd/gsxnative emit android "testdata/corpus/go/$$fixture.gsx" > "examples/counter-android/app/src/main/kotlin/generated/SceneFixture_$$fixture.kt"; \
	done; \
	(cd examples/counter-android && gradle --no-daemon :app:compileDebugKotlin)

android-smoke:
	go run ./cmd/gsxnative build android

android-connected:
	go run ./cmd/gsxnative build android --task :app:connectedDebugAndroidTest

android-managed:
	go run ./cmd/gsxnative build android --task :app:ciApi30DebugAndroidTest --gradle-property android.testoptions.manageddevices.emulator.gpu=swiftshader_indirect

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
	go run ./cmd/gsxnative emit ios testdata/corpus/go/conditional.gsx > /tmp/Toggle.gosx.swift
	go run ./cmd/gsxnative emit android testdata/corpus/go/conditional.gsx > /tmp/Toggle.gosx.kt
	go run ./cmd/gsxnative emit ios testdata/corpus/go/component_ref.gsx > /tmp/Profile.gosx.swift
	go run ./cmd/gsxnative emit android testdata/corpus/go/component_ref.gsx > /tmp/Profile.gosx.kt
	go run ./cmd/gsxnative emit ios testdata/corpus/go/loop.gsx > /tmp/Roster.gosx.swift
	go run ./cmd/gsxnative emit android testdata/corpus/go/loop.gsx > /tmp/Roster.gosx.kt
	go run ./cmd/gsxnative emit ios testdata/corpus/go/form_controls.gsx > /tmp/FormControls.gosx.swift
	go run ./cmd/gsxnative emit android testdata/corpus/go/form_controls.gsx > /tmp/FormControls.gosx.kt
	go run ./cmd/gsxnative emit ios testdata/corpus/go/expressions.gsx > /tmp/Expressions.gosx.swift
	go run ./cmd/gsxnative emit android testdata/corpus/go/expressions.gsx > /tmp/Expressions.gosx.kt
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
	@echo "Generated /tmp/Toggle.gosx.swift"
	@echo "Generated /tmp/Toggle.gosx.kt"
	@echo "Generated /tmp/Profile.gosx.swift"
	@echo "Generated /tmp/Profile.gosx.kt"
	@echo "Generated /tmp/Roster.gosx.swift"
	@echo "Generated /tmp/Roster.gosx.kt"
	@echo "Generated /tmp/FormControls.gosx.swift"
	@echo "Generated /tmp/FormControls.gosx.kt"
	@echo "Generated /tmp/Expressions.gosx.swift"
	@echo "Generated /tmp/Expressions.gosx.kt"

lint:
	go vet ./...
	@unformatted=$$(gofmt -l .); if [ -n "$$unformatted" ]; then echo "Unformatted files:"; echo "$$unformatted"; exit 1; fi
