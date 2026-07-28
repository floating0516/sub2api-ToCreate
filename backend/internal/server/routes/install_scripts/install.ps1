param(
  [Parameter(Mandatory = $true)]
  [string]$Token,
  [string]$BaseUrl = "__TOCREATE_BASE_URL__"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$BaseUrl = $BaseUrl.TrimEnd("/")
$Client = ""
$ClientLabel = ""
$CliCommand = ""
$CliPackage = ""
$MinNodeVersion = 22
$CcSwitchReady = $false

function Write-Banner {
  Write-Host ""
  Write-Host "  ToCreate Quick Start" -ForegroundColor Cyan
  Write-Host "  Install one CLI, CC Switch, and your ToCreate provider."
  Write-Host ""
}

function Write-Section([string]$Text) {
  Write-Host ""
  Write-Host $Text -ForegroundColor Cyan
}

function Write-Progress([string]$Text) {
  Write-Host "› $Text" -ForegroundColor Cyan
}

function Write-Success([string]$Text) {
  Write-Host "✓ $Text" -ForegroundColor Green
}

function Write-WarningText([string]$Text) {
  Write-Host "! $Text" -ForegroundColor Yellow
}

function Stop-Install([string]$Text) {
  Write-Host "✗ $Text" -ForegroundColor Red
  exit 1
}

function Refresh-ProcessPath {
  $machinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
  $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
  $env:Path = "$machinePath;$userPath"
}

function Get-NodeMajorVersion {
  try {
    return [int](& node -p "Number(process.versions.node.split('.')[0])")
  } catch {
    return 0
  }
}

function Ensure-Node {
  $node = Get-Command node -ErrorAction SilentlyContinue
  $npm = Get-Command npm.cmd -ErrorAction SilentlyContinue
  if ($node -and $npm -and (Get-NodeMajorVersion) -ge $MinNodeVersion) {
    Write-Success "Node.js $(& node --version) and npm are available"
    return
  }

  if ($node) {
    Write-WarningText "The installed Node.js version is too old; version $MinNodeVersion or newer is required"
  } else {
    Write-Progress "Node.js was not found"
  }

  if (-not (Get-Command winget.exe -ErrorAction SilentlyContinue)) {
    Stop-Install "Install Node.js $MinNodeVersion+ from https://nodejs.org/ and run this command again."
  }

  Write-Progress "Installing Node.js LTS with winget"
  $process = Start-Process winget.exe -ArgumentList @(
    "install",
    "--id", "OpenJS.NodeJS.LTS",
    "--exact",
    "--silent",
    "--accept-package-agreements",
    "--accept-source-agreements"
  ) -Wait -PassThru
  if ($process.ExitCode -ne 0) {
    Stop-Install "winget could not install Node.js. Install it from https://nodejs.org/ and run this command again."
  }

  Refresh-ProcessPath
  if (-not (Get-Command node -ErrorAction SilentlyContinue) -or
      -not (Get-Command npm.cmd -ErrorAction SilentlyContinue) -or
      (Get-NodeMajorVersion) -lt $MinNodeVersion) {
    Stop-Install "Node.js was installed but is not available in this terminal. Open a new PowerShell window and run the command again."
  }
  Write-Success "Node.js $(& node --version) and npm are ready"
}

function Get-Architecture {
  $architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
  switch ($architecture) {
    "x64" { return "amd64" }
    "arm64" { return "arm64" }
    default { Stop-Install "Unsupported CPU architecture: $architecture" }
  }
}

function Show-ApiError([object]$ErrorObject) {
  $reason = [string]$ErrorObject.reason
  $message = [string]$ErrorObject.message
  switch ($reason) {
    "token_expired" { Stop-Install "The install token expired. Return to Quick Start and choose 'Refresh command'." }
    "token_used" { Stop-Install "This install token was already used. Return to Quick Start and refresh the command." }
    "token_revoked" { Stop-Install "This install token was revoked. Return to Quick Start and refresh the command." }
    "key_disabled" { Stop-Install "The selected API key was deleted, disabled, expired, or exhausted." }
    "no_credit" { Stop-Install "This account needs balance or an active subscription before installation." }
    "client_mismatch" { Stop-Install "The selected API key group no longer matches this CLI." }
    "install_token_rate_limited" { Stop-Install "Too many token requests were made. Wait a minute and try again." }
    default { Stop-Install "ToCreate returned an error: $message" }
  }
}

function Invoke-InstallApi([string]$Path, [hashtable]$Body) {
  try {
    return Invoke-RestMethod `
      -Uri "$BaseUrl$Path" `
      -Method Post `
      -ContentType "application/json" `
      -Body ($Body | ConvertTo-Json -Compress) `
      -TimeoutSec 30
  } catch {
    $raw = $_.ErrorDetails.Message
    if ($raw) {
      $parsedError = $null
      try {
        $parsedError = $raw | ConvertFrom-Json
      } catch {
        $parsedError = $null
      }
      if ($parsedError) {
        Show-ApiError $parsedError
      }
    }
    Stop-Install "Could not reach ToCreate. Check the network connection and run the command again."
  }
}

function Load-InstallMetadata([string]$Arch) {
  $response = Invoke-InstallApi "/api/v1/install-token/peek" @{
    token = $Token
    os = "windows"
    arch = $Arch
  }
  $script:Client = [string]$response.data.client
  switch ($script:Client) {
    "claude-code" {
      $script:ClientLabel = "Claude Code"
      $script:CliCommand = "claude"
      $script:CliPackage = "@anthropic-ai/claude-code@latest"
    }
    "codex" {
      $script:ClientLabel = "Codex"
      $script:CliCommand = "codex"
      $script:CliPackage = "@openai/codex@latest"
    }
    "gemini-cli" {
      $script:ClientLabel = "Gemini CLI"
      $script:CliCommand = "gemini"
      $script:CliPackage = "@google/gemini-cli@latest"
    }
    default {
      Stop-Install "The install token returned an unsupported client."
    }
  }
  Write-Success "Install token is valid for $ClientLabel"
}

function Install-Cli {
  if (Get-Command $CliCommand -ErrorAction SilentlyContinue) {
    Write-Success "$ClientLabel is already installed"
    return
  }

  Write-Progress "Installing $ClientLabel"
  & npm.cmd install -g $CliPackage
  if ($LASTEXITCODE -ne 0) {
    Stop-Install "$ClientLabel installation failed. Check npm network access and global install permissions, then run the command again."
  }
  Refresh-ProcessPath
  if (-not (Get-Command $CliCommand -ErrorAction SilentlyContinue)) {
    Stop-Install "$ClientLabel was installed, but $CliCommand is not on PATH. Open a new terminal and run the command again."
  }
  Write-Success "$ClientLabel installed"
}

function Test-CcSwitchProtocol {
  return (Test-Path "Registry::HKEY_CLASSES_ROOT\ccswitch") -or
    (Test-Path "Registry::HKEY_CURRENT_USER\Software\Classes\ccswitch")
}

function Test-CcSwitchInstalled {
  $paths = @(
    "$env:LOCALAPPDATA\Programs\CC Switch\CC Switch.exe",
    "$env:ProgramFiles\CC Switch\CC Switch.exe",
    "${env:ProgramFiles(x86)}\CC Switch\CC Switch.exe"
  )
  foreach ($path in $paths) {
    if ($path -and (Test-Path $path)) {
      return $true
    }
  }
  return (Test-CcSwitchProtocol)
}

function Ensure-CcSwitch([string]$Arch) {
  if (Test-CcSwitchInstalled) {
    $script:CcSwitchReady = $true
    Write-Success "CC Switch is already installed"
    return
  }

  Write-Progress "Downloading CC Switch"
  $platform = if ($Arch -eq "arm64") { "windows-arm64" } else { "windows" }
  $msi = Join-Path ([System.IO.Path]::GetTempPath()) "tocreate-cc-switch-$([Guid]::NewGuid().ToString('N')).msi"
  try {
    Invoke-WebRequest -Uri "$BaseUrl/download/cc-switch/$platform" -OutFile $msi -UseBasicParsing
    $process = Start-Process msiexec.exe -ArgumentList @(
      "/i",
      "`"$msi`"",
      "/qn",
      "/norestart"
    ) -Wait -PassThru
    if ($process.ExitCode -ne 0 -and $process.ExitCode -ne 3010) {
      throw "msiexec exit code $($process.ExitCode)"
    }
    Start-Sleep -Seconds 2
    $script:CcSwitchReady = Test-CcSwitchInstalled
  } catch {
    $script:CcSwitchReady = $false
  } finally {
    Remove-Item $msi -Force -ErrorAction SilentlyContinue
  }

  if ($CcSwitchReady) {
    Write-Success "CC Switch installed"
  } else {
    Write-WarningText "CC Switch could not be installed automatically; browser confirmation will be used"
  }
}

function Show-NextSteps {
  Write-Host ""
  Write-Host "Next steps" -ForegroundColor White
  Write-Host "1. Finish and enable the $ClientLabel provider in CC Switch."
  Write-Host "2. Close any existing CLI session and start a new one:"
  Write-Host "   $CliCommand" -ForegroundColor Cyan
  Write-Host "3. Send a short test request, for example: `"Reply with ToCreate connected.`""
  Write-Host "Docs: $BaseUrl/custom/codex-claude-import"
}

function Redeem-AndImport([string]$Arch) {
  $response = Invoke-InstallApi "/api/v1/install-token/redeem" @{
    token = $Token
    os = "windows"
    arch = $Arch
  }
  $deeplink = [string]$response.data.deeplink
  $confirmUrl = [string]$response.data.confirm_url
  if (-not $deeplink -or -not $confirmUrl) {
    Stop-Install "ToCreate returned an incomplete CC Switch import payload."
  }

  if ($CcSwitchReady -and (Test-CcSwitchProtocol)) {
    try {
      Start-Process $deeplink
      Write-Success "CC Switch import opened"
      Write-Host ""
      Write-Host "✨ Installation complete" -ForegroundColor Green
      Show-NextSteps
      return
    } catch {
      $script:CcSwitchReady = $false
    }
  }

  Write-WarningText "Automatic CC Switch import needs browser confirmation"
  try {
    Start-Process $confirmUrl
    Write-Success "Opened the one-click import confirmation page"
  } catch {
    Write-Host "Open this URL in a browser to finish the import:"
    Write-Host $confirmUrl
  }
  Write-Host ""
  Write-Host "Installer finished - confirmation required" -ForegroundColor Yellow
  Show-NextSteps
}

if ([string]::IsNullOrWhiteSpace($Token)) {
  Stop-Install "Missing -Token. Copy a fresh command from ToCreate Quick Start."
}

Write-Banner
$arch = Get-Architecture

Write-Section "1. Preflight"
Write-Success "Detected windows/$arch"
Write-Success "Install token is present"
Load-InstallMetadata $arch
Ensure-Node

Write-Section "2. Install tools"
Install-Cli
Ensure-CcSwitch $arch

Write-Section "3. Import config"
Write-Progress "Redeeming the one-time install token"
Redeem-AndImport $arch
