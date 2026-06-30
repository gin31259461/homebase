---
name: homebase-powershell-remote-bootstrap
description: > 
    Editing, reviewing, or debugging Windows PowerShell bootstrap scripts that are 
    downloaded and executed through Invoke-RestMethod/Invoke-WebRequest aliases 
    such as irm or iwr piped to Invoke-Expression/iex,
    especially Homebase bootstrap/windows.ps1, 
    remote install snippets, script param blocks, execution policy handling,
    BOM safety, or parser errors like Unexpected attribute CmdletBinding.
---

# Homebase PowerShell Remote Bootstrap

Use this workflow for PowerShell entrypoints intended to work both as local
scripts and as remote one-liners such as `irm <url> | iex`.

## Script Shape

- Keep `#Requires -Version 5.1` as the first bytes of the file and keep the
  file UTF-8 without BOM.
- Avoid top-level `[CmdletBinding()]` in scripts promoted for `irm/iwr | iex`.
  It is valid for direct script execution, but remote/piped execution can parse
  it as a standalone statement and fail with `Unexpected attribute
  'CmdletBinding'`.
- Put a plain top-level `param(...)` block immediately after `#Requires` when
  script parameters are needed.
- Keep function definitions and multi-line blocks out of test cases that pipe
  line arrays to `iex`; that shape intentionally reproduces fragile input and
  may fail before the real remote path does.

## Remote Execution Hazards

- Treat `irm <url> | iex` as text evaluation, not as normal script-file
  execution.
- Remember that aliases differ by host and version:
  `irm` is `Invoke-RestMethod`, `iwr` is `Invoke-WebRequest`, and `iex` is
  `Invoke-Expression`.
- If reproducing parser behavior locally, distinguish raw string input from
  line-array input:

```powershell
# Parse the script as a single downloaded string.
$null = [scriptblock]::Create((Get-Content -Raw .\bootstrap\windows.ps1))

# Stress the fragile pipeline shape that parses each line separately.
Get-Content .\bootstrap\windows.ps1 | Invoke-Expression
```

## Execution Policy

- Do not let execution-policy setup abort a bootstrap that is already running.
  Policies can be overridden by `Process`, `MachinePolicy`, or `UserPolicy`,
  and `Set-ExecutionPolicy -ErrorAction SilentlyContinue` can still surface a
  terminating security error.
- Prefer:

```powershell
try {
  Set-ExecutionPolicy RemoteSigned -Scope CurrentUser -Force `
    -ErrorAction Stop | Out-Null
} catch {
  # Policy may already be overridden by Process, MachinePolicy, or UserPolicy.
}
```

## Validation Checklist

- Add or update a regression test that protects the remote-safe top of file:
  no BOM, starts with `#Requires`, and no top-level `[CmdletBinding()]`.
- Parse-check with both Windows PowerShell 5.1 and PowerShell 7 when available:

```powershell
$parse = '$null = [scriptblock]::Create(' +
  '(Get-Content -Raw .\bootstrap\windows.ps1))'
powershell -NoProfile -ExecutionPolicy Bypass -Command $parse
pwsh -NoProfile -ExecutionPolicy Bypass -Command $parse
```

- Do not execute the full remote bootstrap as verification unless the user
  explicitly wants side effects. It may install packages, clone or update repos,
  rebuild binaries, and run bootstrap flows.
- For Homebase code changes, still run the repository verification from
  `AGENTS.md`: `gofmt`, `go test ./...`, `go vet ./...`, and build `hb`.
