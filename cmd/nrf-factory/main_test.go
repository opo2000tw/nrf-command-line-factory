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
	"time"
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

func TestFlashProgrammingContinuesAfterRequestCancellation(t *testing.T) {
	programStarted := make(chan struct{})
	inspectContext := make(chan struct{})
	waitingForRelease := make(chan struct{})
	releaseProgram := make(chan struct{})
	programResult := make(chan string, 1)
	released := false
	defer func() {
		if !released {
			close(releaseProgram)
		}
	}()

	runner := func(ctx context.Context, _ string, args []string, output io.Writer) error {
		switch args[1] {
		case "list":
			_, _ = io.WriteString(output, oneJLink)
			return nil
		case "program":
			close(programStarted)
			<-inspectContext
			select {
			case <-ctx.Done():
				programResult <- "context canceled"
				return ctx.Err()
			default:
			}
			close(waitingForRelease)
			select {
			case <-ctx.Done():
				programResult <- "context canceled"
				return ctx.Err()
			case <-releaseProgram:
				programResult <- "released"
				return nil
			}
		default:
			return errors.New("unexpected command")
		}
	}
	application := newApp("nrfutil", runner, true, "ready")

	requestContext, cancelRequest := context.WithCancel(context.Background())
	request := flashRequest(t, "L", "app.hex", ":00000001FF\n").WithContext(requestContext)
	recorder := httptest.NewRecorder()
	handlerDone := make(chan struct{})
	go func() {
		application.handleFlash(recorder, request)
		close(handlerDone)
	}()

	select {
	case <-programStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("program stage did not start")
	}

	cancelRequest()
	close(inspectContext)
	select {
	case result := <-programResult:
		t.Fatalf("program stopped before release: %s", result)
	case <-waitingForRelease:
	case <-time.After(2 * time.Second):
		t.Fatal("program did not inspect its context")
	}

	close(releaseProgram)
	released = true
	select {
	case result := <-programResult:
		if result != "released" {
			t.Fatalf("program result = %q, want released", result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("program did not finish after release")
	}
	select {
	case <-handlerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("flash handler did not return")
	}

	events := decodeEvents(t, recorder.Body.Bytes())
	last := events[len(events)-1]
	if !last.Done || !last.Success || last.State == nil || last.State.LeftCount != 1 {
		t.Fatalf("last event = %+v", last)
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

func TestNormalizeJLinkVersion(t *testing.T) {
	cases := map[string]string{
		"":             "",
		"JLink_V9.60":  "V9.60",
		"JLink_V9.24a": "V9.24a",
		"V9.60":        "V9.60",
		"9.60":         "V9.60",
		// Compact pack ids (same digits as folder names) must expand, not stay "V960".
		"JLink_V960":  "V9.60",
		"JLink_V924a": "V9.24a",
		"V960":        "V9.60",
		"V924a":       "V9.24a",
		"960":         "V9.60",
	}
	for in, want := range cases {
		if got := normalizeJLinkVersion(in); got != want {
			t.Fatalf("normalizeJLinkVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestVersionFromJLinkDir(t *testing.T) {
	// Paths need not exist: we only parse the parent directory name (EvalSymlinks
	// is a no-op when the path is missing).
	root := string(filepath.Separator)
	cases := []struct {
		path string
		want string
	}{
		{filepath.Join(root, "Applications", "SEGGER", "JLink_V960", "JLinkExe"), "V9.60"},
		{filepath.Join(root, "opt", "SEGGER", "JLink_V924a", "JLink.exe"), "V9.24a"},
		{filepath.Join(root, "opt", "SEGGER", "JLink_V948", "JLinkExe"), "V9.48"},
		{filepath.Join(root, "opt", "SEGGER", "JLink_V794e", "JLinkExe"), "V7.94e"},
		{filepath.Join(root, "opt", "SEGGER", "JLink_V794", "JLinkExe"), "V7.94"},
		// Plain "JLink" pointer that does not resolve via symlink → no version in the name.
		// (A real /Applications/SEGGER/JLink symlink is followed by EvalSymlinks.)
		{filepath.Join(t.TempDir(), "SEGGER", "JLink", "JLinkExe"), ""},
		{filepath.Join(t.TempDir(), "bin", "JLinkExe"), ""},
	}
	for _, tc := range cases {
		if got := versionFromJLinkDir(tc.path); got != tc.want {
			t.Fatalf("versionFromJLinkDir(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestJLinkVersionGreater(t *testing.T) {
	// V960 must rank above V924a (not string / path order).
	if !jLinkVersionGreater("V9.60", "V9.24a") {
		t.Fatal("expected V9.60 > V9.24a")
	}
	if jLinkVersionGreater("V9.24a", "V9.60") {
		t.Fatal("expected V9.24a < V9.60")
	}
	if !jLinkVersionGreater("V9.24a", "V9.24") {
		t.Fatal("expected hotfix letter V9.24a > V9.24")
	}
	if !jLinkVersionGreater("V9.48", "V9.24a") {
		t.Fatal("expected V9.48 > V9.24a")
	}
	if !jLinkVersionGreater("V9.60", "") {
		t.Fatal("expected any version > empty")
	}
}

func TestFindJLinkInstallPicksNewest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses unix executable names")
	}
	root := t.TempDir()
	// Create older and newer packs; filesystem order must not win.
	for _, ver := range []string{"JLink_V960", "JLink_V924a", "JLink_V948"} {
		dir := filepath.Join(root, ver)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "JLinkExe"), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	prev := jLinkSearchGlobs
	jLinkSearchGlobs = []string{filepath.Join(root, "JLink_*", "JLinkExe")}
	t.Cleanup(func() { jLinkSearchGlobs = prev })

	got, ok := findJLinkInstall()
	if !ok {
		t.Fatal("expected an install")
	}
	if versionFromJLinkDir(got) != "V9.60" {
		t.Fatalf("picked %q (version %q), want newest JLink_V960", got, versionFromJLinkDir(got))
	}
}

func TestJLinkStatusLabel(t *testing.T) {
	if got := jLinkStatusLabel("V9.60"); got != "SEGGER J-Link V9.60 OK" {
		t.Fatalf("version label = %q", got)
	}
}

func TestJLinkInstallHintUsesTestedVersion(t *testing.T) {
	dir := t.TempDir()
	// Fake nrfutil that only implements `device --version --json` for testedJLinkVersion.
	script := filepath.Join(dir, "nrfutil")
	body := `#!/bin/sh
if [ "$1" = "device" ] && [ "$2" = "--version" ] && [ "$3" = "--json" ]; then
  printf '%s\n' '{"dependencies":[{"expectedVersion":{"version":"JLink_V9.24a"},"name":"JlinkARM","version":"JLink_V9.24a"}]}'
  exit 0
fi
exit 1
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	hint := jLinkInstallHint(script)
	if !strings.Contains(hint, "V9.24a") {
		t.Fatalf("hint %q should include suggested V9.24a", hint)
	}
	msg := jLinkMissingMessage(script)
	if !strings.Contains(msg, "找不到") || !strings.Contains(msg, "V9.24a") {
		t.Fatalf("missing message should name suggested version: %q", msg)
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
	ok, msg := probeJLink("nonexistent-nrfutil")
	if ok || msg == "" {
		t.Fatalf("expected J-Link missing, got ok=%v msg=%q", ok, msg)
	}
	if !strings.Contains(msg, "找不到") {
		t.Fatalf("missing message = %q", msg)
	}

	// Binary on PATH but no version readable → fail and ask to reinstall.
	pathFake := filepath.Join(pathDir, "JLinkExe")
	if err := os.WriteFile(pathFake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ok, msg = probeJLink("nonexistent-nrfutil")
	if ok {
		t.Fatal("expected failure when version cannot be read")
	}
	if !strings.Contains(msg, "重新安裝") {
		t.Fatalf("unknown-version message = %q", msg)
	}
	if err := os.Remove(pathFake); err != nil {
		t.Fatal(err)
	}

	// Standard install dir with versioned folder name → OK with that version.
	instDir := filepath.Join(installRoot, "JLink_V794")
	if err := os.MkdirAll(instDir, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(instDir, "JLinkExe")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ok, ver := probeJLink("nonexistent-nrfutil")
	if !ok {
		t.Fatalf("expected J-Link OK via versioned install dir, got %q", ver)
	}
	if ver != "V7.94" {
		t.Fatalf("version from install dir = %q, want V7.94", ver)
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
	application.probe = func(string) toolStatus {
		return toolStatus{ready: true, deviceReady: true, message: "nrfutil ok · device ok"}
	}

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
	if last.State == nil || !last.State.ToolReady || !last.State.DeviceReady || last.State.Busy {
		t.Fatalf("state = %+v", last.State)
	}
}

func TestInstallDeviceFailureReportsError(t *testing.T) {
	runner := func(_ context.Context, _ string, _ []string, output io.Writer) error {
		_, _ = io.WriteString(output, "network error\n")
		return errors.New("exit status 1")
	}
	application := newApp("nrfutil", runner, false, "缺 device command")
	application.probe = func(string) toolStatus { return toolStatus{message: "仍缺 device command"} }

	recorder := httptest.NewRecorder()
	application.handleInstallDevice(recorder, httptest.NewRequest(http.MethodPost, "/api/install-device", nil))

	events := decodeEvents(t, recorder.Body.Bytes())
	last := events[len(events)-1]
	if !last.Done || last.Success || last.Level != "error" {
		t.Fatalf("last event = %+v", last)
	}
	if last.State == nil || last.State.ToolReady || last.State.DeviceReady || last.State.Busy {
		t.Fatalf("state = %+v", last.State)
	}
}

func TestToolUploadSetsCommand(t *testing.T) {
	application := newApp("nrfutil", nil, false, "找不到 nrfutil")
	var probed string
	application.probe = func(path string) toolStatus {
		probed = path
		return toolStatus{ready: true, deviceReady: true, message: "工具就緒"}
	}

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
