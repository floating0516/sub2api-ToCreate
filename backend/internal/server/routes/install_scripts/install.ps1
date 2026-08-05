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
  $banner = @'
 ______   ______   ______     ______     ______     ______   ______   ______
/\__  _\ /\  __ \ /\  ___\   /\  == \   /\  ___\   /\  __ \ /\__  _\ /\  ___\
\/_/\ \/ \ \ \/\ \\ \ \____  \ \  __<   \ \  __\   \ \  __ \\/_/\ \/ \ \  __\
   \ \_\  \ \_____\\ \_____\  \ \_\ \_\  \ \_____\  \ \_\ \_\  \ \_\  \ \_____\
    \/_/   \/_____/ \/_____/   \/_/ /_/   \/_____/   \/_/\/_/   \/_/   \/_____/
'@
  Write-Host $banner -ForegroundColor Cyan
  Write-Host ""
  Write-Host "                               ToCreate Quick Start" -ForegroundColor Green
  Write-Host ""
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

function Test-NodeReady {
  return (
    (Get-Command node -ErrorAction SilentlyContinue) -and
    (Get-Command npm.cmd -ErrorAction SilentlyContinue) -and
    (Get-NodeMajorVersion) -ge $MinNodeVersion
  )
}

function Invoke-NodeWinget([string]$Action, [switch]$Force) {
  $wingetArguments = @(
    $Action,
    "--id", "OpenJS.NodeJS.LTS",
    "--exact",
    "--source", "winget",
    "--silent",
    "--accept-package-agreements",
    "--accept-source-agreements"
  )
  if ($Force) {
    $wingetArguments += "--force"
  }

  try {
    $process = Start-Process winget.exe -ArgumentList $wingetArguments -Wait -PassThru
    return $process.ExitCode
  } catch {
    return 1
  }
}

function Ensure-Node {
  $node = Get-Command node -ErrorAction SilentlyContinue
  if (Test-NodeReady) {
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

  $action = if ($node) { "upgrade" } else { "install" }
  $progressLabel = if ($node) { "Upgrading" } else { "Installing" }
  Write-Progress "$progressLabel Node.js LTS with winget"
  $exitCode = Invoke-NodeWinget $action
  Refresh-ProcessPath

  if ($exitCode -ne 0 -or -not (Test-NodeReady)) {
    Write-WarningText "winget did not make Node.js $MinNodeVersion+ available on the first attempt"
    Write-Progress "Installing the current Node.js LTS package with winget"
    $exitCode = Invoke-NodeWinget "install" -Force
    Refresh-ProcessPath
  }

  if ($exitCode -ne 0 -or -not (Test-NodeReady)) {
    $activeNode = Get-Command node -ErrorAction SilentlyContinue
    if ($activeNode) {
      Stop-Install "Node.js $MinNodeVersion+ was installed, but $($activeNode.Source) still reports $(& node --version). Remove the older Node.js PATH entry and run this command again."
    }
    Stop-Install "winget could not install Node.js $MinNodeVersion+. Install it from https://nodejs.org/ and run this command again."
  }
  Write-Success "Node.js $(& node --version) and npm are ready"
}

function Get-Architecture {
  $architecture = ""
  $runtimeInformationTypes = @(
    "System.Runtime.InteropServices.RuntimeInformation, mscorlib",
    "System.Runtime.InteropServices.RuntimeInformation, System.Private.CoreLib",
    "System.Runtime.InteropServices.RuntimeInformation, System.Runtime.InteropServices.RuntimeInformation"
  )

  foreach ($typeName in $runtimeInformationTypes) {
    try {
      $runtimeInformation = [Type]::GetType($typeName, $false)
      if (-not $runtimeInformation) {
        continue
      }
      $osArchitecture = $runtimeInformation.GetProperty("OSArchitecture")
      if (-not $osArchitecture) {
        continue
      }
      $value = $osArchitecture.GetValue($null)
      if ($value) {
        $architecture = $value.ToString().ToLowerInvariant()
        break
      }
    } catch {
      $architecture = ""
    }
  }

  if ([string]::IsNullOrWhiteSpace($architecture)) {
    $architecture = [string]$env:PROCESSOR_ARCHITEW6432
    if ([string]::IsNullOrWhiteSpace($architecture)) {
      $architecture = [string]$env:PROCESSOR_ARCHITECTURE
    }
    $architecture = $architecture.Trim().ToLowerInvariant()
  }

  switch ($architecture) {
    "x64" { return "amd64" }
    "amd64" { return "amd64" }
    "arm64" { return "arm64" }
    "aarch64" { return "arm64" }
    "" { Stop-Install "Could not detect the Windows CPU architecture." }
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
  $client = [string]$response.data.client
  $metadata = $null
  switch ($client) {
    "claude-code" {
      $metadata = [pscustomobject]@{
        Client = $client
        ClientLabel = "Claude Code"
        CliCommand = "claude"
        CliPackage = "@anthropic-ai/claude-code@latest"
      }
    }
    "codex" {
      $metadata = [pscustomobject]@{
        Client = $client
        ClientLabel = "Codex"
        CliCommand = "codex"
        CliPackage = "@openai/codex@latest"
      }
    }
    "gemini-cli" {
      $metadata = [pscustomobject]@{
        Client = $client
        ClientLabel = "Gemini CLI"
        CliCommand = "gemini"
        CliPackage = "@google/gemini-cli@latest"
      }
    }
    default {
      Stop-Install "The install token returned an unsupported client."
    }
  }
  Write-Success "Install token is valid for $($metadata.ClientLabel)"
  return $metadata
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
    Write-Success "CC Switch is already installed"
    return $true
  }

  Write-Progress "Downloading CC Switch"
  $platform = if ($Arch -eq "arm64") { "windows-arm64" } else { "windows" }
  $msi = Join-Path ([System.IO.Path]::GetTempPath()) "tocreate-cc-switch-$([Guid]::NewGuid().ToString('N')).msi"
  $ccSwitchReady = $false
  try {
    $null = Invoke-WebRequest -Uri "$BaseUrl/download/cc-switch/$platform" -OutFile $msi -UseBasicParsing
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
    $ccSwitchReady = Test-CcSwitchInstalled
  } catch {
    $ccSwitchReady = $false
  } finally {
    Remove-Item $msi -Force -ErrorAction SilentlyContinue
  }

  if ($ccSwitchReady) {
    Write-Success "CC Switch installed"
  } else {
    Write-WarningText "CC Switch could not be installed automatically; browser confirmation will be used"
  }
  return $ccSwitchReady
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

function Redeem-AndImport([string]$Arch, [bool]$CcSwitchAvailable) {
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

  if ($CcSwitchAvailable -and (Test-CcSwitchProtocol)) {
    try {
      Start-Process $deeplink
      Write-Success "CC Switch import opened"
      Write-Host ""
      Write-Host "✨ Installation complete" -ForegroundColor Green
      Show-NextSteps
      return
    } catch {
      $CcSwitchAvailable = $false
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
$installMetadata = Load-InstallMetadata $arch
$Client = $installMetadata.Client
$ClientLabel = $installMetadata.ClientLabel
$CliCommand = $installMetadata.CliCommand
$CliPackage = $installMetadata.CliPackage
Ensure-Node

Write-Section "2. Install tools"
Install-Cli
$CcSwitchReady = Ensure-CcSwitch $arch

Write-Section "3. Import config"
Write-Progress "Redeeming the one-time install token"
Redeem-AndImport $arch $CcSwitchReady
