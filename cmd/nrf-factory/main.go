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
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	maxUploadBytes = 64 << 20
	flashTimeout   = 5 * time.Minute
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
	toolReady   bool
	toolMessage string
}

type snapshot struct {
	LeftCount   int    `json:"leftCount"`
	RightCount  int    `json:"rightCount"`
	Busy        bool   `json:"busy"`
	ToolReady   bool   `json:"toolReady"`
	ToolMessage string `json:"toolMessage"`
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

	ctx, cancel := context.WithTimeout(r.Context(), flashTimeout)
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

func (a *app) program(ctx context.Context, output io.Writer, firmwarePath string) error {
	var discovery bytes.Buffer
	listArgs := []string{"device", "list", "--traits", "jlink", "--json"}
	if err := a.run(ctx, a.command, listArgs, &discovery); err != nil {
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
	if err := a.run(ctx, a.command, args, output); err != nil {
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

func (a *app) currentSnapshot() snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	return snapshot{
		LeftCount:   a.leftCount,
		RightCount:  a.rightCount,
		Busy:        a.busy,
		ToolReady:   a.toolReady,
		ToolMessage: a.toolMessage,
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

func probeNRFUtil(command string) (bool, string) {
	path, err := exec.LookPath(command)
	if err != nil {
		return false, "找不到 nrfutil，請先安裝 Nordic nRF Util 與 device command"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	coreVersion, err := firstOutputLine(ctx, path, "--version")
	if err != nil {
		return false, "nrfutil 無法執行"
	}
	deviceVersion, err := firstOutputLine(ctx, path, "device", "--version")
	if err != nil {
		return false, "缺少 nrfutil device command；請執行 nrfutil install device"
	}
	return true, coreVersion + " · " + deviceVersion
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

func main() {
	noBrowser := flag.Bool("no-browser", false, "do not open the system browser")
	flag.Parse()

	command := strings.TrimSpace(os.Getenv("NRFUTIL_PATH"))
	if command == "" {
		command = "nrfutil"
	}
	ready, message := probeNRFUtil(command)
	application := newApp(command, execCommand, ready, message)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	url := "http://" + listener.Addr().String()
	fmt.Println("nRF Factory:", url)
	if !*noBrowser {
		if err := openBrowser(url); err != nil {
			log.Printf("open browser: %v", err)
		}
	}

	server := &http.Server{
		Handler:           application.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
