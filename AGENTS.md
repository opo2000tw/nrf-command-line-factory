# Repository Guidelines

## Project Structure & Module Organization

`README.md` defines a Go-based nRF programming desktop application for Windows 10 and 11, with external firmware selection, L/R target selection, status output, and success counters.

The executable entry point and application logic live directly in `cmd/nrf-factory/` (`main.go`, `browser.go`), with the embedded UI at `cmd/nrf-factory/web/index.html`. Keep Go tests beside the code they exercise as `*_test.go`. Cross-compiled binaries go to `dist/` (untracked). Firmware images must remain external files selected at runtime; do not embed them in the application.

## Build, Test, and Development Commands

After `go.mod` and the initial source tree are present, use these standard commands:

- `go mod tidy` synchronizes module dependencies.
- `go fmt ./...` formats all Go packages.
- `go vet ./...` checks common correctness issues.
- `go test ./...` runs the complete test suite.
- `go build ./...` verifies that every package builds.
- `make dist` cross-compiles the release binaries (windows/amd64, darwin/arm64) into `dist/` with version stamping; `make package` also builds the release zips and `SHA256SUMS`. A bare `GOOS=windows GOARCH=amd64 go build -o dist/nrf-factory-windows-amd64.exe ./cmd/nrf-factory` works too but reports version `dev`.

Do not document a command as supported until its required module, package, or configuration file is committed.

## Coding Style & Naming Conventions

Use `gofmt` output without manual alignment. Package names should be short, lowercase, and singular. Use PascalCase for exported identifiers, camelCase for unexported identifiers, and descriptive lowercase filenames. Keep UI code separate from nRF command execution so device behavior can be tested without launching the desktop interface. Preserve the user-facing `L` and `R` terminology used by the requirements.

## Testing Guidelines

Prefer table-driven unit tests for firmware-path validation, target selection, command failures, and counter reset behavior. Name tests `TestFunction` and use named subtests for scenarios. Isolate hardware access behind interfaces and use fakes in routine tests. There is no coverage threshold yet; new behavior should include focused regression tests. Clearly identify tests requiring an attached nRF device in the pull request.

## Windows Verification via Parallels (prlctl)

Windows-side checks run against the local Parallels VM named `Windows 11`, driven purely from the Mac CLI with `prlctl` (verify it is running first with `prlctl list --all`). The full verified record lives in `.docs/reviews/windows-parallels-verification.md`.

- Run commands inside the VM with `prlctl exec "Windows 11" powershell -EncodedCommand <base64>`. Write the PowerShell script to a file first and encode it with `iconv -f UTF-8 -t UTF-16LE` piped to `base64`; inlining the command string breaks on nested quoting (zsh to prlctl to PowerShell) and the CP950 codepage.
- The guest shell is Windows PowerShell 5.1: always pass `-UseBasicParsing` to `Invoke-WebRequest` and set `$ProgressPreference` to `SilentlyContinue`, or CLIXML progress noise pollutes the output.
- Exchange files through the `\\Mac\Home` share (the Mac home directory). After copying an executable into the guest, run `Unblock-File` to strip the Mark-of-the-Web, or SmartScreen blocks it.
- Write intermediate results to guest-local files under `C:\Users\Public\` and read them back with `type`; redirect stdout and stderr of long-running commands to files.
- Take screenshots from the Mac side with `prlctl capture "Windows 11" --file <path>`. This grabs the VM framebuffer on the host: no macOS screen-recording permission, unaffected by guest session isolation, and more reliable than capturing the Parallels window. Do not attempt screenshots from inside the guest.
- Session-0 limitation: `prlctl exec` runs in the service context, so GUI apps it starts are not visible on the console session and synchronous exec of a GUI app does not return; `schtasks` redirection to the interactive token is unreliable too. Verify visible windows (for example the Chrome app window) by double-clicking interactively inside the VM, then capture the screen from the host as evidence. Headless services are unaffected: start them with `Start-Process` and probe loopback endpoints with `Invoke-WebRequest`.
- A `prlctl exec` whose script launches a long-lived background child via `Start-Process` does not return even for headless processes with I/O redirected — it waits on the whole process tree. Wrap the call in a Mac-side `timeout`, treat the timeout as expected, and confirm the child is alive with a separate fast `prlctl exec` running `Get-Process -Id <pid>`.
- Multi-byte (Traditional Chinese) text is mojibaked by the console round-trip even when the underlying data is valid UTF-8. Emit it as Base64 of UTF-8 bytes inside the guest (`[Convert]::ToBase64String` over `UTF8.GetBytes`) and decode with `base64 -d` on the Mac; read guest files as raw bytes with `[System.IO.File]::ReadAllBytes`, since `Get-Content` in text mode corrupts the content during its own decode.
- Captured output has CRLF line endings (plain `grep` may silently match nothing; use `rg`) and is wrapped in CLIXML noise (a leading `#< CLIXML` line and a trailing `<Objs>` progress blob). Have the script emit marker-prefixed lines and select only those on the Mac side.

## Commit & Pull Request Guidelines

Use concise, imperative subjects such as `Add firmware selection validation`, and keep each commit focused on one concern.

Pull requests should explain the user-visible change, list verification commands, and link the relevant issue. Include screenshots for UI changes. For hardware-dependent work, record the Windows version, device model, programming tool, and observed result. Do not commit proprietary firmware, device identifiers, credentials, or machine-specific tool paths.
