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
	assertFileContains(t, filepath.Join(dir, "src/app.gsx"), "//gosx:data name=loadGreeting method=GET path=/api/greeting output=message:string ttl=30s retry=2 auth=optional")
	assertFileContains(t, filepath.Join(dir, "ios/project.yml"), "name: SampleApp")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/SampleAppApp.swift"), "GSXRouter(initial: GSXRoutes.home)")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/App.g.swift"), "public struct Home: GSXComponent")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public enum GSXRoutes")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public convenience init(baseURL: String, defaultHeaders: [String: String] = [:]) throws")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public convenience init(baseURL: String, defaultHeaders: [String: String] = [:], tokenStore: any GSXTokenStore) throws")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public enum GSXDataLoaders")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public struct GSXGeneratedDataLoadGreetingResponse")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public func loadGreeting() async throws -> GSXGeneratedDataLoadGreetingResponse")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), `GSXRequestPolicy(name: "loadGreeting", cacheTTLSeconds: 30, auth: GSXAuthRequirement.optional, retryAttempts: 2)`)
	assertFileContains(t, filepath.Join(dir, "android/settings.gradle.kts"), "project(\":gsx-nativekit\").projectDir")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/com/example/sample/MainActivity.kt"), "rememberGSXRouter(GSXRoutes.home)")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/App.kt"), "fun Home(props: HomeProps)")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "object GSXRoutes")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "import com.gosx.nativekit.GSXTokenStore")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "constructor(baseURL: String, defaultHeaders: Map<String, String> = emptyMap())")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "constructor(baseURL: String, defaultHeaders: Map<String, String> = emptyMap(), tokenStore: GSXTokenStore)")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "object GSXDataLoaders")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "data class GSXGeneratedDataLoadGreetingResponse")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), "suspend fun loadGreeting(): GSXGeneratedDataLoadGreetingResponse")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/GSXDeclarations.kt"), `GSXRequestPolicy(name = "loadGreeting", cacheTTLSeconds = 30, auth = GSXAuthRequirement.Optional, retryAttempts = 2)`)

	var cfg projectConfig
	data, err := os.ReadFile(filepath.Join(dir, "gosxnative.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Source != "src/app.gsx" || cfg.IOS.SupportOutput == "" || cfg.Android.SupportOutput == "" ||
		len(cfg.Routes) != 0 || len(cfg.DataLoaders) != 0 || len(cfg.Actions) != 0 {
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
			strings.HasPrefix(strings.TrimSpace(line), "//gosx:action") {
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
	cfg.Routes = []routeDeclaration{{Name: "legacyHome", Path: "/", Component: "Home"}}
	cfg.DataLoaders = []endpointDeclaration{{Name: "legacyLoad", Method: "GET", Path: "/api/legacy"}}
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
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/GSXDeclarations.g.swift"), "public func legacyLoad() async throws -> GSXResponse")
}

func TestParseSourceDeclarations(t *testing.T) {
	decls, err := parseSourceDeclarations([]byte(`
//gosx:route name=home path=/ component=Home
//gosx:route name=details path=/details/:id component=Home params=id:string,count:int
//gosx:data name=loadGreeting method=GET path=/api/greeting params=locale:string output=message:string ttl=45s retry=2 auth=optional
//gosx:action name=submitGreeting method=POST path=/api/greeting input=message:string output=message:string invalidates=loadGreeting optimistic=echo auth=required retry=3
//gosx:native swift
`))
	if err != nil {
		t.Fatalf("parse source declarations: %v", err)
	}
	if len(decls.Routes) != 2 || len(decls.DataLoaders) != 1 || len(decls.Actions) != 1 {
		t.Fatalf("unexpected declarations: %#v", decls)
	}
	if got := decls.Routes[1].Params; len(got) != 2 || got[0].Name != "id" || got[1].Type != "int" {
		t.Fatalf("unexpected params: %#v", got)
	}
	if loader := decls.DataLoaders[0]; len(loader.Params) != 1 || len(loader.Output) != 1 ||
		loader.CacheTTLSeconds != 45 || loader.RetryAttempts != 2 || loader.Auth != "optional" {
		t.Fatalf("unexpected loader declaration: %#v", loader)
	}
	if action := decls.Actions[0]; len(action.Input) != 1 || len(action.Output) != 1 ||
		len(action.Invalidates) != 1 || action.Invalidates[0] != "loadGreeting" ||
		action.Optimistic != "echo" || action.Auth != "required" || action.RetryAttempts != 3 {
		t.Fatalf("unexpected action declaration: %#v", action)
	}
}

func TestEmitTypedEndpointDeclarations(t *testing.T) {
	cfg := &projectConfig{
		DataLoaders: []endpointDeclaration{{
			Name:            "loadProfile",
			Method:          "GET",
			Path:            "/api/users/:id/profile",
			Params:          []paramDeclaration{{Name: "id", Type: "string"}, {Name: "includePosts", Type: "bool"}},
			Output:          []paramDeclaration{{Name: "displayName", Type: "string"}, {Name: "postCount", Type: "int"}},
			CacheTTLSeconds: 120,
			Auth:            "required",
			RetryAttempts:   3,
		}},
		Actions: []endpointDeclaration{{
			Name:        "saveProfile",
			Method:      "PATCH",
			Path:        "/api/users/:id/profile",
			Params:      []paramDeclaration{{Name: "id", Type: "string"}},
			Input:       []paramDeclaration{{Name: "displayName", Type: "string"}},
			Output:      []paramDeclaration{{Name: "displayName", Type: "string"}},
			Invalidates: []string{"loadProfile"},
			Optimistic:  "profileEcho",
			Auth:        "required",
		}},
	}

	swift := string(emitSwiftDeclarations(cfg))
	if !strings.Contains(swift, `GSXRequest.resolvedPath("/api/users/:id/profile", params: ["id": id, "includePosts": String(includePosts)])`) {
		t.Fatalf("expected Swift path parameter lowering, got:\n%s", swift)
	}
	if !strings.Contains(swift, "public func loadProfile(id: String, includePosts: Bool) async throws -> GSXGeneratedDataLoadProfileResponse") {
		t.Fatalf("expected Swift typed loader signature, got:\n%s", swift)
	}
	if !strings.Contains(swift, `GSXRequestPolicy(name: "saveProfile", invalidates: ["loadProfile"], optimistic: "profileEcho", auth: GSXAuthRequirement.required)`) {
		t.Fatalf("expected Swift action policy metadata, got:\n%s", swift)
	}

	kotlin := string(emitKotlinDeclarations(cfg))
	if !strings.Contains(kotlin, `GSXRequest.resolvedPath("/api/users/:id/profile", params = mapOf("id" to id, "includePosts" to includePosts.toString()))`) {
		t.Fatalf("expected Kotlin path parameter lowering, got:\n%s", kotlin)
	}
	if !strings.Contains(kotlin, "suspend fun saveProfile(id: String, displayName: String): GSXGeneratedActionSaveProfileResponse") {
		t.Fatalf("expected Kotlin typed action signature, got:\n%s", kotlin)
	}
	if !strings.Contains(kotlin, `GSXRequestPolicy(name = "saveProfile", invalidates = listOf("loadProfile"), optimistic = "profileEcho", auth = GSXAuthRequirement.Required)`) {
		t.Fatalf("expected Kotlin action policy metadata, got:\n%s", kotlin)
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
