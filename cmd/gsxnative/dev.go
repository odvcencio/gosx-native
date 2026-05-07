package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type devOptions struct {
	target    string
	once      bool
	interval  time.Duration
	buildArgs []string
}

type watchedFile struct {
	size    int64
	modTime time.Time
}

func runDev(args []string) error {
	return runDevWithContext(context.Background(), args)
}

func runDevWithContext(ctx context.Context, args []string) error {
	opts, err := parseDevOptions(args)
	if err != nil {
		return err
	}
	if err := runDevBuild(ctx, opts); err != nil {
		return err
	}
	if opts.once {
		return nil
	}
	watchRoot, err := devWatchRoot(opts)
	if err != nil {
		return err
	}
	snapshot, err := snapshotDevFiles(watchRoot)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "watching %s for %s codegen\n", watchRoot, opts.target)
	ticker := time.NewTicker(opts.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			next, changed, err := changedDevFiles(watchRoot, snapshot)
			if err != nil {
				fmt.Fprintf(os.Stderr, "dev watch: %v\n", err)
				continue
			}
			if !changed {
				continue
			}
			if err := runDevBuild(ctx, opts); err != nil {
				fmt.Fprintf(os.Stderr, "dev build: %v\n", err)
			}
			snapshot = next
		}
	}
}

func parseDevOptions(args []string) (devOptions, error) {
	opts := devOptions{target: "all", interval: 500 * time.Millisecond}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if i == 0 && isDevTarget(arg) {
			opts.target = strings.ToLower(arg)
			continue
		}
		switch {
		case arg == "--once":
			opts.once = true
		case arg == "--target":
			i++
			if i >= len(args) {
				return devOptions{}, fmt.Errorf("flag needs an argument: --target")
			}
			if !isDevTarget(args[i]) {
				return devOptions{}, fmt.Errorf("unknown dev target: %s (supported: ios, android, all)", args[i])
			}
			opts.target = strings.ToLower(args[i])
		case strings.HasPrefix(arg, "--target="):
			value := strings.TrimPrefix(arg, "--target=")
			if !isDevTarget(value) {
				return devOptions{}, fmt.Errorf("unknown dev target: %s (supported: ios, android, all)", value)
			}
			opts.target = strings.ToLower(value)
		case arg == "--interval":
			i++
			if i >= len(args) {
				return devOptions{}, fmt.Errorf("flag needs an argument: --interval")
			}
			interval, err := time.ParseDuration(args[i])
			if err != nil || interval <= 0 {
				return devOptions{}, fmt.Errorf("invalid dev interval %q", args[i])
			}
			opts.interval = interval
		case strings.HasPrefix(arg, "--interval="):
			value := strings.TrimPrefix(arg, "--interval=")
			interval, err := time.ParseDuration(value)
			if err != nil || interval <= 0 {
				return devOptions{}, fmt.Errorf("invalid dev interval %q", value)
			}
			opts.interval = interval
		default:
			opts.buildArgs = append(opts.buildArgs, arg)
		}
	}
	return opts, nil
}

func isDevTarget(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ios", "android", "all":
		return true
	default:
		return false
	}
}

func runDevBuild(ctx context.Context, opts devOptions) error {
	args := append([]string{opts.target}, opts.buildArgs...)
	if !hasBuildFlag(args, "codegen-only") {
		args = append(args, "--codegen-only")
	}
	return runBuildWithContext(ctx, args)
}

func hasBuildFlag(args []string, name string) bool {
	short := "-" + name
	long := "--" + name
	for _, arg := range args {
		if arg == short || arg == long || strings.HasPrefix(arg, short+"=") || strings.HasPrefix(arg, long+"=") {
			return true
		}
	}
	return false
}

func devWatchRoot(opts devOptions) (string, error) {
	buildOpts, err := parseBuildOptions(opts.target, opts.buildArgs)
	if err != nil {
		return "", err
	}
	if err := applyBuildProjectConfig(&buildOpts, opts.target); err != nil {
		return "", err
	}
	if buildOpts.projectBaseDir != "" {
		return buildOpts.projectBaseDir, nil
	}
	if buildOpts.source == "" {
		root, err := repoRoot()
		if err != nil {
			return "", fmt.Errorf("missing source; pass --source or run from a directory with gosxnative.json")
		}
		buildOpts.source = repoDefault(root, "testdata/corpus/swift/counter.swift.gsx")
	}
	info, err := os.Stat(buildOpts.source)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return buildOpts.source, nil
	}
	return filepath.Dir(buildOpts.source), nil
}

func changedDevFiles(root string, previous map[string]watchedFile) (map[string]watchedFile, bool, error) {
	next, err := snapshotDevFiles(root)
	if err != nil {
		return nil, false, err
	}
	if len(previous) != len(next) {
		return next, true, nil
	}
	for path, state := range next {
		old, ok := previous[path]
		if !ok || old.size != state.size || !old.modTime.Equal(state.modTime) {
			return next, true, nil
		}
	}
	return next, false, nil
}

func snapshotDevFiles(root string) (map[string]watchedFile, error) {
	files := make(map[string]watchedFile)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".gradle", "DerivedData", "build":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		if !isDevSourceFile(path) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		files[path] = watchedFile{size: info.Size(), modTime: info.ModTime()}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func isDevSourceFile(path string) bool {
	name := filepath.Base(path)
	if name == "gosxnative.json" || name == "capabilities.json" {
		return true
	}
	return strings.HasSuffix(name, ".gsx") ||
		strings.HasSuffix(name, ".swift.gsx") ||
		strings.HasSuffix(name, ".kt.gsx")
}
