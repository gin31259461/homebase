#Requires -Version 5.1
param(
  [string] $HomebaseRepo = 'https://github.com/gin31259461/homebase.git',
  [string] $Branch = 'main',
  [string] $DotfilesRepo = '',
  [switch] $Yes,
  [switch] $Install
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
try {
  Set-ExecutionPolicy RemoteSigned -Scope CurrentUser -Force -ErrorAction Stop | Out-Null
} catch {
  # Policy may already be overridden by Process, MachinePolicy, or UserPolicy.
}

function Add-UserPath {
  param([Parameter(Mandatory)][string] $Path)

  if (-not (Test-Path -LiteralPath $Path)) {
    New-Item -ItemType Directory -Path $Path -Force | Out-Null
  }

  $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
  $entries = @($userPath -split ';' | Where-Object { $_ })
  if ($entries -notcontains $Path) {
    [Environment]::SetEnvironmentVariable('Path', (($entries + $Path) -join ';'), 'User')
  }

  $processEntries = @($env:Path -split ';' | Where-Object { $_ })
  if ($processEntries -notcontains $Path) {
    $env:Path = ($processEntries + $Path) -join ';'
  }
}

function Update-ProcessEnvironment {
  $machineEnv = [Environment]::GetEnvironmentVariables('Machine')
  $userEnv = [Environment]::GetEnvironmentVariables('User')
  $machineEnv.GetEnumerator() | ForEach-Object {
    [Environment]::SetEnvironmentVariable($_.Key, $_.Value, 'Process')
  }
  $userEnv.GetEnumerator() | ForEach-Object {
    [Environment]::SetEnvironmentVariable($_.Key, $_.Value, 'Process')
  }
}

function Install-WingetPackage {
  param(
    [Parameter(Mandatory)][string] $Id,
    [Parameter(Mandatory)][string] $Command
  )

  if (Get-Command $Command -ErrorAction SilentlyContinue) {
    return
  }
  if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
    throw "winget is required to install $Id"
  }
  winget install --id $Id --source winget --accept-source-agreements --accept-package-agreements
  Update-ProcessEnvironment
}

$homebaseDir = Join-Path $HOME '.local\lib\homebase'
$binDir = Join-Path $HOME '.local\bin'
$hb = Join-Path $binDir 'hb.exe'

Add-UserPath $binDir

Install-WingetPackage -Id 'Git.Git' -Command 'git'
Install-WingetPackage -Id 'GoLang.Go' -Command 'go'

if (Test-Path -LiteralPath (Join-Path $homebaseDir '.git')) {
  git -C $homebaseDir fetch origin $Branch
  git -C $homebaseDir checkout $Branch
  git -C $homebaseDir pull --ff-only origin $Branch
} elseif (Test-Path -LiteralPath $homebaseDir) {
  throw "Homebase directory already exists but is not a git checkout: $homebaseDir"
} else {
  git clone --branch $Branch --depth 1 $HomebaseRepo $homebaseDir
}

Push-Location $homebaseDir
try {
  go build -o $hb ./cmd/hb
} finally {
  Pop-Location
}

$hbArgs = @('bootstrap')
if ($Yes) { $hbArgs += '--yes' }
if ($Install) { $hbArgs += '--install' }
if ($DotfilesRepo) {
  $hbArgs += '--repo'
  $hbArgs += $DotfilesRepo
}

& $hb @hbArgs
