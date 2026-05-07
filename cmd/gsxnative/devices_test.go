package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type fakeDevicesRunner struct {
	outputs  map[string][]byte
	commands []recordedCommand
}

func (f *fakeDevicesRunner) Output(_ context.Context, dir, name string, args ...string) ([]byte, error) {
	f.commands = append(f.commands, recordedCommand{
		dir:  dir,
		name: name,
		args: append([]string(nil), args...),
	})
	key := name + " " + strings.Join(args, " ")
	return append([]byte(nil), f.outputs[key]...), nil
}

func useFakeDevicesRunner(t *testing.T, outputs map[string][]byte) *fakeDevicesRunner {
	t.Helper()
	previous := devicesRunner
	fake := &fakeDevicesRunner{outputs: outputs}
	devicesRunner = fake
	t.Cleanup(func() {
		devicesRunner = previous
	})
	return fake
}

func TestParseAndroidDevices(t *testing.T) {
	devices := parseAndroidDevices([]byte(`List of devices attached
emulator-5554 device product:sdk_gphone64_x86_64 model:sdk_gphone64_x86_64 device:emu64x transport_id:1
R58M123 unauthorized usb:1-1 product:foo model:Galaxy_S25 device:s25

`))
	if len(devices) != 2 {
		t.Fatalf("expected two Android devices, got %#v", devices)
	}
	if devices[0].Target != "android" || devices[0].ID != "emulator-5554" ||
		devices[0].Name != "sdk_gphone64_x86_64" || devices[0].Runtime != "sdk_gphone64_x86_64" ||
		!devices[0].Available {
		t.Fatalf("unexpected emulator device: %#v", devices[0])
	}
	if devices[1].State != "unauthorized" || devices[1].Available {
		t.Fatalf("unexpected unauthorized device: %#v", devices[1])
	}
}

func TestParseIOSDevices(t *testing.T) {
	devices, err := parseIOSDevices([]byte(`{
  "devices": {
    "com.apple.CoreSimulator.SimRuntime.iOS-26-0": [
      {"udid": "A-B-C", "name": "iPhone 17", "state": "Shutdown", "isAvailable": true}
    ],
    "com.apple.CoreSimulator.SimRuntime.iOS-25-0": [
      {"udid": "D-E-F", "name": "iPad Pro", "state": "Booted", "isAvailable": false}
    ]
  }
}`))
	if err != nil {
		t.Fatalf("parse ios devices: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected two iOS devices, got %#v", devices)
	}
	if devices[0].Target != "ios" || devices[0].ID != "D-E-F" ||
		devices[0].Name != "iPad Pro" || devices[0].Runtime != "com.apple.CoreSimulator.SimRuntime.iOS-25-0" ||
		devices[0].Available {
		t.Fatalf("unexpected first sorted iOS device: %#v", devices[0])
	}
	if devices[1].ID != "A-B-C" || !devices[1].Available {
		t.Fatalf("unexpected second sorted iOS device: %#v", devices[1])
	}
}

func TestRunDevicesJSONListsAllTargets(t *testing.T) {
	fake := useFakeDevicesRunner(t, map[string][]byte{
		"adb devices -l": []byte(`List of devices attached
emulator-5554 device product:sdk model:Pixel_9 device:emu
`),
		"xcrun simctl list devices available --json": []byte(`{
  "devices": {
    "com.apple.CoreSimulator.SimRuntime.iOS-26-0": [
      {"udid": "IOS-1", "name": "iPhone 17", "state": "Shutdown", "isAvailable": true}
    ]
  }
}`),
	})
	var stdout bytes.Buffer
	previousStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writePipe
	err = runDevices([]string{"--json"})
	_ = writePipe.Close()
	os.Stdout = previousStdout
	if err != nil {
		t.Fatalf("devices: %v", err)
	}
	if _, err := stdout.ReadFrom(readPipe); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	var devices []nativeDevice
	if err := json.Unmarshal(stdout.Bytes(), &devices); err != nil {
		t.Fatalf("parse devices JSON: %v\n%s", err, stdout.String())
	}
	if len(devices) != 2 || devices[0].Target != "android" || devices[1].Target != "ios" {
		t.Fatalf("unexpected device list: %#v", devices)
	}
	if len(fake.commands) != 2 || fake.commands[0].name != "adb" || fake.commands[1].name != "xcrun" {
		t.Fatalf("unexpected commands: %#v", fake.commands)
	}
}

func TestParseDevicesOptionsAcceptsTargetBeforeFlags(t *testing.T) {
	opts, err := parseDevicesOptions([]string{"android", "--json", "--adb", "custom-adb"})
	if err != nil {
		t.Fatalf("parse devices options: %v", err)
	}
	if opts.target != "android" || !opts.json || opts.adb != "custom-adb" {
		t.Fatalf("unexpected devices options: %#v", opts)
	}
}

func TestRunDevicesRejectsUnknownTarget(t *testing.T) {
	err := runDevices([]string{"watchos"})
	if err == nil || !strings.Contains(err.Error(), "unknown devices target") {
		t.Fatalf("expected target validation error, got %v", err)
	}
}
