@echo off
REM Double-click on Windows Explorer to run nRF Factory (Git Bash or WSL not required for the .exe path).
REM Prefers dist\nrf-factory-windows-amd64.exe via bash exec.sh if bash exists; else runs the exe directly.

setlocal
cd /d "%~dp0"

if exist "dist\nrf-factory-windows-amd64.exe" (
  if exist "C:\Program Files\Git\bin\bash.exe" (
    "C:\Program Files\Git\bin\bash.exe" "%~dp0exec.sh"
    goto :end
  )
  where bash >nul 2>&1
  if %ERRORLEVEL%==0 (
    bash "%~dp0exec.sh"
    goto :end
  )
  REM No bash: run exe and open default browser (no --app= session).
  start "" "dist\nrf-factory-windows-amd64.exe"
  echo Started nrf-factory-windows-amd64.exe
  echo Close the console or the process to stop. For app-window session, install Git Bash and use exec.sh.
  pause
  goto :end
)

echo dist\nrf-factory-windows-amd64.exe not found.
echo Build it first, or set NRF_FACTORY_BIN.
pause

:end
endlocal
