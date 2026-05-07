package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/odvcencio/gosx-native/emit/android"
	"github.com/odvcencio/gosx-native/emit/ios"
)

type initOptions struct {
	name        string
	module      string
	runtimeRoot string
	force       bool
	dir         string
}

func runInit(args []string) error {
	opts, err := parseInitOptions(args)
	if err != nil {
		return err
	}
	if opts.name == "" {
		opts.name = pascalIdentifier(filepath.Base(opts.dir), "GSXApp")
	} else {
		opts.name = pascalIdentifier(opts.name, "GSXApp")
	}
	if absDir, err := filepath.Abs(opts.dir); err == nil {
		opts.dir = absDir
	}
	if opts.module == "" {
		opts.module = "com.gosxnative." + strings.ToLower(opts.name)
	}
	if err := validateAndroidModule(opts.module); err != nil {
		return err
	}
	if opts.runtimeRoot == "" {
		root, err := repoRoot()
		if err != nil {
			return fmt.Errorf("cannot infer runtime root: %w", err)
		}
		opts.runtimeRoot = root
	}
	if err := prepareInitDir(opts.dir, opts.force); err != nil {
		return err
	}
	return scaffoldProject(opts)
}

func parseInitOptions(args []string) (initOptions, error) {
	var opts initOptions
	fs := flag.NewFlagSet("gsxnative init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.name, "name", "", "native app name")
	fs.StringVar(&opts.module, "module", "", "Android application ID and Kotlin package")
	fs.StringVar(&opts.runtimeRoot, "runtime-root", "", "gosx-native checkout containing runtime/ios and runtime/android")
	fs.BoolVar(&opts.force, "force", false, "write into an existing non-empty directory")
	normalizedArgs, err := normalizeInitArgs(args)
	if err != nil {
		return initOptions{}, err
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return initOptions{}, err
	}
	if fs.NArg() != 1 {
		return initOptions{}, fmt.Errorf("usage: gsxnative init <dir> [--name AppName] [--module com.example.app] [--runtime-root /path/to/gosx-native] [--force]")
	}
	opts.dir = filepath.Clean(fs.Arg(0))
	return opts, nil
}

func normalizeInitArgs(args []string) ([]string, error) {
	var flags []string
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			flags = append(flags, arg)
			if initFlagNeedsValue(arg) && !strings.Contains(arg, "=") {
				i++
				if i >= len(args) {
					return nil, fmt.Errorf("flag needs an argument: %s", arg)
				}
				flags = append(flags, args[i])
			}
			continue
		}
		positional = append(positional, arg)
	}
	return append(flags, positional...), nil
}

func initFlagNeedsValue(arg string) bool {
	name := strings.TrimLeft(arg, "-")
	if cut, _, ok := strings.Cut(name, "="); ok {
		name = cut
	}
	switch name {
	case "name", "module", "runtime-root":
		return true
	default:
		return false
	}
}

func prepareInitDir(dir string, force bool) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dir, 0755)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("init target %s exists and is not a directory", dir)
	}
	if force {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("init target %s is not empty; pass --force to write into it", dir)
	}
	return nil
}

func scaffoldProject(opts initOptions) error {
	iosDir := filepath.Join(opts.dir, "ios")
	androidDir := filepath.Join(opts.dir, "android")
	sourcePath := filepath.Join(opts.dir, "src/app.gsx")
	iosOutput := filepath.Join(iosDir, opts.name, "Generated/App.g.swift")
	androidOutput := filepath.Join(androidDir, "app/src/main/kotlin/generated/App.kt")

	source := initGoSXSource(goPackageName(opts.name))
	if err := writeInitFile(sourcePath, source, opts.force); err != nil {
		return err
	}

	swiftSource, kotlinSource, err := emitInitialNativeSources(sourcePath)
	if err != nil {
		return err
	}

	files := map[string]string{
		filepath.Join(opts.dir, "README.md"):                    initReadme(opts.name),
		filepath.Join(opts.dir, "gosxnative.json"):              initConfig(opts, sourcePath, iosDir, iosOutput, androidDir, androidOutput),
		filepath.Join(iosDir, "project.yml"):                    initXcodeGenProject(opts, iosDir),
		filepath.Join(iosDir, opts.name, opts.name+"App.swift"): initSwiftApp(opts.name),
		iosOutput: swiftSource,
		filepath.Join(androidDir, "settings.gradle.kts"):                                              initAndroidSettings(opts, androidDir),
		filepath.Join(androidDir, "build.gradle.kts"):                                                 initAndroidRootBuild(),
		filepath.Join(androidDir, "app/build.gradle.kts"):                                             initAndroidAppBuild(opts),
		filepath.Join(androidDir, "app/src/main/AndroidManifest.xml"):                                 initAndroidManifest(opts.module),
		filepath.Join(androidDir, "app/src/main/res/values/styles.xml"):                               initAndroidStyles(opts.name),
		filepath.Join(androidDir, "app/src/main/kotlin", packagePath(opts.module), "MainActivity.kt"): initAndroidMainActivity(opts),
		androidOutput: kotlinSource,
	}
	for path, data := range files {
		if err := writeInitFile(path, data, opts.force); err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stdout, "created gosx-native app %s at %s\n", opts.name, opts.dir)
	return nil
}

func emitInitialNativeSources(sourcePath string) (string, string, error) {
	mod, err := compileFile(sourcePath)
	if err != nil {
		return "", "", err
	}
	var swift bytes.Buffer
	if err := ios.Emit(mod, &swift); err != nil {
		return "", "", err
	}
	var kotlin bytes.Buffer
	if err := android.Emit(mod, &kotlin); err != nil {
		return "", "", err
	}
	return swift.String(), kotlin.String(), nil
}

func writeInitFile(path, data string, force bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("refusing to overwrite %s; pass --force", path)
		} else if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return os.WriteFile(path, []byte(data), 0644)
}

func initGoSXSource(pkg string) string {
	return fmt.Sprintf(`package %s

type HomeProps struct {
	Title string
}

//gosx:island
func Home(props HomeProps) Node {
	count := signal.New(0)

	increment := func() {
		count.Set(count.Get() + 1)
	}

	return <vstack>
		<text>{props.Title}</text>
		<text>Native iOS, Android, and Scene3D from one GoSX source.</text>
		<button onClick={increment}>Increment</button>
		<text>{count.Get()}</text>
	</vstack>
}

type SceneDemoProps struct {
	Width  int
	Height int
}

func SceneDemo(props SceneDemoProps) Node {
	return <Scene3D class="native-scene" width={props.Width} height={props.Height}>
		<Camera z={7} fov={64} near={0.1} far={96} />
		<Environment ambientColor="#f4fbff" ambientIntensity={0.2} />
		<DirectionalLight id="sun" color="#fff1d6" intensity={1.2} />
		<Mesh id="hero" kind="box" width={1.8} height={1.2} depth={0.8} color="#8de1ff" />
		<Points id="stars" count={24} size={0.06} />
	</Scene3D>
}
`, pkg)
}

func initConfig(opts initOptions, sourcePath, iosDir, iosOutput, androidDir, androidOutput string) string {
	cfg := projectConfig{
		Name:   opts.name,
		Module: opts.module,
		Source: relSlash(opts.dir, sourcePath),
		IOS: projectTargetConfig{
			Project:      relSlash(opts.dir, iosDir),
			Output:       relSlash(opts.dir, iosOutput),
			XcodeProject: opts.name,
			Scheme:       opts.name,
		},
		Android: projectTargetConfig{
			Project: relSlash(opts.dir, androidDir),
			Output:  relSlash(opts.dir, androidOutput),
		},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(data) + "\n"
}

func initReadme(appName string) string {
	return fmt.Sprintf(`# %s

Generated by gosx-native.

- Build both native shells: `+"`gsxnative build all`"+`
- Build iOS only: `+"`gsxnative build ios`"+`
- Build Android only: `+"`gsxnative build android`"+`
- Source entrypoint: `+"`src/app.gsx`"+`

The build commands discover `+"`gosxnative.json`"+` from this directory or a parent directory.
`, appName)
}

func initXcodeGenProject(opts initOptions, iosDir string) string {
	runtimePath := relSlash(iosDir, filepath.Join(opts.runtimeRoot, "runtime/ios"))
	return fmt.Sprintf(`name: %s
options:
  bundleIdPrefix: %s
  deploymentTarget:
    iOS: "17.0"
  createIntermediateGroups: true

packages:
  GSXNativeKit:
    path: %s

targets:
  %s:
    type: application
    platform: iOS
    sources:
      - %s
    dependencies:
      - package: GSXNativeKit
        product: GSXNativeKit
    settings:
      base:
        GENERATE_INFOPLIST_FILE: YES
        INFOPLIST_KEY_UIApplicationSceneManifest_Generation: YES
        INFOPLIST_KEY_UILaunchScreen_Generation: YES
        TARGETED_DEVICE_FAMILY: "1,2"

schemes:
  %s:
    build:
      targets:
        %s: all
`, opts.name, opts.module, runtimePath, opts.name, opts.name, opts.name, opts.name)
}

func initSwiftApp(appName string) string {
	return fmt.Sprintf(`import SwiftUI
import GSXNativeKit

@main
struct %sApp: SwiftUI.App {
    @StateObject private var router = GSXRouter(initial: GSXRoute("home"))

    var body: some Scene {
        WindowGroup {
            VStack(spacing: 16) {
                HStack(spacing: 12) {
                    Text("Route: \(router.current.name)")
                    Button("Details") {
                        router.push(GSXRoute("details", params: ["id": "1"]))
                    }
                    Button("Home") {
                        router.reset(to: GSXRoute("home"))
                    }
                }
                Home(props: .init(title: "%s"))
                SceneDemo(props: .init(width: 320, height: 180))
            }
            .padding()
        }
    }
}
`, appName, splitWords(appName))
}

func initAndroidSettings(opts initOptions, androidDir string) string {
	runtimePath := relSlash(androidDir, filepath.Join(opts.runtimeRoot, "runtime/android"))
	return fmt.Sprintf(`pluginManagement {
    repositories {
        google()
        mavenCentral()
        gradlePluginPortal()
    }
}

dependencyResolutionManagement {
    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
    repositories {
        google()
        mavenCentral()
    }
}

rootProject.name = "%sAndroid"
include(":app", ":gsx-nativekit")
project(":gsx-nativekit").projectDir = file("%s")
`, opts.name, runtimePath)
}

func initAndroidRootBuild() string {
	return `plugins {
    id("com.android.application") version "9.2.0" apply false
    id("com.android.library") version "9.2.0" apply false
    id("org.jetbrains.kotlin.plugin.compose") version "2.3.21" apply false
}
`
}

func initAndroidAppBuild(opts initOptions) string {
	return fmt.Sprintf(`plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.plugin.compose")
}

android {
    namespace = "%s"
    compileSdk = 36

    defaultConfig {
        applicationId = "%s"
        minSdk = 26
        targetSdk = 36
        versionCode = 1
        versionName = "0.1.0"
    }

    buildFeatures {
        compose = true
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}

dependencies {
    val composeBom = platform("androidx.compose:compose-bom:2026.04.01")

    implementation(composeBom)
    implementation(project(":gsx-nativekit"))
    implementation("androidx.activity:activity-compose:1.13.0")
    implementation("androidx.compose.foundation:foundation")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.runtime:runtime")
    implementation("androidx.compose.ui:ui")
}
`, opts.module, opts.module)
}

func initAndroidManifest(module string) string {
	return fmt.Sprintf(`<manifest xmlns:android="http://schemas.android.com/apk/res/android">
    <application
        android:theme="@style/Theme.GSXNative"
        android:label="@string/app_name"
        android:allowBackup="false"
        android:supportsRtl="true">
        <activity
            android:name="%s.MainActivity"
            android:exported="true">
            <intent-filter>
                <action android:name="android.intent.action.MAIN" />
                <category android:name="android.intent.category.LAUNCHER" />
            </intent-filter>
        </activity>
    </application>
</manifest>
`, module)
}

func initAndroidStyles(appName string) string {
	return fmt.Sprintf(`<resources>
    <string name="app_name">%s</string>
    <style name="Theme.GSXNative" parent="android:style/Theme.Material.Light.NoActionBar" />
</resources>
`, splitWords(appName))
}

func initAndroidMainActivity(opts initOptions) string {
	return fmt.Sprintf(`package %s

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.ui.Modifier
import com.gosx.nativekit.GSXRoute
import com.gosx.nativekit.rememberGSXRouter
import generated.Home
import generated.HomeProps
import generated.SceneDemo
import generated.SceneDemoProps

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme {
                val router = rememberGSXRouter(GSXRoute("home"))
                Column(modifier = Modifier.fillMaxWidth()) {
                    Text(text = "Route: ${router.current.name}")
                    Button(onClick = { router.push(GSXRoute("details", mapOf("id" to "1"))) }) {
                        Text(text = "Details")
                    }
                    Button(onClick = { router.reset(GSXRoute("home")) }) {
                        Text(text = "Home")
                    }
                    Home(HomeProps(title = "%s"))
                    SceneDemo(SceneDemoProps(width = 320, height = 180))
                }
            }
        }
    }
}
`, opts.module, splitWords(opts.name))
}

func validateAndroidModule(module string) error {
	pieces := strings.Split(module, ".")
	if len(pieces) < 2 {
		return fmt.Errorf("module %q must contain at least two dot-separated identifiers", module)
	}
	for _, piece := range pieces {
		if !identifierPattern.MatchString(piece) {
			return fmt.Errorf("module %q contains invalid identifier %q", module, piece)
		}
	}
	return nil
}

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func pascalIdentifier(value, fallback string) string {
	var words []string
	var current strings.Builder
	flush := func() {
		if current.Len() > 0 {
			words = append(words, current.String())
			current.Reset()
		}
	}
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	if len(words) == 0 {
		return fallback
	}
	var out strings.Builder
	for _, word := range words {
		runes := []rune(word)
		if len(runes) == 0 {
			continue
		}
		out.WriteRune(unicode.ToUpper(runes[0]))
		for _, r := range runes[1:] {
			out.WriteRune(r)
		}
	}
	name := out.String()
	if name == "" || !unicode.IsLetter([]rune(name)[0]) {
		return fallback + name
	}
	return name
}

func goPackageName(appName string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(appName) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
		}
	}
	pkg := out.String()
	if pkg == "" || !unicode.IsLetter([]rune(pkg)[0]) {
		return "app"
	}
	return pkg
}

func packagePath(module string) string {
	return filepath.Join(strings.Split(module, ".")...)
}

func relSlash(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func splitWords(value string) string {
	var words []string
	var current strings.Builder
	var previousLower bool
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			previousLower = false
			continue
		}
		if current.Len() > 0 && previousLower && unicode.IsUpper(r) {
			words = append(words, current.String())
			current.Reset()
		}
		current.WriteRune(r)
		previousLower = unicode.IsLower(r)
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return strings.Join(words, " ")
}
