package main

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	maxUploadBytes  = 64 << 20
	toolUploadBytes = 128 << 20
	flashTimeout    = 5 * time.Minute
	installTimeout  = 10 * time.Minute
)

// Build metadata, injected via
// -ldflags "-X main.version=... -X main.commit=... -X main.date=...".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

//go:embed web/index.html
var webFiles embed.FS

type commandRunner func(context.Context, string, []string, io.Writer) error

type app struct {
	mu          sync.Mutex
	leftCount   int
	rightCount  int
	busy        bool
	command     string
	run         commandRunner
	probe       func(string) toolStatus
	toolReady   bool
	deviceReady bool
	toolMessage string
}

// toolStatus is what a preflight probe reports. deviceReady tracks the nrfutil
// device command specifically (independent of the J-Link check), so the UI can
// grey out "安裝 device 命令" once it is already installed.
type toolStatus struct {
	ready       bool
	deviceReady bool
	message     string
}

type snapshot struct {
	LeftCount   int    `json:"leftCount"`
	RightCount  int    `json:"rightCount"`
	Busy        bool   `json:"busy"`
	ToolReady   bool   `json:"toolReady"`
	DeviceReady bool   `json:"deviceReady"`
	ToolMessage string `json:"toolMessage"`
	Command     string `json:"command"`
	Version     string `json:"version"`
}

type apiEvent struct {
	Level   string    `json:"level"`
	Message string    `json:"message"`
	Done    bool      `json:"done,omitempty"`
	Success bool      `json:"success,omitempty"`
	State   *snapshot `json:"state,omitempty"`
}

func newApp(command string, run commandRunner, ready bool, message string) *app {
	return &app{
		command:     command,
		run:         run,
		probe:       preflight,
		toolReady:   ready,
		toolMessage: message,
	}
}

func (a *app) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleIndex)
	mux.HandleFunc("/api/state", a.handleState)
	mux.HandleFunc("/api/reset", a.handleReset)
	mux.HandleFunc("/api/flash", a.handleFlash)
	mux.HandleFunc("/api/tool", a.handleTool)
	mux.HandleFunc("/api/install-device", a.handleInstallDevice)
	mux.HandleFunc("/api/detect", a.handleDetect)
	return mux
}

func (a *app) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	page, err := webFiles.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "UI asset unavailable", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self' data:")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(page)
}

func (a *app) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, a.currentSnapshot())
}

func (a *app) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	a.mu.Lock()
	if a.busy {
		a.mu.Unlock()
		writeError(w, http.StatusConflict, "燒錄進行中，無法清除計數")
		return
	}
	a.leftCount = 0
	a.rightCount = 0
	a.mu.Unlock()

	writeJSON(w, http.StatusOK, a.currentSnapshot())
}

func (a *app) handleFlash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.isBusy() {
		writeError(w, http.StatusConflict, "已有燒錄工作進行中")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "韌體上傳失敗或檔案過大")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	side := strings.ToUpper(strings.TrimSpace(r.FormValue("side")))
	if side != "L" && side != "R" {
		writeError(w, http.StatusBadRequest, "請選擇 L 或 R")
		return
	}

	firmware, header, err := r.FormFile("firmware")
	if err != nil {
		writeError(w, http.StatusBadRequest, "請選擇韌體檔案")
		return
	}
	defer firmware.Close()

	displayName := safeFilename(header)
	if !strings.EqualFold(filepath.Ext(displayName), ".hex") {
		writeError(w, http.StatusBadRequest, "第一版只支援 .hex 韌體")
		return
	}

	temp, err := os.CreateTemp("", "nrf-factory-*.hex")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "無法建立暫存韌體")
		return
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)

	written, copyErr := io.Copy(temp, firmware)
	closeErr := temp.Close()
	if copyErr != nil || closeErr != nil {
		writeError(w, http.StatusInternalServerError, "無法儲存暫存韌體")
		return
	}
	if written == 0 {
		writeError(w, http.StatusBadRequest, "韌體檔案是空的")
		return
	}

	if !a.beginFlash() {
		writeError(w, http.StatusConflict, "已有燒錄工作進行中")
		return
	}

	stream := newEventWriter(w)
	_ = stream.event("info", fmt.Sprintf("%s 側：準備燒錄 %s", side, displayName), false, false, nil)

	// Hardware writes must finish once started. a.busy prevents re-entry,
	// while the timeout still places an upper bound on the operation.
	ctx, cancel := context.WithTimeout(context.Background(), flashTimeout)
	defer cancel()
	err = a.program(ctx, stream, tempPath)
	_ = stream.flushPending()

	success := err == nil
	state := a.finishFlash(side, success)
	if success {
		_ = stream.event("success", fmt.Sprintf("%s 側燒錄成功", side), true, true, &state)
		return
	}

	message := "燒錄失敗：" + err.Error()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		message = "燒錄逾時，已停止 nrfutil"
	}
	_ = stream.event("error", message, true, false, &state)
}

// handleTool accepts an uploaded nrfutil executable, stores it, points the app
// at it, and re-runs preflight so the status reflects the new tool.
func (a *app) handleTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !a.beginFlash() {
		writeError(w, http.StatusConflict, "工作進行中，無法設定工具")
		return
	}
	defer a.clearBusy()

	r.Body = http.MaxBytesReader(w, r.Body, toolUploadBytes)
	if err := r.ParseMultipartForm(toolUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "上傳失敗或檔案過大")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	file, header, err := r.FormFile("tool")
	if err != nil {
		writeError(w, http.StatusBadRequest, "請選擇 nrfutil 執行檔")
		return
	}
	defer file.Close()

	path, err := saveToolBinary(file, header)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "無法儲存 nrfutil 執行檔")
		return
	}

	status := a.probe(path)
	a.mu.Lock()
	a.command = path
	a.toolReady = status.ready
	a.deviceReady = status.deviceReady
	a.toolMessage = status.message
	a.mu.Unlock()

	writeJSON(w, http.StatusOK, a.currentSnapshot())
}

// handleInstallDevice streams `nrfutil install device` to the client and
// re-runs preflight when it finishes.
func (a *app) handleInstallDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !a.beginFlash() {
		writeError(w, http.StatusConflict, "已有工作進行中")
		return
	}

	command := a.currentCommand()
	stream := newEventWriter(w)
	_ = stream.event("info", "安裝 nrfutil device 命令…", false, false, nil)

	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
	defer cancel()
	runErr := a.run(ctx, command, []string{"install", "device"}, stream)
	_ = stream.flushPending()

	status := a.probe(command)
	a.mu.Lock()
	a.toolReady = status.ready
	a.deviceReady = status.deviceReady
	a.toolMessage = status.message
	a.busy = false
	a.mu.Unlock()
	state := a.currentSnapshot()

	if runErr == nil {
		_ = stream.event("success", "device 命令安裝完成", true, true, &state)
		return
	}
	message := "安裝失敗：" + runErr.Error()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		message = "安裝逾時"
	}
	_ = stream.event("error", message, true, false, &state)
}

// detectCheck is one row of the detect panel: whether that thing is present
// plus a human-readable message.
type detectCheck struct {
	Found   bool   `json:"found"`
	Message string `json:"message"`
}

// detectResult reports the two-stage detection separately: the J-Link debug
// probe (via `nrfutil device list`) and the target MCU behind it (via
// `nrfutil device device-info`, which actually reads the chip).
type detectResult struct {
	JLink   detectCheck `json:"jlink"`
	MCU     detectCheck `json:"mcu"`
	Serials []string    `json:"serials,omitempty"`
}

// handleDetect runs two nrfutil commands (same on macOS and Windows; the
// resolved binary handles the .exe): `device list` to find the J-Link probe,
// then `device device-info` to confirm the MCU itself is connected and powered.
// J-Link and MCU presence are reported as independent checks.
func (a *app) handleDetect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !a.beginFlash() {
		writeError(w, http.StatusConflict, "工作進行中，無法偵測")
		return
	}
	defer a.clearBusy()

	command := a.currentCommand()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stage 1: the J-Link debug probe.
	var listOut bytes.Buffer
	listArgs := []string{"device", "list", "--traits", "jlink", "--json"}
	if err := a.run(ctx, command, listArgs, &listOut); err != nil {
		detail := strings.TrimSpace(listOut.String())
		message := "無法列舉 J-Link，請確認已安裝 nrfutil device 命令"
		if detail != "" {
			message = "無法列舉 J-Link：" + detail
		}
		writeJSON(w, http.StatusOK, detectResult{
			JLink: detectCheck{Message: message},
			MCU:   detectCheck{Message: "無法偵測 MCU（需先確認 J-Link）"},
		})
		return
	}

	serials, err := parseJLinkSerials(listOut.Bytes())
	if err != nil {
		writeJSON(w, http.StatusOK, detectResult{
			JLink: detectCheck{Message: "無法解析 J-Link 清單：" + err.Error()},
			MCU:   detectCheck{Message: "無法偵測 MCU"},
		})
		return
	}

	switch len(serials) {
	case 0:
		writeJSON(w, http.StatusOK, detectResult{
			JLink: detectCheck{Message: "未偵測到 J-Link，請確認 USB 連接"},
			MCU:   detectCheck{Message: "無法偵測 MCU（需先接上 J-Link）"},
		})
		return
	case 1:
		// exactly one probe; continue to the MCU check
	default:
		writeJSON(w, http.StatusOK, detectResult{
			Serials: serials,
			JLink:   detectCheck{Message: "偵測到多個 J-Link，燒錄前請只保留一個：" + strings.Join(serials, ", ")},
			MCU:     detectCheck{Message: "無法偵測 MCU（請只保留一個 J-Link）"},
		})
		return
	}

	serial := serials[0]
	result := detectResult{
		Serials: serials,
		JLink:   detectCheck{Found: true, Message: "J-Link 已連接：" + serial},
	}

	// Stage 2: the target MCU behind the probe.
	var infoOut bytes.Buffer
	infoArgs := []string{"device", "device-info", "--serial-number", serial, "--json"}
	if err := a.run(ctx, command, infoArgs, &infoOut); err != nil {
		result.MCU = detectCheck{Message: "讀不到 MCU（晶片未連接或未供電）"}
	} else {
		result.MCU = detectCheck{Found: true, Message: "MCU 已連接"}
	}
	writeJSON(w, http.StatusOK, result)
}

func saveToolBinary(file io.Reader, header *multipart.FileHeader) (string, error) {
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if runtime.GOOS == "windows" && ext == "" {
		ext = ".exe"
	}
	path := filepath.Join(os.TempDir(), "nrf-factory-nrfutil"+ext)
	out, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, file); err != nil {
		_ = out.Close()
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	_ = os.Chmod(path, 0o755)
	return path, nil
}

func (a *app) program(ctx context.Context, output io.Writer, firmwarePath string) error {
	command := a.currentCommand()
	var discovery bytes.Buffer
	listArgs := []string{"device", "list", "--traits", "jlink", "--json"}
	if err := a.run(ctx, command, listArgs, &discovery); err != nil {
		detail := strings.TrimSpace(discovery.String())
		if detail == "" {
			return fmt.Errorf("無法列舉 J-Link: %w", err)
		}
		return fmt.Errorf("無法列舉 J-Link: %s", detail)
	}

	serials, err := parseJLinkSerials(discovery.Bytes())
	if err != nil {
		return err
	}
	if len(serials) == 0 {
		return errors.New("找不到 J-Link")
	}
	if len(serials) > 1 {
		return fmt.Errorf("找到多個 J-Link，請只保留一個：%s", strings.Join(serials, ", "))
	}

	serial := serials[0]
	_, _ = fmt.Fprintf(output, "J-Link %s\n", serial)
	args := []string{
		"device", "program",
		"--serial-number", serial,
		"--firmware", firmwarePath,
		"--options", "chip_erase_mode=ERASE_RANGES_TOUCHED_BY_FIRMWARE,verify=VERIFY_READ,reset=RESET_SYSTEM",
	}
	if err := a.run(ctx, command, args, output); err != nil {
		return fmt.Errorf("nrfutil program: %w", err)
	}
	return nil
}

func (a *app) beginFlash() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.busy {
		return false
	}
	a.busy = true
	return true
}

func (a *app) finishFlash(side string, success bool) snapshot {
	a.mu.Lock()
	if success {
		if side == "L" {
			a.leftCount++
		} else {
			a.rightCount++
		}
	}
	a.busy = false
	a.mu.Unlock()
	return a.currentSnapshot()
}

func (a *app) isBusy() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.busy
}

func (a *app) clearBusy() {
	a.mu.Lock()
	a.busy = false
	a.mu.Unlock()
}

func (a *app) currentCommand() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.command
}

func (a *app) currentSnapshot() snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	return snapshot{
		LeftCount:   a.leftCount,
		RightCount:  a.rightCount,
		Busy:        a.busy,
		ToolReady:   a.toolReady,
		DeviceReady: a.deviceReady,
		ToolMessage: a.toolMessage,
		Command:     a.command,
		Version:     version,
	}
}

type nrfDevice struct {
	SerialNumber string `json:"serialNumber"`
	Traits       struct {
		JLink bool `json:"jlink"`
	} `json:"traits"`
}

type deviceSet struct {
	Devices []nrfDevice `json:"devices"`
}

type nrfMessage struct {
	Data struct {
		Devices []nrfDevice `json:"devices"`
		Data    *deviceSet  `json:"data"`
	} `json:"data"`
}

func parseJLinkSerials(output []byte) ([]string, error) {
	serials := make(map[string]struct{})
	validMessages := 0
	scanner := bufio.NewScanner(bytes.NewReader(output))
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var message nrfMessage
		if err := json.Unmarshal(line, &message); err != nil {
			continue
		}
		validMessages++
		collectJLinks(serials, message.Data.Devices)
		if message.Data.Data != nil {
			collectJLinks(serials, message.Data.Data.Devices)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("無法讀取 nrfutil JSON: %w", err)
	}
	if validMessages == 0 {
		return nil, errors.New("nrfutil 未回傳可解析的 JSON Lines")
	}

	result := make([]string, 0, len(serials))
	for serial := range serials {
		result = append(result, serial)
	}
	sort.Strings(result)
	return result, nil
}

func collectJLinks(serials map[string]struct{}, devices []nrfDevice) {
	for _, device := range devices {
		if device.Traits.JLink && device.SerialNumber != "" {
			serials[device.SerialNumber] = struct{}{}
		}
	}
}

func safeFilename(header *multipart.FileHeader) string {
	name := strings.ReplaceAll(header.Filename, "\\", "/")
	name = filepath.Base(name)
	if name == "." || name == "/" || name == "" {
		return "firmware.hex"
	}
	return name
}

type eventWriter struct {
	mu      sync.Mutex
	writer  http.ResponseWriter
	encoder *json.Encoder
	flusher http.Flusher
	pending string
}

func newEventWriter(w http.ResponseWriter) *eventWriter {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	return &eventWriter{writer: w, encoder: json.NewEncoder(w), flusher: flusher}
}

func (w *eventWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	normalized := strings.ReplaceAll(string(p), "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	w.pending += normalized
	parts := strings.Split(w.pending, "\n")
	w.pending = parts[len(parts)-1]
	for _, line := range parts[:len(parts)-1] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if err := w.emitLocked(apiEvent{Level: "info", Message: line}); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

func (w *eventWriter) flushPending() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if strings.TrimSpace(w.pending) == "" {
		w.pending = ""
		return nil
	}
	message := w.pending
	w.pending = ""
	return w.emitLocked(apiEvent{Level: "info", Message: message})
}

func (w *eventWriter) event(level, message string, done, success bool, state *snapshot) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.emitLocked(apiEvent{Level: level, Message: message, Done: done, Success: success, State: state})
}

func (w *eventWriter) emitLocked(event apiEvent) error {
	if err := w.encoder.Encode(event); err != nil {
		return err
	}
	if w.flusher != nil {
		w.flusher.Flush()
	}
	return nil
}

func execCommand(ctx context.Context, name string, args []string, output io.Writer) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = output
	cmd.Stderr = output
	return cmd.Run()
}

func probeNRFUtil(command string) toolStatus {
	path, err := exec.LookPath(command)
	if err != nil {
		return toolStatus{message: "找不到 nrfutil，請先安裝 Nordic nRF Util 與 device command"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	coreVersion, err := firstOutputLine(ctx, path, "--version")
	if err != nil {
		return toolStatus{message: "nrfutil 無法執行"}
	}
	deviceVersion, err := firstOutputLine(ctx, path, "device", "--version")
	if err != nil {
		return toolStatus{message: "缺少 nrfutil device command；請執行 nrfutil install device"}
	}
	return toolStatus{ready: true, deviceReady: true, message: coreVersion + " · " + deviceVersion}
}

// jLinkSearchGlobs lists glob patterns for the SEGGER J-Link executable in its
// standard install directories. It is a variable so tests can point it at a
// fixture. The SEGGER installer usually does NOT add J-Link to PATH, so we look
// here as well instead of forcing the operator to edit PATH.
var jLinkSearchGlobs = defaultJLinkGlobs()

func defaultJLinkGlobs() []string {
	switch runtime.GOOS {
	case "windows":
		var globs []string
		for _, base := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")} {
			if base == "" {
				continue
			}
			globs = append(globs,
				filepath.Join(base, "SEGGER", "JLink", "JLink.exe"),
				filepath.Join(base, "SEGGER", "JLink_*", "JLink.exe"),
			)
		}
		return globs
	case "darwin":
		return []string{
			"/Applications/SEGGER/JLink/JLinkExe",
			"/Applications/SEGGER/JLink_*/JLinkExe",
		}
	default:
		return []string{
			"/opt/SEGGER/JLink/JLinkExe",
			"/opt/SEGGER/JLink_*/JLinkExe",
		}
	}
}

// findJLinkInstall returns a SEGGER J-Link executable from the standard install
// locations. Preference order:
//  1. the current "JLink" install pointer (symlink / junction to the active pack)
//  2. the highest JLink_V* folder (e.g. V960 > V924a > V794), not filesystem order
func findJLinkInstall() (string, bool) {
	var best string
	var bestVer string
	for _, pattern := range jLinkSearchGlobs {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil || info.IsDir() {
				continue
			}
			dir := filepath.Base(filepath.Dir(match))
			// Current install name used by SEGGER on every OS ("JLink" → active pack).
			if strings.EqualFold(dir, "JLink") {
				return match, true
			}
			ver := versionFromJLinkDir(match)
			if best == "" || jLinkVersionGreater(ver, bestVer) {
				best, bestVer = match, ver
			}
		}
	}
	if best == "" {
		return "", false
	}
	return best, true
}

var (
	jLinkExpectedRe  = regexp.MustCompile(`"expectedVersion"\s*:\s*\{\s*"version"\s*:\s*"([^"]+)"`)
	jLinkInstalledRe = regexp.MustCompile(`"name"\s*:\s*"JlinkARM"\s*,\s*"version"\s*:\s*"([^"]+)"`)
	jLinkDetectedRe  = regexp.MustCompile(`Detected SEGGER J-Link version:\s*(\S+)`)
	jLinkCommanderRe = regexp.MustCompile(`J-Link Commander\s+(V[\d.]+[a-zA-Z]?)`)
	// SEGGER pack folders: JLink_V960, JLink_V924a, JLink_V794e — digits are
	// major+minor with a fixed 2-digit minor, optional single letter hotfix.
	jLinkDirVerRe = regexp.MustCompile(`(?i)JLink_V(\d{3,})([a-zA-Z]?)$`)
	jLinkSemVerRe = regexp.MustCompile(`(?i)^V?(\d+)\.(\d+)([a-zA-Z]?)$`)
)

// testedJLinkVersion asks nrfutil which J-Link version the device command was
// tested against, so a missing-J-Link message can name the right one. Returns
// "" when it cannot be determined.
func testedJLinkVersion(command string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, command, "device", "--version", "--json").CombinedOutput()
	if err != nil {
		return ""
	}
	if m := jLinkExpectedRe.FindSubmatch(out); m != nil {
		return string(m[1])
	}
	return ""
}

// resolveJLink returns the absolute path of a J-Link CLI if installed (PATH or
// standard install dir). The SEGGER installer usually does not add PATH entries.
func resolveJLink() (string, bool) {
	for _, name := range []string{"JLinkExe", "JLink"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, true
		}
	}
	return findJLinkInstall()
}

// normalizeJLinkVersion turns tags like JLink_V9.60 / V9.60 / 9.60 into "V9.60".
// Compact folder forms (JLink_V960 / JLink_V924a) are handled by versionFromJLinkDir,
// not here — stripping only the "JLink_" prefix would leave the misleading "V960".
func normalizeJLinkVersion(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = strings.TrimPrefix(raw, "JLink_")
	raw = strings.TrimPrefix(raw, "jlink_")
	if raw == "" {
		return ""
	}
	// Compact pack id without dots (960, 924a) → expand like folder names.
	if m := jLinkDirVerRe.FindStringSubmatch("JLink_V" + strings.TrimPrefix(strings.TrimPrefix(raw, "V"), "v")); m != nil && !strings.Contains(raw, ".") {
		digits, suffix := m[1], m[2]
		if len(digits) >= 3 {
			return "V" + digits[:len(digits)-2] + "." + digits[len(digits)-2:] + suffix
		}
	}
	if raw[0] == 'V' || raw[0] == 'v' {
		return "V" + raw[1:]
	}
	return "V" + raw
}

// versionFromJLinkDir parses SEGGER install folder names.
//
//	JLink_V960  → V9.60
//	JLink_V924a → V9.24a
//	JLink_V794e → V7.94e
//
// Rule (SEGGER pack naming): after "JLink_V", all but the last two digits are the
// major version; the last two digits are the minor; an optional trailing letter is
// a hotfix suffix. Dotted names (JLink_V9.60) are not used for folder names on
// disk — those come from nrfutil / commander banners via normalizeJLinkVersion.
func versionFromJLinkDir(path string) string {
	dir := filepath.Base(filepath.Dir(path))
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		dir = filepath.Base(filepath.Dir(resolved))
	}
	m := jLinkDirVerRe.FindStringSubmatch(dir)
	if m == nil {
		return ""
	}
	digits, suffix := m[1], m[2]
	if len(digits) < 3 {
		return ""
	}
	major, minor := digits[:len(digits)-2], digits[len(digits)-2:]
	return "V" + major + "." + minor + suffix
}

// jLinkVersionGreater reports whether a is a higher SEGGER release than b.
// Empty versions sort lowest. Hotfix letter: V9.24 < V9.24a < V9.24b < V9.60.
func jLinkVersionGreater(a, b string) bool {
	am, ai, al, aok := parseJLinkVersion(a)
	bm, bi, bl, bok := parseJLinkVersion(b)
	if !aok {
		return false
	}
	if !bok {
		return true
	}
	if am != bm {
		return am > bm
	}
	if ai != bi {
		return ai > bi
	}
	// No letter is treated as the base release; letter hotfixes sort after it.
	return al > bl
}

func parseJLinkVersion(v string) (major, minor int, letter byte, ok bool) {
	v = normalizeJLinkVersion(v)
	m := jLinkSemVerRe.FindStringSubmatch(v)
	if m == nil {
		return 0, 0, 0, false
	}
	if _, err := fmt.Sscanf(m[1], "%d", &major); err != nil {
		return 0, 0, 0, false
	}
	if _, err := fmt.Sscanf(m[2], "%d", &minor); err != nil {
		return 0, 0, 0, false
	}
	if m[3] != "" {
		letter = m[3][0]
		if letter >= 'A' && letter <= 'Z' {
			letter += 'a' - 'A'
		}
	}
	return major, minor, letter, true
}

// versionFromJLinkExe runs the commander briefly and parses the banner version.
func versionFromJLinkExe(exe string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe)
	cmd.Stdin = strings.NewReader("exit\n")
	out, _ := cmd.CombinedOutput()
	if m := jLinkCommanderRe.FindSubmatch(out); m != nil {
		return normalizeJLinkVersion(string(m[1]))
	}
	return ""
}

// installedJLinkVersion prefers the version nrfutil actually loaded, then the
// install directory name, then the commander banner.
func installedJLinkVersion(nrfutilCmd, jlinkExe string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if nrfutilCmd != "" {
		if out, err := exec.CommandContext(ctx, nrfutilCmd, "device", "--version", "--json").CombinedOutput(); err == nil {
			if m := jLinkInstalledRe.FindSubmatch(out); m != nil {
				return normalizeJLinkVersion(string(m[1]))
			}
		}
		if out, err := exec.CommandContext(ctx, nrfutilCmd, "device", "--version").CombinedOutput(); err == nil {
			if m := jLinkDetectedRe.FindSubmatch(out); m != nil {
				return normalizeJLinkVersion(string(m[1]))
			}
		}
	}
	if v := versionFromJLinkDir(jlinkExe); v != "" {
		return v
	}
	return versionFromJLinkExe(jlinkExe)
}

// jLinkStatusLabel is the preflight fragment when the version is known,
// e.g. "SEGGER J-Link V9.60 OK". version must be non-empty.
func jLinkStatusLabel(version string) string {
	return "SEGGER J-Link " + version + " OK"
}

// suggestedJLinkVersion is the pack version nrfutil-device was tested with
// (e.g. "V9.24a"), or "" if unknown. Prefer this over any host-local install.
func suggestedJLinkVersion(command string) string {
	return normalizeJLinkVersion(testedJLinkVersion(command))
}

// jLinkInstallHint appends the nrfutil-tested version when available so the
// operator knows which pack to download.
func jLinkInstallHint(command string) string {
	if v := suggestedJLinkVersion(command); v != "" {
		return "建議版本 " + v + "（nrfutil device tested 版，較新版通常亦可）"
	}
	return "請至 SEGGER 官網下載 J-Link Software and Documentation Pack"
}

// jLinkMissingMessage asks the operator to install into the standard path.
func jLinkMissingMessage(command string) string {
	return "找不到 SEGGER J-Link；請安裝 J-Link Software（SEGGER 官網），" +
		jLinkInstallHint(command) +
		"。裝到標準路徑即可、不必手動設 PATH"
}

// jLinkUnknownVersionMessage is used when a binary is present but no install
// version can be read — treat as broken install and ask for a clean reinstall.
func jLinkUnknownVersionMessage(command string) string {
	return "偵測到 SEGGER J-Link 但讀不到安裝版本，請重新安裝 J-Link Software（SEGGER 官網），" +
		jLinkInstallHint(command) +
		"。裝到標準路徑後重開程式"
}

// probeJLink reports whether a usable SEGGER J-Link install is present (PATH or
// standard install dir) and its version can be read. On success the second
// return is the version (e.g. "V9.60"). On failure it is a human error asking
// to install or reinstall. The actual probe link is verified later by
// `nrfutil device list`.
func probeJLink(command string) (bool, string) {
	path, ok := resolveJLink()
	if !ok {
		return false, jLinkMissingMessage(command)
	}
	ver := installedJLinkVersion(command, path)
	if ver == "" {
		return false, jLinkUnknownVersionMessage(command)
	}
	return true, ver
}

// preflight aggregates the environment checks and names whichever tool is missing.
// deviceReady is preserved from the nrfutil probe even when J-Link is missing, so
// the UI can tell "device command already installed" apart from the aggregate.
func preflight(command string) toolStatus {
	status := probeNRFUtil(command)
	if !status.ready {
		return status
	}
	ok, jmsg := probeJLink(command)
	if !ok {
		return toolStatus{ready: false, deviceReady: status.deviceReady, message: jmsg}
	}
	return toolStatus{ready: true, deviceReady: true, message: status.message + " · " + jLinkStatusLabel(jmsg)}
}

func firstOutputLine(ctx context.Context, name string, args ...string) (string, error) {
	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return line, nil
		}
	}
	return "", errors.New("empty version output")
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// nrfutilBinaryName is the bundled nrfutil filename for this OS.
func nrfutilBinaryName() string {
	if runtime.GOOS == "windows" {
		return "nrfutil.exe"
	}
	return "nrfutil"
}

// bundledNRFUtilDirs lists directories that may hold a bundled nrfutil under a
// 3rd/ folder shipped with the release (next to the executable) or the working
// directory.
func bundledNRFUtilDirs() []string {
	var dirs []string
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		dirs = append(dirs, filepath.Join(exeDir, "3rd"), filepath.Join(exeDir, "..", "3rd"))
	}
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, filepath.Join(cwd, "3rd"))
	}
	return dirs
}

// findBundledNRFUtil returns the first bundled nrfutil executable found in dirs.
func findBundledNRFUtil(dirs []string) (string, bool) {
	name := nrfutilBinaryName()
	for _, dir := range dirs {
		path := filepath.Join(dir, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, true
		}
	}
	return "", false
}

// resolveNRFUtil picks the nrfutil to use: an explicit NRFUTIL_PATH, then the
// bundled 3rd/ copy shipped with the release, then plain "nrfutil" on PATH.
func resolveNRFUtil() string {
	if path := strings.TrimSpace(os.Getenv("NRFUTIL_PATH")); path != "" {
		return path
	}
	if path, ok := findBundledNRFUtil(bundledNRFUtilDirs()); ok {
		prepareBundled(path)
		return path
	}
	return "nrfutil"
}

// prepareBundled makes a bundled tool runnable. Zipping can drop the exec bit,
// and a downloaded macOS copy carries com.apple.quarantine which Gatekeeper
// blocks; clearing it here spares the operator a manual "allow" step. Both are
// best-effort.
func prepareBundled(path string) {
	_ = os.Chmod(path, 0o755)
	if runtime.GOOS == "darwin" {
		_ = exec.Command("xattr", "-d", "com.apple.quarantine", path).Run()
	}
}

func main() {
	noBrowser := flag.Bool("no-browser", false, "serve only; do not launch a browser")
	showVersion := flag.Bool("version", false, "print version and exit")
	// :0 = OS picks a free port. Set e.g. 127.0.0.1:17832 for a fixed factory port.
	addr := flag.String("addr", "127.0.0.1:0", "loopback listen address (host:port)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("nRF Factory %s (%s, built %s)\n", version, commit, date)
		return
	}

	command := resolveNRFUtil()
	status := preflight(command)
	application := newApp(command, execCommand, status.ready, status.message)
	application.deviceReady = status.deviceReady

	listenAddr := strings.TrimSpace(*addr)
	if listenAddr == "" {
		listenAddr = "127.0.0.1:0"
	}
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	url := "http://" + listener.Addr().String()
	fmt.Printf("nRF Factory %s\n", version)
	fmt.Println("nRF Factory:", url)

	server := &http.Server{
		Handler:           application.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Default: open a dedicated Chrome/Edge app window whose close ends the
	// session. Fall back to a normal browser tab when no Chromium is found.
	var session *browserSession
	switch {
	case *noBrowser:
		fmt.Println("未開啟瀏覽器；用上面的網址自行開啟，按 Ctrl+C 停止")
	default:
		if s, ok := startAppWindow(url); ok {
			session = s
			fmt.Println("關閉 app 視窗或按 Ctrl+C 即停止 nRF Factory")
		} else {
			if err := openBrowser(url); err != nil {
				log.Printf("open browser: %v", err)
			}
			fmt.Println("找不到 Chrome/Edge app 視窗模式，已開一般分頁；按 Ctrl+C 停止")
		}
	}

	// A nil channel blocks forever, so without an app window only a signal or a
	// server failure ends the wait.
	var browserClosed <-chan struct{}
	if session != nil {
		browserClosed = session.done
	}

	select {
	case <-browserClosed:
		fmt.Println("app 視窗已關閉，停止服務")
	case s := <-sigCh:
		fmt.Printf("收到 %s，停止服務\n", s)
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Print(err)
		}
	}

	if application.isBusy() {
		fmt.Println("燒錄仍在進行，等待完成後才停止（再按一次 Ctrl+C 可強制停止）")
		ticker := time.NewTicker(200 * time.Millisecond)
		timeout := time.NewTimer(flashTimeout)
	waitForFlash:
		for application.isBusy() {
			select {
			case s := <-sigCh:
				fmt.Printf("收到 %s，強制停止\n", s)
				break waitForFlash
			case <-ticker.C:
			case <-timeout.C:
				fmt.Println("等待燒錄完成逾時，強制停止")
				break waitForFlash
			}
		}
		ticker.Stop()
		timeout.Stop()
	}

	session.kill()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
