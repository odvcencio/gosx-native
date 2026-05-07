package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStorePlanWritesSubmissionTargetsFromManifest(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "gsxnative-artifacts.json")
	outputPath := filepath.Join(dir, "store-plan.json")
	notesPath := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(notesPath, []byte("Ship it.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	manifest := buildArtifactManifest{
		Version:     1,
		Name:        "Shop",
		Module:      "com.example.shop",
		Environment: "production",
		Targets: []nativeBuildResult{
			{
				Target:      "ios",
				Release:     true,
				Environment: "production",
				ExpectedArtifacts: []expectedArtifact{
					{Kind: "ios_archive", Path: filepath.Join(dir, "ios/Shop.xcarchive")},
				},
				PublishedArtifacts: []publishedArtifact{
					{Kind: "ios_export", Source: filepath.Join(dir, "ios/export"), Path: filepath.Join(dir, "published/ios/export")},
				},
				Signing: &signingSummary{BundleID: "com.example.shop.ios"},
			},
			{
				Target:      "android",
				Release:     true,
				Environment: "production",
				PublishedArtifacts: []publishedArtifact{
					{Kind: "android_aab", Source: filepath.Join(dir, "android/app-release.aab"), Path: filepath.Join(dir, "published/android/app-release.aab")},
				},
			},
		},
	}
	writeTestManifest(t, manifestPath, manifest)

	err := runStorePlan([]string{
		"--artifact-manifest", manifestPath,
		"--output", outputPath,
		"--ios-track", "testflight-beta",
		"--android-track", "production",
		"--release-notes", notesPath,
	})
	if err != nil {
		t.Fatalf("store-plan: %v", err)
	}
	plan := readTestStorePlan(t, outputPath)
	if plan.Version != 1 || plan.Name != "Shop" || plan.Module != "com.example.shop" || plan.ReleaseNotes != "Ship it." {
		t.Fatalf("unexpected plan header: %#v", plan)
	}
	if len(plan.Targets) != 2 {
		t.Fatalf("expected two store targets, got %#v", plan.Targets)
	}
	ios := plan.Targets[0]
	if ios.Target != "ios" || ios.Provider != "app-store-connect" || ios.Track != "testflight-beta" ||
		ios.ArtifactKind != "ios_export" || ios.ArtifactPath != filepath.Join(dir, "published/ios/export") ||
		!ios.Published || ios.BundleID != "com.example.shop.ios" || len(ios.Warnings) != 0 {
		t.Fatalf("unexpected iOS plan target: %#v", ios)
	}
	if !containsString(ios.RequiredSecrets, "APP_STORE_CONNECT_PRIVATE_KEY") {
		t.Fatalf("expected App Store Connect secrets, got %#v", ios.RequiredSecrets)
	}
	android := plan.Targets[1]
	if android.Target != "android" || android.Provider != "google-play" || android.Track != "production" ||
		android.ArtifactKind != "android_aab" || android.ArtifactPath != filepath.Join(dir, "published/android/app-release.aab") ||
		!android.Published || android.PackageName != "com.example.shop" || len(android.Warnings) != 0 {
		t.Fatalf("unexpected Android plan target: %#v", android)
	}
	if !containsString(android.RequiredSecrets, "GOOGLE_PLAY_SERVICE_ACCOUNT_JSON") {
		t.Fatalf("expected Google Play secret, got %#v", android.RequiredSecrets)
	}
}

func TestStorePlanWarnsForNonStoreReadyArtifacts(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "gsxnative-artifacts.json")
	outputPath := filepath.Join(dir, "store-plan.json")
	manifest := buildArtifactManifest{
		Version: 1,
		Module:  "com.example.preview",
		Targets: []nativeBuildResult{
			{
				Target:  "ios",
				Release: true,
				ExpectedArtifacts: []expectedArtifact{
					{Kind: "ios_archive", Path: filepath.Join(dir, "ios/Preview.xcarchive")},
				},
			},
			{
				Target:  "android",
				Release: false,
				ExpectedArtifacts: []expectedArtifact{
					{Kind: "android_apk", Path: filepath.Join(dir, "android/app-debug.apk")},
				},
			},
		},
	}
	writeTestManifest(t, manifestPath, manifest)

	if err := runStorePlan([]string{"--artifact-manifest", manifestPath, "--output", outputPath}); err != nil {
		t.Fatalf("store-plan: %v", err)
	}
	plan := readTestStorePlan(t, outputPath)
	if len(plan.Targets) != 2 {
		t.Fatalf("expected two store targets, got %#v", plan.Targets)
	}
	if got := strings.Join(plan.Targets[0].Warnings, "\n"); !strings.Contains(got, "exported app package") {
		t.Fatalf("expected iOS export warning, got %#v", plan.Targets[0].Warnings)
	}
	if got := strings.Join(plan.Targets[1].Warnings, "\n"); !strings.Contains(got, "Android App Bundle") || !strings.Contains(got, "--release") {
		t.Fatalf("expected Android AAB and release warnings, got %#v", plan.Targets[1].Warnings)
	}
}

func TestStorePlanRejectsUnsupportedProvider(t *testing.T) {
	_, err := parseStorePlanOptions([]string{"--artifact-manifest", "build/manifest.json", "--android-provider", "other"})
	if err == nil || !strings.Contains(err.Error(), "unsupported android store provider") {
		t.Fatalf("expected unsupported provider error, got %v", err)
	}
}

func writeTestManifest(t *testing.T, path string, manifest buildArtifactManifest) {
	t.Helper()
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

func readTestStorePlan(t *testing.T, path string) storePublishPlan {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read store plan: %v", err)
	}
	var plan storePublishPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatalf("parse store plan: %v", err)
	}
	return plan
}
