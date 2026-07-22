package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestAppWindowArgs(t *testing.T) {
	args := appWindowArgs("http://127.0.0.1:17832", "/tmp/profile")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--app=http://127.0.0.1:17832") {
		t.Fatalf("missing --app flag: %v", args)
	}
	if !strings.Contains(joined, "--user-data-dir=/tmp/profile") {
		t.Fatalf("missing --user-data-dir flag: %v", args)
	}
}

func TestBrowserCandidatesNonEmpty(t *testing.T) {
	if len(browserCandidates()) == 0 {
		t.Fatalf("no browser candidates for %s", runtime.GOOS)
	}
}

// fakeBrowser writes an executable /bin/sh stub that runs script, then points
// the session's browser finder at it. The launch/wait/kill machinery is the
// same for the stub as for a real Chrome, so we never bind a real browser.
func fakeBrowser(t *testing.T, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake-browser session test is unix-only")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-browser")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	prev := browserFinder
	browserFinder = func() (string, bool) { return path, true }
	t.Cleanup(func() { browserFinder = prev })
	t.Setenv("NRF_FACTORY_PROFILE", t.TempDir())
}

func TestBrowserSessionClosesOnExit(t *testing.T) {
	fakeBrowser(t, "exit 0")

	session, ok := startAppWindow("http://127.0.0.1:0")
	if !ok {
		t.Fatal("startAppWindow returned not-ok for the fake browser")
	}
	select {
	case <-session.done:
	case <-time.After(5 * time.Second):
		t.Fatal("session did not close after the browser exited")
	}
}

func TestBrowserSessionKillStopsProcess(t *testing.T) {
	// exec replaces the shell with sleep, so killing the pid kills the sleeper.
	fakeBrowser(t, "exec sleep 30")

	session, ok := startAppWindow("http://127.0.0.1:0")
	if !ok {
		t.Fatal("startAppWindow returned not-ok for the fake browser")
	}
	select {
	case <-session.done:
		t.Fatal("session closed before kill was called")
	case <-time.After(100 * time.Millisecond):
	}

	session.kill()
	select {
	case <-session.done:
	case <-time.After(5 * time.Second):
		t.Fatal("session did not close after kill")
	}
}
