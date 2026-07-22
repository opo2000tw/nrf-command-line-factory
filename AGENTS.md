# Repository Guidelines

## Project Structure & Module Organization

`README.md` defines a Go-based nRF programming desktop application for Windows 10 and 11, with external firmware selection, L/R target selection, status output, and success counters.

As implementation is added, place executable entry points in `cmd/nrf-factory/`, reusable device and application logic in `internal/`, and UI resources in `assets/`. Keep Go tests beside the package they exercise as `*_test.go`. Write generated executables to `dist/` and keep that directory untracked. Firmware images must remain external files selected at runtime; do not embed them in the application.

## Build, Test, and Development Commands

After `go.mod` and the initial source tree are present, use these standard commands:

- `go mod tidy` synchronizes module dependencies.
- `go fmt ./...` formats all Go packages.
- `go vet ./...` checks common correctness issues.
- `go test ./...` runs the complete test suite.
- `go build ./...` verifies that every package builds.
- `GOOS=windows GOARCH=amd64 go build -o dist/nrf-factory.exe ./cmd/nrf-factory` creates the Windows executable from macOS or Linux.

Do not document a command as supported until its required module, package, or configuration file is committed.

## Coding Style & Naming Conventions

Use `gofmt` output without manual alignment. Package names should be short, lowercase, and singular. Use PascalCase for exported identifiers, camelCase for unexported identifiers, and descriptive lowercase filenames. Keep UI code separate from nRF command execution so device behavior can be tested without launching the desktop interface. Preserve the user-facing `L` and `R` terminology used by the requirements.

## Testing Guidelines

Prefer table-driven unit tests for firmware-path validation, target selection, command failures, and counter reset behavior. Name tests `TestFunction` and use named subtests for scenarios. Isolate hardware access behind interfaces and use fakes in routine tests. There is no coverage threshold yet; new behavior should include focused regression tests. Clearly identify tests requiring an attached nRF device in the pull request.

## Commit & Pull Request Guidelines

Use concise, imperative subjects such as `Add firmware selection validation`, and keep each commit focused on one concern.

Pull requests should explain the user-visible change, list verification commands, and link the relevant issue. Include screenshots for UI changes. For hardware-dependent work, record the Windows version, device model, programming tool, and observed result. Do not commit proprietary firmware, device identifiers, credentials, or machine-specific tool paths.
