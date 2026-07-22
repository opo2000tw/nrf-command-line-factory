package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

const oneJLink = `{"type":"info","data":{"devices":[{"serialNumber":"000802009570","traits":{"jlink":true}}]}}` + "\n"

func TestParseJLinkSerials(t *testing.T) {
	input := strings.Join([]string{
		`{"type":"task_begin","data":{}}`,
		`{"type":"info","data":{"devices":[{"serialNumber":"UART","traits":{"jlink":false}},{"serialNumber":"0002","traits":{"jlink":true}},{"serialNumber":"0001","traits":{"jlink":true}}]}}`,
		`{"type":"task_end","data":{"data":{"devices":[{"serialNumber":"0002","traits":{"jlink":true}}]}}}`,
	}, "\n")

	serials, err := parseJLinkSerials([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"0001", "0002"}
	if !reflect.DeepEqual(serials, want) {
		t.Fatalf("serials = %v, want %v", serials, want)
	}
}

func TestFlashEndpointSuccess(t *testing.T) {
	var programArgs []string
	runner := func(_ context.Context, _ string, args []string, output io.Writer) error {
		switch args[1] {
		case "list":
			_, _ = io.WriteString(output, oneJLink)
			return nil
		case "program":
			programArgs = append([]string(nil), args...)
			_, _ = io.WriteString(output, "Erasing ranges\r\nProgramming\nVerified\n")
			return nil
		default:
			return errors.New("unexpected command")
		}
	}
	application := newApp("nrfutil", runner, true, "ready")

	request := flashRequest(t, "R", "release build.hex", ":00000001FF\n")
	recorder := httptest.NewRecorder()
	application.handleFlash(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	events := decodeEvents(t, recorder.Body.Bytes())
	last := events[len(events)-1]
	if !last.Done || !last.Success || last.Level != "success" {
		t.Fatalf("last event = %+v", last)
	}
	if last.State == nil || last.State.RightCount != 1 || last.State.LeftCount != 0 {
		t.Fatalf("state = %+v", last.State)
	}

	wantOptions := "chip_erase_mode=ERASE_RANGES_TOUCHED_BY_FIRMWARE,verify=VERIFY_READ,reset=RESET_SYSTEM"
	if !containsPair(programArgs, "--serial-number", "000802009570") {
		t.Fatalf("missing serial number in %v", programArgs)
	}
	if !containsPair(programArgs, "--options", wantOptions) {
		t.Fatalf("missing safe program options in %v", programArgs)
	}
	firmwareIndex := indexOf(programArgs, "--firmware")
	if firmwareIndex < 0 || firmwareIndex+1 >= len(programArgs) || !strings.HasSuffix(programArgs[firmwareIndex+1], ".hex") {
		t.Fatalf("missing temporary hex path in %v", programArgs)
	}
}

func TestFlashFailureDoesNotIncrement(t *testing.T) {
	runner := func(_ context.Context, _ string, args []string, output io.Writer) error {
		if args[1] == "list" {
			_, _ = io.WriteString(output, oneJLink)
			return nil
		}
		_, _ = io.WriteString(output, "LOW_VOLTAGE\n")
		return errors.New("exit status 1")
	}
	application := newApp("nrfutil", runner, true, "ready")

	recorder := httptest.NewRecorder()
	application.handleFlash(recorder, flashRequest(t, "L", "app.hex", ":00000001FF\n"))

	events := decodeEvents(t, recorder.Body.Bytes())
	last := events[len(events)-1]
	if !last.Done || last.Success || last.Level != "error" {
		t.Fatalf("last event = %+v", last)
	}
	if last.State == nil || last.State.LeftCount != 0 || last.State.Busy {
		t.Fatalf("state = %+v", last.State)
	}
}

func TestFlashRejectsNonHex(t *testing.T) {
	called := false
	application := newApp("nrfutil", func(context.Context, string, []string, io.Writer) error {
		called = true
		return nil
	}, true, "ready")

	recorder := httptest.NewRecorder()
	application.handleFlash(recorder, flashRequest(t, "L", "firmware.zip", "data"))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if called {
		t.Fatal("nrfutil runner was called for invalid firmware")
	}
}

func TestResetCounts(t *testing.T) {
	application := newApp("nrfutil", nil, true, "ready")
	if !application.beginFlash() {
		t.Fatal("could not start flash")
	}
	application.finishFlash("L", true)

	request := httptest.NewRequest(http.MethodPost, "/api/reset", nil)
	recorder := httptest.NewRecorder()
	application.handleReset(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}

	var state snapshot
	if err := json.Unmarshal(recorder.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if state.LeftCount != 0 || state.RightCount != 0 {
		t.Fatalf("state = %+v", state)
	}
}

func TestProbeJLink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("J-Link probe fixture is unix-only")
	}
	pathDir := t.TempDir()
	installRoot := t.TempDir()
	t.Setenv("PATH", pathDir)
	prev := jLinkSearchGlobs
	jLinkSearchGlobs = []string{filepath.Join(installRoot, "JLink*", "JLinkExe")}
	t.Cleanup(func() { jLinkSearchGlobs = prev })

	// nothing on PATH and nothing installed
	if ok, msg := probeJLink("nonexistent-nrfutil"); ok || msg == "" {
		t.Fatalf("expected J-Link missing, got ok=%v msg=%q", ok, msg)
	}

	// detected via PATH
	pathFake := filepath.Join(pathDir, "JLinkExe")
	if err := os.WriteFile(pathFake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if ok, _ := probeJLink("nonexistent-nrfutil"); !ok {
		t.Fatal("expected J-Link detected via JLinkExe on PATH")
	}
	if err := os.Remove(pathFake); err != nil {
		t.Fatal(err)
	}

	// detected via the standard install dir without any PATH entry
	instDir := filepath.Join(installRoot, "JLink_V794")
	if err := os.MkdirAll(instDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instDir, "JLinkExe"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if ok, _ := probeJLink("nonexistent-nrfutil"); !ok {
		t.Fatal("expected J-Link detected via the standard install dir without PATH")
	}
}

func TestFindBundledNRFUtil(t *testing.T) {
	root := t.TempDir()
	thirdDir := filepath.Join(root, "3rd")
	if err := os.MkdirAll(thirdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, ok := findBundledNRFUtil([]string{thirdDir}); ok {
		t.Fatal("expected no bundled nrfutil before it exists")
	}
	bin := filepath.Join(thirdDir, nrfutilBinaryName())
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, ok := findBundledNRFUtil([]string{thirdDir})
	if !ok || got != bin {
		t.Fatalf("findBundledNRFUtil = %q, %v; want %q, true", got, ok, bin)
	}
}

func flashRequest(t *testing.T, side, filename, content string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("side", side); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("firmware", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(part, content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/flash", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func decodeEvents(t *testing.T, output []byte) []apiEvent {
	t.Helper()
	var events []apiEvent
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		var event apiEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatalf("decode event %q: %v", scanner.Text(), err)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 {
		t.Fatal("no events returned")
	}
	return events
}

func containsPair(values []string, key, value string) bool {
	index := indexOf(values, key)
	return index >= 0 && index+1 < len(values) && values[index+1] == value
}

func indexOf(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}

func TestInstallDeviceStreamsAndRefreshes(t *testing.T) {
	var gotArgs []string
	runner := func(_ context.Context, _ string, args []string, output io.Writer) error {
		gotArgs = append([]string(nil), args...)
		_, _ = io.WriteString(output, "Installing device\nDone\n")
		return nil
	}
	application := newApp("nrfutil", runner, false, "缺 device command")
	application.probe = func(string) (bool, string) { return true, "nrfutil ok · device ok" }

	recorder := httptest.NewRecorder()
	application.handleInstallDevice(recorder, httptest.NewRequest(http.MethodPost, "/api/install-device", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !reflect.DeepEqual(gotArgs, []string{"install", "device"}) {
		t.Fatalf("args = %v", gotArgs)
	}
	events := decodeEvents(t, recorder.Body.Bytes())
	last := events[len(events)-1]
	if !last.Done || !last.Success || last.Level != "success" {
		t.Fatalf("last event = %+v", last)
	}
	if last.State == nil || !last.State.ToolReady || last.State.Busy {
		t.Fatalf("state = %+v", last.State)
	}
}

func TestInstallDeviceFailureReportsError(t *testing.T) {
	runner := func(_ context.Context, _ string, _ []string, output io.Writer) error {
		_, _ = io.WriteString(output, "network error\n")
		return errors.New("exit status 1")
	}
	application := newApp("nrfutil", runner, false, "缺 device command")
	application.probe = func(string) (bool, string) { return false, "仍缺 device command" }

	recorder := httptest.NewRecorder()
	application.handleInstallDevice(recorder, httptest.NewRequest(http.MethodPost, "/api/install-device", nil))

	events := decodeEvents(t, recorder.Body.Bytes())
	last := events[len(events)-1]
	if !last.Done || last.Success || last.Level != "error" {
		t.Fatalf("last event = %+v", last)
	}
	if last.State == nil || last.State.ToolReady || last.State.Busy {
		t.Fatalf("state = %+v", last.State)
	}
}

func TestToolUploadSetsCommand(t *testing.T) {
	application := newApp("nrfutil", nil, false, "找不到 nrfutil")
	var probed string
	application.probe = func(path string) (bool, string) { probed = path; return true, "工具就緒" }

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("tool", "nrfutil.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(part, "fake-binary"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/tool", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	application.handleTool(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var state snapshot
	if err := json.Unmarshal(recorder.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if !state.ToolReady || !strings.HasSuffix(state.Command, "nrf-factory-nrfutil.exe") {
		t.Fatalf("state = %+v", state)
	}
	if probed != state.Command {
		t.Fatalf("probed %q != command %q", probed, state.Command)
	}
	_ = os.Remove(state.Command)
}
