package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/odvcencio/gosx/nir"
)

func TestInitScaffoldsNativeProject(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sample-app")
	if err := runInit([]string{dir, "--name", "SampleApp", "--module", "com.example.sample"}); err != nil {
		t.Fatalf("init: %v", err)
	}

	assertFileContains(t, filepath.Join(dir, "src/app.gsx"), "func Home(props HomeProps) Node")
	assertFileContains(t, filepath.Join(dir, "src/app.gsx"), "//gosx:route name=home path=/ component=Home")
	assertFileContains(t, filepath.Join(dir, "src/app.gsx"), "//gosx:data name=loadGreeting method=GET path=/api/greeting output=message:string ttl=30s retry=2 backoff=250ms max_backoff=2s auth=optional network=cache_when_offline")
	assertFileContains(t, filepath.Join(dir, "src/app.gsx"), "//gosx:capability name=network targets=ios,android required=true")
	assertFileContains(t, filepath.Join(dir, "src/app.gsx"), "//gosx:bridge service=Vault method=echo path=/api/bridge/Vault.echo input=message:string output=message:string auth=required retry=2")
	assertFileContains(t, filepath.Join(dir, "README.md"), "gsxnative build all --release --artifact-manifest build/gsxnative-artifacts.json")
	assertFileContains(t, filepath.Join(dir, "README.md"), ".gsxnative/signing.json")
	assertFileContains(t, filepath.Join(dir, ".gsxnative/.gitignore"), "signing.json")
	assertFileContains(t, filepath.Join(dir, "ios/project.yml"), "name: SampleApp")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/SampleAppApp.swift"), "GSXRouter(initial: GSXRoutes.home)")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/SampleAppApp.swift"), "NativeObservability.install()")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Native/BridgeServices.swift"), "struct NativeCapabilityProvider: GSXCapabilityProvider")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Native/BridgeServices.swift"), `static let available: Set<String> = ["network"]`)
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Native/BridgeServices.swift"), "final class VaultBridge: GSXBridgeService")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Native/BridgeServices.swift"), "func dispatch(_ envelope: GSXBridgeEnvelope) async throws -> GSXBridgeResult")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Native/BridgeServices.swift"), `let input = try envelope.decodedPayload(GSXGeneratedBridgeVaultEchoInput.self)`)
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Native/BridgeServices.swift"), "func echo(message: String) async throws -> GSXGeneratedBridgeVaultEchoResponse")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Native/Observability.swift"), "GSXCrashReporting.shared.configure(reporter: crashReporter)")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Native/Observability.swift"), "GSXCrashReporting.shared.installUncaughtExceptionHandler()")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/App.g.swift"), "public struct Home: GSXComponent")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public enum GSXRoutes")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public enum GSXCapabilities")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), `GSXGeneratedCapabilitySpec(name: "network", targets: ["ios", "android"], required: true)`)
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public enum GSXBridges")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public final class GSXGeneratedBridgeClient")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public func vaultEcho(message: String) async throws -> GSXGeneratedBridgeVaultEchoResponse")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public convenience init(transport: any GSXTransport, networkStatusProvider: any GSXNetworkStatusProvider)")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public convenience init(baseURL: String, defaultHeaders: [String: String] = [:]) throws")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public convenience init(baseURL: String, defaultHeaders: [String: String] = [:], tokenStore: any GSXTokenStore) throws")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public enum GSXDataLoaders")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public struct GSXGeneratedDataLoadGreetingResponse")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public func loadGreeting() async throws -> GSXGeneratedDataLoadGreetingResponse")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), `GSXRequestPolicy(name: "loadGreeting", cacheTTLSeconds: 30, auth: GSXAuthRequirement.optional, retryAttempts: 2, retryBaseDelayMillis: 250, retryMaxDelayMillis: 2000, networkPolicy: GSXNetworkPolicy.cacheWhenOffline)`)
	assertFileContains(t, filepath.Join(dir, "android/settings.gradle.kts"), "project(\":gsx-nativekit\").projectDir")
	assertFileContains(t, filepath.Join(dir, "android/app/build.gradle.kts"), "val gsxSigningStoreFile = providers.gradleProperty(\"gsxSigningStoreFile\")")
	assertFileContains(t, filepath.Join(dir, "android/app/build.gradle.kts"), "create(\"gsxRelease\")")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/com/example/sample/MainActivity.kt"), "rememberGSXRouter(GSXRoutes.home)")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/com/example/sample/MainActivity.kt"), "NativeObservability.install()")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/com/example/sample/BridgeServices.kt"), "class NativeCapabilityProvider : GSXCapabilityProvider")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/com/example/sample/BridgeServices.kt"), `val available: Set<String> = setOf("network")`)
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/com/example/sample/BridgeServices.kt"), "class VaultBridge : GSXBridgeService")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/com/example/sample/BridgeServices.kt"), "override suspend fun dispatch(envelope: GSXBridgeEnvelope): GSXBridgeResult")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/com/example/sample/BridgeServices.kt"), "val input = GSXGeneratedBridgeVaultEchoInput.fromJSON(envelope.payload.orEmpty())")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/com/example/sample/BridgeServices.kt"), "suspend fun echo(message: String): GSXGeneratedBridgeVaultEchoResponse")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/com/example/sample/Observability.kt"), "GSXCrashReporting.configure(crashReporter)")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/com/example/sample/Observability.kt"), "GSXCrashReporting.installDefaultUncaughtExceptionHandler()")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/App.kt"), "fun Home(props: HomeProps)")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "object GSXRoutes")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "object GSXCapabilities")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), `GSXGeneratedCapabilitySpec(name = "network", targets = listOf("ios", "android"), required = true)`)
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "object GSXBridges")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "class GSXGeneratedBridgeClient")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "suspend fun vaultEcho(message: String): GSXGeneratedBridgeVaultEchoResponse")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "constructor(transport: GSXTransport, networkStatusProvider: GSXNetworkStatusProvider)")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "import com.gosx.nativekit.GSXNetworkStatusProvider")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "import com.gosx.nativekit.GSXTokenStore")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "constructor(baseURL: String, defaultHeaders: Map<String, String> = emptyMap())")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "constructor(baseURL: String, defaultHeaders: Map<String, String> = emptyMap(), tokenStore: GSXTokenStore)")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "object GSXDataLoaders")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "data class GSXGeneratedDataLoadGreetingResponse")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "suspend fun loadGreeting(): GSXGeneratedDataLoadGreetingResponse")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), `GSXRequestPolicy(name = "loadGreeting", cacheTTLSeconds = 30, auth = GSXAuthRequirement.Optional, retryAttempts = 2, retryBaseDelayMillis = 250, retryMaxDelayMillis = 2000, networkPolicy = GSXNetworkPolicy.CacheWhenOffline)`)

	var cfg projectConfig
	data, err := os.ReadFile(filepath.Join(dir, "gosxnative.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Source != "src/app.gsx" || cfg.IOS.SupportOutput == "" || cfg.Android.SupportOutput == "" ||
		len(cfg.Routes) != 0 || len(cfg.DataLoaders) != 0 || len(cfg.Actions) != 0 ||
		len(cfg.Capabilities) != 0 || len(cfg.Bridges) != 0 {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestInitRefusesNonEmptyDirectoryWithoutForce(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	err := runInit([]string{"--name", "SampleApp", dir})
	if err == nil {
		t.Fatalf("expected non-empty directory error")
	}
	if !strings.Contains(err.Error(), "is not empty") {
		t.Fatalf("expected non-empty directory error, got %v", err)
	}
}

func TestBuildUsesDiscoveredProjectConfig(t *testing.T) {
	fake := useFakeBuildRunner(t)
	dir := filepath.Join(t.TempDir(), "sample-app")
	if err := runInit([]string{"--name", "SampleApp", "--module", "com.example.sample", dir}); err != nil {
		t.Fatalf("init: %v", err)
	}

	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousDir)
	})

	if err := runBuild([]string{"android"}); err != nil {
		t.Fatalf("build from config: %v", err)
	}

	generated := filepath.Join(dir, "android/app/src/main/kotlin/generated/App.kt")
	assertFileContains(t, generated, "GSXScene3D(scene = GSXScene3DScene(")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "class GSXGeneratedActionClient")
	if len(fake.commands) != 1 {
		t.Fatalf("expected one Gradle command, got %#v", fake.commands)
	}
	if fake.commands[0].dir != filepath.Join(dir, "android") {
		t.Fatalf("expected Gradle to run in generated Android project, got %#v", fake.commands[0])
	}
}

func TestBuildIOSUsesDiscoveredProjectConfig(t *testing.T) {
	fake := useFakeBuildRunner(t)
	dir := filepath.Join(t.TempDir(), "sample-app")
	if err := runInit([]string{"--name", "SampleApp", "--module", "com.example.sample", dir}); err != nil {
		t.Fatalf("init: %v", err)
	}

	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(filepath.Join(dir, "src")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousDir)
	})

	if err := runBuild([]string{"ios"}); err != nil {
		t.Fatalf("build from config: %v", err)
	}

	if len(fake.commands) != 2 {
		t.Fatalf("expected xcodegen and xcodebuild, got %#v", fake.commands)
	}
	if fake.commands[0].dir != filepath.Join(dir, "ios") || fake.commands[0].name != "xcodegen" {
		t.Fatalf("unexpected xcodegen command: %#v", fake.commands[0])
	}
	joinedArgs := strings.Join(fake.commands[1].args, " ")
	if !strings.Contains(joinedArgs, filepath.Join(dir, "ios", "SampleApp.xcodeproj")) ||
		!strings.Contains(joinedArgs, "-scheme SampleApp") {
		t.Fatalf("unexpected xcodebuild args: %q", joinedArgs)
	}
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public final class GSXGeneratedActionClient")
}

func TestBuildValidatesSourceRouteComponents(t *testing.T) {
	fake := useFakeBuildRunner(t)
	dir := filepath.Join(t.TempDir(), "sample-app")
	if err := runInit([]string{"--name", "SampleApp", "--module", "com.example.sample", dir}); err != nil {
		t.Fatalf("init: %v", err)
	}

	sourcePath := filepath.Join(dir, "src/app.gsx")
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	updated := strings.Replace(string(data), "name=home path=/ component=Home", "name=home path=/ component=Missing", 1)
	if err := os.WriteFile(sourcePath, []byte(updated), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	err = runBuild([]string{"android", "--config", filepath.Join(dir, "gosxnative.json")})
	if err == nil {
		t.Fatalf("expected route validation error")
	}
	if !strings.Contains(err.Error(), `route home references unknown component "Missing"`) {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fake.commands) != 0 {
		t.Fatalf("expected no native build commands, got %#v", fake.commands)
	}
}

func TestBuildUsesConfigDeclarationsWhenSourceHasNone(t *testing.T) {
	useFakeBuildRunner(t)
	dir := filepath.Join(t.TempDir(), "sample-app")
	if err := runInit([]string{"--name", "SampleApp", "--module", "com.example.sample", dir}); err != nil {
		t.Fatalf("init: %v", err)
	}

	sourcePath := filepath.Join(dir, "src/app.gsx")
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "//gosx:route") ||
			strings.HasPrefix(strings.TrimSpace(line), "//gosx:data") ||
			strings.HasPrefix(strings.TrimSpace(line), "//gosx:action") ||
			strings.HasPrefix(strings.TrimSpace(line), "//gosx:capability") ||
			strings.HasPrefix(strings.TrimSpace(line), "//gosx:bridge") {
			continue
		}
		lines = append(lines, line)
	}
	if err := os.WriteFile(sourcePath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	configPath := filepath.Join(dir, "gosxnative.json")
	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg projectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	cfg.Routes = []routeDeclaration{{Name: "legacyHome", Path: "/", Component: "Home", Auth: "required"}}
	cfg.DataLoaders = []endpointDeclaration{{Name: "legacyLoad", Method: "GET", Path: "/api/legacy"}}
	cfg.Capabilities = []capabilityDeclaration{{Name: "secureStorage", Targets: []string{"ios"}, Required: true}}
	cfg.Bridges = []bridgeDeclaration{{
		Service: "LegacyVault",
		Method:  "echo",
		Path:    "/api/bridge/LegacyVault.echo",
		Input:   []paramDeclaration{{Name: "message", Type: "string"}},
		Output:  []paramDeclaration{{Name: "message", Type: "string"}},
		Auth:    "required",
	}}
	updated, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, append(updated, '\n'), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := runBuild([]string{"ios", "--config", configPath}); err != nil {
		t.Fatalf("build from config declarations: %v", err)
	}
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), `GSXGeneratedRouteSpec(name: "legacyHome"`)
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), `auth: GSXAuthRequirement.required`)
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public func legacyLoad() async throws -> GSXResponse")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), `GSXGeneratedCapabilitySpec(name: "secureStorage", targets: ["ios"], required: true)`)
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public func legacyVaultEcho(message: String) async throws -> GSXGeneratedBridgeLegacyVaultEchoResponse")
}

func TestParseSourceDeclarations(t *testing.T) {
	decls, err := parseSourceDeclarations([]byte(`
//gosx:route name=home path=/ component=Home
//gosx:route name=details path=/details/:id component=Home params=id:string,count:int auth=required
//gosx:data name=loadGreeting method=GET path=/api/greeting params=locale:string output=message:string ttl=45s retry=2 backoff=250ms max_backoff=2s auth=optional network=cache_when_offline
//gosx:action name=submitGreeting method=POST path=/api/greeting input=message:string output=message:string invalidates=loadGreeting optimistic=echo auth=required retry=3 backoff=100
//gosx:capability name=network targets=ios,android required=true
//gosx:bridge service=Vault method=encrypt path=/api/bridge/Vault.encrypt input=plain:string output=cipher:string auth=required retry=2
//gosx:native swift
`))
	if err != nil {
		t.Fatalf("parse source declarations: %v", err)
	}
	if len(decls.Routes) != 2 || len(decls.DataLoaders) != 1 || len(decls.Actions) != 1 ||
		len(decls.Capabilities) != 1 || len(decls.Bridges) != 1 {
		t.Fatalf("unexpected declarations: %#v", decls)
	}
	if got := decls.Routes[1].Params; len(got) != 2 || got[0].Name != "id" || got[1].Type != "int" {
		t.Fatalf("unexpected params: %#v", got)
	}
	if decls.Routes[1].Auth != "required" {
		t.Fatalf("unexpected route auth: %#v", decls.Routes[1])
	}
	if loader := decls.DataLoaders[0]; len(loader.Params) != 1 || len(loader.Output) != 1 ||
		loader.CacheTTLSeconds != 45 || loader.RetryAttempts != 2 || loader.RetryBaseMillis != 250 ||
		loader.RetryMaxMillis != 2000 || loader.Auth != "optional" || loader.NetworkPolicy != "cache_when_offline" {
		t.Fatalf("unexpected loader declaration: %#v", loader)
	}
	if action := decls.Actions[0]; len(action.Input) != 1 || len(action.Output) != 1 ||
		len(action.Invalidates) != 1 || action.Invalidates[0] != "loadGreeting" ||
		action.Optimistic != "echo" || action.Auth != "required" || action.RetryAttempts != 3 ||
		action.RetryBaseMillis != 100 {
		t.Fatalf("unexpected action declaration: %#v", action)
	}
	if capability := decls.Capabilities[0]; capability.Name != "network" || len(capability.Targets) != 2 || !capability.Required {
		t.Fatalf("unexpected capability declaration: %#v", capability)
	}
	if bridge := decls.Bridges[0]; bridge.Service != "Vault" || bridge.Method != "encrypt" || bridge.Auth != "required" ||
		bridge.RetryAttempts != 2 || len(bridge.Input) != 1 || len(bridge.Output) != 1 {
		t.Fatalf("unexpected bridge declaration: %#v", bridge)
	}
}

func TestParseSourceDeclarationsRejectsInvalidNetworkPolicy(t *testing.T) {
	_, err := parseSourceDeclarations([]byte(`
//gosx:data name=loadGreeting method=GET path=/api/greeting network=carrier_wave
`))
	if err == nil || !strings.Contains(err.Error(), "network policy") {
		t.Fatalf("expected invalid network policy error, got %v", err)
	}
}

func TestEmitTypedEndpointDeclarations(t *testing.T) {
	cfg := &projectConfig{
		Routes: []routeDeclaration{{
			Name:      "profile",
			Path:      "/users/:id/profile",
			Component: "Profile",
			Params:    []paramDeclaration{{Name: "id", Type: "string"}},
			Auth:      "required",
		}},
		DataLoaders: []endpointDeclaration{{
			Name:            "loadProfile",
			Method:          "GET",
			Path:            "/api/users/:id/profile",
			Params:          []paramDeclaration{{Name: "id", Type: "string"}, {Name: "includePosts", Type: "bool"}},
			Output:          []paramDeclaration{{Name: "displayName", Type: "string"}, {Name: "postCount", Type: "int"}},
			CacheTTLSeconds: 120,
			Auth:            "required",
			RetryAttempts:   3,
			RetryBaseMillis: 250,
			RetryMaxMillis:  2000,
			NetworkPolicy:   "cache_when_offline",
		}},
		Actions: []endpointDeclaration{{
			Name:            "saveProfile",
			Method:          "PATCH",
			Path:            "/api/users/:id/profile",
			Params:          []paramDeclaration{{Name: "id", Type: "string"}},
			Input:           []paramDeclaration{{Name: "displayName", Type: "string"}},
			Output:          []paramDeclaration{{Name: "displayName", Type: "string"}},
			Invalidates:     []string{"loadProfile"},
			Optimistic:      "profileEcho",
			Auth:            "required",
			RetryBaseMillis: 100,
		}},
	}

	swift := string(emitSwiftDeclarations(cfg))
	if !strings.Contains(swift, `GSXRequest.resolvedPath("/api/users/:id/profile", params: ["id": id, "includePosts": String(includePosts)])`) {
		t.Fatalf("expected Swift path parameter lowering, got:\n%s", swift)
	}
	if !strings.Contains(swift, `auth: GSXAuthRequirement.required`) {
		t.Fatalf("expected Swift route auth metadata, got:\n%s", swift)
	}
	if !strings.Contains(swift, "public func loadProfile(id: String, includePosts: Bool) async throws -> GSXGeneratedDataLoadProfileResponse") {
		t.Fatalf("expected Swift typed loader signature, got:\n%s", swift)
	}
	if !strings.Contains(swift, `GSXRequestPolicy(name: "loadProfile", cacheTTLSeconds: 120, auth: GSXAuthRequirement.required, retryAttempts: 3, retryBaseDelayMillis: 250, retryMaxDelayMillis: 2000, networkPolicy: GSXNetworkPolicy.cacheWhenOffline)`) {
		t.Fatalf("expected Swift loader network policy metadata, got:\n%s", swift)
	}
	if !strings.Contains(swift, `GSXRequestPolicy(name: "saveProfile", invalidates: ["loadProfile"], optimistic: "profileEcho", auth: GSXAuthRequirement.required, retryBaseDelayMillis: 100)`) {
		t.Fatalf("expected Swift action policy metadata, got:\n%s", swift)
	}

	kotlin := string(emitKotlinDeclarations(cfg))
	if !strings.Contains(kotlin, `GSXRequest.resolvedPath("/api/users/:id/profile", params = mapOf("id" to id, "includePosts" to includePosts.toString()))`) {
		t.Fatalf("expected Kotlin path parameter lowering, got:\n%s", kotlin)
	}
	if !strings.Contains(kotlin, `auth = GSXAuthRequirement.Required`) {
		t.Fatalf("expected Kotlin route auth metadata, got:\n%s", kotlin)
	}
	if !strings.Contains(kotlin, "suspend fun saveProfile(id: String, displayName: String): GSXGeneratedActionSaveProfileResponse") {
		t.Fatalf("expected Kotlin typed action signature, got:\n%s", kotlin)
	}
	if !strings.Contains(kotlin, `GSXRequestPolicy(name = "loadProfile", cacheTTLSeconds = 120, auth = GSXAuthRequirement.Required, retryAttempts = 3, retryBaseDelayMillis = 250, retryMaxDelayMillis = 2000, networkPolicy = GSXNetworkPolicy.CacheWhenOffline)`) {
		t.Fatalf("expected Kotlin loader network policy metadata, got:\n%s", kotlin)
	}
	if !strings.Contains(kotlin, `GSXRequestPolicy(name = "saveProfile", invalidates = listOf("loadProfile"), optimistic = "profileEcho", auth = GSXAuthRequirement.Required, retryBaseDelayMillis = 100)`) {
		t.Fatalf("expected Kotlin action policy metadata, got:\n%s", kotlin)
	}
}

func TestEmitBridgeCapabilityDeclarations(t *testing.T) {
	cfg := &projectConfig{
		Capabilities: []capabilityDeclaration{{
			Name:     "secureStorage",
			Targets:  []string{"ios", "android"},
			Required: true,
		}},
		Bridges: []bridgeDeclaration{{
			Service:       "Vault",
			Method:        "encrypt",
			Path:          "/api/bridge/Vault.encrypt",
			Input:         []paramDeclaration{{Name: "plain", Type: "string"}},
			Output:        []paramDeclaration{{Name: "cipher", Type: "string"}},
			Auth:          "required",
			RetryAttempts: 2,
		}},
	}

	swift := string(emitSwiftDeclarations(cfg))
	if !strings.Contains(swift, `GSXGeneratedCapabilitySpec(name: "secureStorage", targets: ["ios", "android"], required: true)`) {
		t.Fatalf("expected Swift capability spec, got:\n%s", swift)
	}
	if !strings.Contains(swift, "public static func negotiate(") ||
		!strings.Contains(swift, "try await negotiator.negotiate(required: runtimeSpecs, target: target, path: path)") {
		t.Fatalf("expected Swift capability negotiation helper, got:\n%s", swift)
	}
	if !strings.Contains(swift, "public func vaultEncrypt(plain: String) async throws -> GSXGeneratedBridgeVaultEncryptResponse") {
		t.Fatalf("expected Swift bridge method, got:\n%s", swift)
	}
	if !strings.Contains(swift, `let request = try GSXRequest.json(method: "POST", path: "/api/bridge/Vault.encrypt", body: input)`) {
		t.Fatalf("expected Swift bridge JSON request, got:\n%s", swift)
	}
	if !strings.Contains(swift, `GSXRequestPolicy(name: "Vault.encrypt", auth: GSXAuthRequirement.required, retryAttempts: 2)`) {
		t.Fatalf("expected Swift bridge policy, got:\n%s", swift)
	}

	kotlin := string(emitKotlinDeclarations(cfg))
	if !strings.Contains(kotlin, `GSXGeneratedCapabilitySpec(name = "secureStorage", targets = listOf("ios", "android"), required = true)`) {
		t.Fatalf("expected Kotlin capability spec, got:\n%s", kotlin)
	}
	if !strings.Contains(kotlin, "suspend fun negotiate(") ||
		!strings.Contains(kotlin, "negotiator.negotiate(required = runtimeSpecs, target = target, path = path)") {
		t.Fatalf("expected Kotlin capability negotiation helper, got:\n%s", kotlin)
	}
	if !strings.Contains(kotlin, "suspend fun vaultEncrypt(plain: String): GSXGeneratedBridgeVaultEncryptResponse") {
		t.Fatalf("expected Kotlin bridge method, got:\n%s", kotlin)
	}
	if !strings.Contains(kotlin, "fun fromJSON(json: String): GSXGeneratedBridgeVaultEncryptInput") ||
		!strings.Contains(kotlin, "fun toJSON(): String") {
		t.Fatalf("expected Kotlin bridge models to encode and decode, got:\n%s", kotlin)
	}
	if !strings.Contains(kotlin, `val request = GSXRequest.json(method = "POST", path = "/api/bridge/Vault.encrypt", json = input.toJSON())`) {
		t.Fatalf("expected Kotlin bridge JSON request, got:\n%s", kotlin)
	}
	if !strings.Contains(kotlin, `GSXRequestPolicy(name = "Vault.encrypt", auth = GSXAuthRequirement.Required, retryAttempts = 2)`) {
		t.Fatalf("expected Kotlin bridge policy, got:\n%s", kotlin)
	}
}

func TestValidateActionInvalidatesKnownLoaders(t *testing.T) {
	cfg := &projectConfig{
		DataLoaders: []endpointDeclaration{{Name: "loadProfile", Method: "GET", Path: "/api/profile"}},
		Actions: []endpointDeclaration{{
			Name:        "saveProfile",
			Method:      "POST",
			Path:        "/api/profile",
			Invalidates: []string{"missingLoader"},
		}},
	}

	err := validateProjectDeclarations(cfg, &nir.Module{})
	if err == nil {
		t.Fatalf("expected invalidation validation error")
	}
	if !strings.Contains(err.Error(), `action saveProfile invalidates unknown data loader "missingLoader"`) {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("expected %s to contain %q, got:\n%s", path, want, data)
	}
}
