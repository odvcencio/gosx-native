package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
)

type commandOutputRunner interface {
	Output(ctx context.Context, dir, name string, args ...string) ([]byte, error)
}

type osCommandOutputRunner struct{}

func (osCommandOutputRunner) Output(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message != "" {
			return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, message)
		}
		return nil, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}

var devicesRunner commandOutputRunner = osCommandOutputRunner{}

type devicesOptions struct {
	target string
	json   bool
	adb    string
	xcrun  string
}

type nativeDevice struct {
	Target    string `json:"target"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	State     string `json:"state"`
	Runtime   string `json:"runtime,omitempty"`
	Available bool   `json:"available"`
}

func runDevices(args []string) error {
	return runDevicesWithContext(context.Background(), args)
}

func runDevicesWithContext(ctx context.Context, args []string) error {
	opts, err := parseDevicesOptions(args)
	if err != nil {
		return err
	}
	devices, err := listNativeDevices(ctx, opts)
	if err != nil {
		return err
	}
	if opts.json {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(devices)
	}
	for _, device := range devices {
		fmt.Fprintf(os.Stdout, "%s\t%s\t%s\t%s", device.Target, device.ID, device.Name, device.State)
		if device.Runtime != "" {
			fmt.Fprintf(os.Stdout, "\t%s", device.Runtime)
		}
		fmt.Fprintln(os.Stdout)
	}
	return nil
}

func parseDevicesOptions(args []string) (devicesOptions, error) {
	opts := devicesOptions{target: "all", adb: "adb", xcrun: "xcrun"}
	targetName := ""
	flagArgs := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if targetName == "" && isDevicesTarget(arg) {
			targetName = strings.ToLower(strings.TrimSpace(arg))
			continue
		}
		flagArgs = append(flagArgs, arg)
		if deviceFlagNeedsValue(arg) && !strings.Contains(arg, "=") && i+1 < len(args) {
			i++
			flagArgs = append(flagArgs, args[i])
		}
	}
	fs := flag.NewFlagSet("gsxnative devices", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&opts.json, "json", false, "emit devices as JSON")
	fs.StringVar(&opts.adb, "adb", opts.adb, "adb executable")
	fs.StringVar(&opts.xcrun, "xcrun", opts.xcrun, "xcrun executable")
	if err := fs.Parse(flagArgs); err != nil {
		return devicesOptions{}, err
	}
	if fs.NArg() > 0 {
		if targetName == "" && fs.NArg() == 1 {
			return devicesOptions{}, fmt.Errorf("unknown devices target: %s (supported: ios, android, all)", fs.Arg(0))
		}
		return devicesOptions{}, fmt.Errorf("usage: gsxnative devices [ios|android|all] [--json]")
	}
	if targetName != "" {
		opts.target = targetName
	}
	return opts, nil
}

func isDevicesTarget(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ios", "android", "all":
		return true
	default:
		return false
	}
}

func deviceFlagNeedsValue(arg string) bool {
	switch arg {
	case "--adb", "--xcrun":
		return true
	default:
		return false
	}
}

func listNativeDevices(ctx context.Context, opts devicesOptions) ([]nativeDevice, error) {
	var devices []nativeDevice
	if opts.target == "android" || opts.target == "all" {
		androidDevices, err := listAndroidDevices(ctx, opts.adb)
		if err != nil {
			return nil, err
		}
		devices = append(devices, androidDevices...)
	}
	if opts.target == "ios" || opts.target == "all" {
		iosDevices, err := listIOSDevices(ctx, opts.xcrun)
		if err != nil {
			return nil, err
		}
		devices = append(devices, iosDevices...)
	}
	sort.SliceStable(devices, func(i, j int) bool {
		if devices[i].Target != devices[j].Target {
			return devices[i].Target < devices[j].Target
		}
		if devices[i].Name != devices[j].Name {
			return devices[i].Name < devices[j].Name
		}
		return devices[i].ID < devices[j].ID
	})
	return devices, nil
}

func listAndroidDevices(ctx context.Context, adb string) ([]nativeDevice, error) {
	out, err := devicesRunner.Output(ctx, "", firstNonEmpty(adb, "adb"), "devices", "-l")
	if err != nil {
		return nil, err
	}
	return parseAndroidDevices(out), nil
}

func parseAndroidDevices(out []byte) []nativeDevice {
	var devices []nativeDevice
	for _, raw := range strings.Split(string(out), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "List of devices") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		attrs := androidDeviceAttrs(fields[2:])
		name := firstNonEmpty(attrs["model"], attrs["device"], fields[0])
		devices = append(devices, nativeDevice{
			Target:    "android",
			ID:        fields[0],
			Name:      name,
			State:     fields[1],
			Runtime:   attrs["product"],
			Available: fields[1] == "device",
		})
	}
	return devices
}

func androidDeviceAttrs(fields []string) map[string]string {
	attrs := make(map[string]string)
	for _, field := range fields {
		key, value, ok := strings.Cut(field, ":")
		if !ok || key == "" {
			continue
		}
		attrs[key] = value
	}
	return attrs
}

func listIOSDevices(ctx context.Context, xcrun string) ([]nativeDevice, error) {
	out, err := devicesRunner.Output(ctx, "", firstNonEmpty(xcrun, "xcrun"), "simctl", "list", "devices", "available", "--json")
	if err != nil {
		return nil, err
	}
	devices, err := parseIOSDevices(out)
	if err != nil {
		return nil, err
	}
	return devices, nil
}

type simctlDeviceList struct {
	Devices map[string][]simctlDevice `json:"devices"`
}

type simctlDevice struct {
	UDID        string `json:"udid"`
	Name        string `json:"name"`
	State       string `json:"state"`
	IsAvailable bool   `json:"isAvailable"`
}

func parseIOSDevices(out []byte) ([]nativeDevice, error) {
	decoder := json.NewDecoder(bytes.NewReader(out))
	var payload simctlDeviceList
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("parse simctl devices JSON: %w", err)
	}
	var runtimes []string
	for runtimeName := range payload.Devices {
		runtimes = append(runtimes, runtimeName)
	}
	sort.Strings(runtimes)
	var devices []nativeDevice
	for _, runtimeName := range runtimes {
		for _, device := range payload.Devices[runtimeName] {
			devices = append(devices, nativeDevice{
				Target:    "ios",
				ID:        device.UDID,
				Name:      device.Name,
				State:     device.State,
				Runtime:   runtimeName,
				Available: device.IsAvailable,
			})
		}
	}
	return devices, nil
}
