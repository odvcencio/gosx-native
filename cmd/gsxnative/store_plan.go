package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type storePlanOptions struct {
	artifactManifest string
	output           string
	iosProvider      string
	androidProvider  string
	iosTrack         string
	androidTrack     string
	releaseNotesPath string
}

type storePublishPlan struct {
	Version      int               `json:"version"`
	Name         string            `json:"name,omitempty"`
	Module       string            `json:"module,omitempty"`
	Environment  string            `json:"environment,omitempty"`
	ReleaseNotes string            `json:"release_notes,omitempty"`
	Targets      []storePlanTarget `json:"targets"`
}

type storePlanTarget struct {
	Target          string   `json:"target"`
	Provider        string   `json:"provider"`
	Track           string   `json:"track"`
	ArtifactKind    string   `json:"artifact_kind,omitempty"`
	ArtifactPath    string   `json:"artifact_path,omitempty"`
	Published       bool     `json:"published"`
	BundleID        string   `json:"bundle_id,omitempty"`
	PackageName     string   `json:"package_name,omitempty"`
	Environment     string   `json:"environment,omitempty"`
	Release         bool     `json:"release"`
	RequiredSecrets []string `json:"required_secrets,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
}

type storeArtifactCandidate struct {
	Kind      string
	Path      string
	Published bool
}

func runStorePlan(args []string) error {
	opts, err := parseStorePlanOptions(args)
	if err != nil {
		return err
	}
	manifest, err := readStorePlanManifest(opts.artifactManifest)
	if err != nil {
		return err
	}
	plan, err := storePlanFromManifest(manifest, opts)
	if err != nil {
		return err
	}
	return writeStorePlan(opts.output, plan)
}

func parseStorePlanOptions(args []string) (storePlanOptions, error) {
	opts := storePlanOptions{
		iosProvider:     "app-store-connect",
		androidProvider: "google-play",
		iosTrack:        "testflight",
		androidTrack:    "internal",
	}
	fs := flag.NewFlagSet("gsxnative store-plan", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.artifactManifest, "artifact-manifest", "", "build artifact manifest from gsxnative build --artifact-manifest")
	fs.StringVar(&opts.output, "output", "", "write store submission plan JSON to this path; defaults to stdout")
	fs.StringVar(&opts.iosProvider, "ios-provider", opts.iosProvider, "iOS store provider")
	fs.StringVar(&opts.androidProvider, "android-provider", opts.androidProvider, "Android store provider")
	fs.StringVar(&opts.iosTrack, "ios-track", opts.iosTrack, "iOS release track")
	fs.StringVar(&opts.androidTrack, "android-track", opts.androidTrack, "Android release track")
	fs.StringVar(&opts.releaseNotesPath, "release-notes", "", "release notes text file to embed in the plan")
	if err := fs.Parse(args); err != nil {
		return storePlanOptions{}, err
	}
	if fs.NArg() > 1 {
		return storePlanOptions{}, fmt.Errorf("usage: gsxnative store-plan --artifact-manifest build/gsxnative-artifacts.json [flags]")
	}
	if fs.NArg() == 1 {
		if opts.artifactManifest != "" {
			return storePlanOptions{}, fmt.Errorf("artifact manifest specified both as --artifact-manifest and positional argument")
		}
		opts.artifactManifest = fs.Arg(0)
	}
	if strings.TrimSpace(opts.artifactManifest) == "" {
		return storePlanOptions{}, fmt.Errorf("store-plan requires --artifact-manifest")
	}
	if err := validateStoreProvider("ios", opts.iosProvider, "app-store-connect"); err != nil {
		return storePlanOptions{}, err
	}
	if err := validateStoreProvider("android", opts.androidProvider, "google-play"); err != nil {
		return storePlanOptions{}, err
	}
	return opts, nil
}

func validateStoreProvider(targetName, got, want string) error {
	if strings.TrimSpace(got) != want {
		return fmt.Errorf("unsupported %s store provider %q (supported: %s)", targetName, got, want)
	}
	return nil
}

func readStorePlanManifest(path string) (buildArtifactManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return buildArtifactManifest{}, err
	}
	var manifest buildArtifactManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return buildArtifactManifest{}, fmt.Errorf("parse artifact manifest %s: %w", path, err)
	}
	if manifest.Version != 1 {
		return buildArtifactManifest{}, fmt.Errorf("unsupported artifact manifest version %d", manifest.Version)
	}
	return manifest, nil
}

func storePlanFromManifest(manifest buildArtifactManifest, opts storePlanOptions) (storePublishPlan, error) {
	releaseNotes, err := readReleaseNotes(opts.releaseNotesPath)
	if err != nil {
		return storePublishPlan{}, err
	}
	plan := storePublishPlan{
		Version:      1,
		Name:         manifest.Name,
		Module:       manifest.Module,
		Environment:  manifest.Environment,
		ReleaseNotes: releaseNotes,
	}
	for _, targetResult := range manifest.Targets {
		switch targetResult.Target {
		case "ios":
			plan.Targets = append(plan.Targets, iosStorePlanTarget(manifest, targetResult, opts))
		case "android":
			plan.Targets = append(plan.Targets, androidStorePlanTarget(manifest, targetResult, opts))
		}
	}
	if len(plan.Targets) == 0 {
		return storePublishPlan{}, fmt.Errorf("artifact manifest has no ios or android targets")
	}
	return plan, nil
}

func readReleaseNotes(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func iosStorePlanTarget(manifest buildArtifactManifest, result nativeBuildResult, opts storePlanOptions) storePlanTarget {
	artifact, warnings := selectStoreArtifact(result, "ios_export", "ios_archive")
	if artifact.Kind != "ios_export" {
		warnings = append(warnings, "iOS store submission expects an exported app package; pass --export-options-plist to produce an export directory")
	}
	if !result.Release {
		warnings = append(warnings, "iOS target was not built with --release")
	}
	bundleID := ""
	if result.Signing != nil {
		bundleID = result.Signing.BundleID
	}
	if bundleID == "" {
		bundleID = manifest.Module
	}
	return storePlanTarget{
		Target:          "ios",
		Provider:        opts.iosProvider,
		Track:           opts.iosTrack,
		ArtifactKind:    artifact.Kind,
		ArtifactPath:    artifact.Path,
		Published:       artifact.Published,
		BundleID:        bundleID,
		Environment:     firstNonEmpty(result.Environment, manifest.Environment),
		Release:         result.Release,
		RequiredSecrets: []string{"APP_STORE_CONNECT_KEY_ID", "APP_STORE_CONNECT_ISSUER_ID", "APP_STORE_CONNECT_PRIVATE_KEY"},
		Warnings:        warnings,
	}
}

func androidStorePlanTarget(manifest buildArtifactManifest, result nativeBuildResult, opts storePlanOptions) storePlanTarget {
	artifact, warnings := selectStoreArtifact(result, "android_aab", "android_apk")
	if artifact.Kind != "android_aab" {
		warnings = append(warnings, "Google Play release submission expects an Android App Bundle; add --task :app:bundleRelease for release builds")
	}
	if !result.Release {
		warnings = append(warnings, "Android target was not built with --release")
	}
	return storePlanTarget{
		Target:          "android",
		Provider:        opts.androidProvider,
		Track:           opts.androidTrack,
		ArtifactKind:    artifact.Kind,
		ArtifactPath:    artifact.Path,
		Published:       artifact.Published,
		PackageName:     manifest.Module,
		Environment:     firstNonEmpty(result.Environment, manifest.Environment),
		Release:         result.Release,
		RequiredSecrets: []string{"GOOGLE_PLAY_SERVICE_ACCOUNT_JSON"},
		Warnings:        warnings,
	}
}

func selectStoreArtifact(result nativeBuildResult, preferredKind, fallbackKind string) (storeArtifactCandidate, []string) {
	candidates := storeArtifactCandidates(result)
	if artifact, ok := firstStoreArtifact(candidates, preferredKind); ok {
		return artifact, nil
	}
	if artifact, ok := firstStoreArtifact(candidates, fallbackKind); ok {
		return artifact, nil
	}
	return storeArtifactCandidate{}, []string{"no store artifact found in build manifest"}
}

func storeArtifactCandidates(result nativeBuildResult) []storeArtifactCandidate {
	var candidates []storeArtifactCandidate
	for _, artifact := range result.PublishedArtifacts {
		candidates = append(candidates, storeArtifactCandidate{
			Kind:      artifact.Kind,
			Path:      artifact.Path,
			Published: true,
		})
	}
	for _, artifact := range result.ExpectedArtifacts {
		candidates = append(candidates, storeArtifactCandidate{
			Kind: artifact.Kind,
			Path: artifact.Path,
		})
	}
	return candidates
}

func firstStoreArtifact(candidates []storeArtifactCandidate, kind string) (storeArtifactCandidate, bool) {
	for _, candidate := range candidates {
		if candidate.Kind == kind && strings.TrimSpace(candidate.Path) != "" {
			return candidate, true
		}
	}
	return storeArtifactCandidate{}, false
}

func writeStorePlan(path string, plan storePublishPlan) error {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(plan); err != nil {
		return err
	}
	if strings.TrimSpace(path) == "" {
		_, err := os.Stdout.Write(buf.Bytes())
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}
