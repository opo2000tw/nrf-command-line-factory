package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

// browserFinder locates the browser to launch. It is a package variable so
// tests can point the session at a fake executable instead of a real Chrome.
var browserFinder = findBrowser

// appWindowArgs turns url into a dedicated Chromium-family app window. A private
// --user-data-dir plus a single --app= window means the process we start owns
// the profile and exits when the user closes that window, so the session is
// waitable across macOS and Windows.
func appWindowArgs(url, profileDir string) []string {
	return []string{
		"--user-data-dir=" + profileDir,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-features=Translate",
		"--app=" + url,
	}
}

// browserCandidates lists Chromium-family executables for the current OS,
// most-preferred first. PATH names are resolved with exec.LookPath; absolute
// paths are probed on disk.
func browserCandidates() []string {
	switch runtime.GOOS {
	case "windows":
		bases := []string{
			os.Getenv("ProgramFiles"),
			os.Getenv("ProgramFiles(x86)"),
			os.Getenv("LocalAppData"),
		}
		rels := []string{
			`Google\Chrome\Application\chrome.exe`,
			`Microsoft\Edge\Application\msedge.exe`,
			`Chromium\Application\chrome.exe`,
			`BraveSoftware\Brave-Browser\Application\brave.exe`,
		}
		var out []string
		for _, rel := range rels {
			for _, base := range bases {
				if base != "" {
					out = append(out, filepath.Join(base, rel))
				}
			}
		}
		return out
	case "darwin":
		return []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
		}
	default:
		return []string{
			"google-chrome", "google-chrome-stable",
			"chromium", "chromium-browser",
			"microsoft-edge", "msedge",
			"brave-browser", "brave",
		}
	}
}

// findBrowser returns the first Chromium-family browser present on this machine.
func findBrowser() (string, bool) {
	for _, candidate := range browserCandidates() {
		if filepath.IsAbs(candidate) {
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, true
			}
			continue
		}
		if resolved, err := exec.LookPath(candidate); err == nil {
			return resolved, true
		}
	}
	return "", false
}

// appProfileDir returns the dedicated Chrome profile directory, creating it if
// needed. NRF_FACTORY_PROFILE overrides the default temp location.
func appProfileDir() (string, error) {
	dir := os.Getenv("NRF_FACTORY_PROFILE")
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "nrf-factory-profile")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// clearSingletonLocks removes Chromium singleton files left by a killed session;
// a stale lock makes a fresh launch exit immediately.
func clearSingletonLocks(profileDir string) {
	for _, name := range []string{"SingletonLock", "SingletonCookie", "SingletonSocket"} {
		_ = os.Remove(filepath.Join(profileDir, name))
	}
}

// browserSession is a launched app-window browser process. done closes when the
// process exits, whether because the user closed the window or kill was called.
type browserSession struct {
	cmd      *exec.Cmd
	done     chan struct{}
	killOnce sync.Once
}

// startAppWindow launches url in a dedicated Chromium app window. It returns
// (nil, false) when no browser is found or the launch fails, so the caller can
// fall back to a plain system-browser tab.
func startAppWindow(url string) (*browserSession, bool) {
	bin, ok := browserFinder()
	if !ok {
		return nil, false
	}
	profileDir, err := appProfileDir()
	if err != nil {
		return nil, false
	}
	clearSingletonLocks(profileDir)

	cmd := exec.Command(bin, appWindowArgs(url, profileDir)...)
	if err := cmd.Start(); err != nil {
		return nil, false
	}

	session := &browserSession{cmd: cmd, done: make(chan struct{})}
	go func() {
		_ = cmd.Wait()
		close(session.done)
	}()
	return session, true
}

// kill terminates the browser process if it is still running. Safe on a nil
// session and safe to call more than once.
//
// ponytail: kills only the process we started, not the whole Chromium process
// group. For a dedicated-profile app window closing the master is enough; a
// process-group kill would need per-OS SysProcAttr and is deferred.
func (s *browserSession) kill() {
	if s == nil || s.cmd == nil || s.cmd.Process == nil {
		return
	}
	s.killOnce.Do(func() {
		_ = s.cmd.Process.Kill()
	})
}
