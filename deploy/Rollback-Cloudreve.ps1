#Requires -Version 7.0
[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$BackupDirectory,
    [string]$DeploymentDirectory = 'D:\docker\server\cloudreve'
)
$ErrorActionPreference = 'Stop'
$deployment = (Resolve-Path -LiteralPath $DeploymentDirectory).Path
$backup = (Resolve-Path -LiteralPath $BackupDirectory).Path
$expectedRoot = (Join-Path $deployment 'backup') + [IO.Path]::DirectorySeparatorChar
if (-not $backup.StartsWith($expectedRoot, [StringComparison]::OrdinalIgnoreCase)) { throw 'Backup must be inside this deployment backup directory.' }
$record = Get-Content -LiteralPath (Join-Path $backup 'deployment.json') -Raw | ConvertFrom-Json
$dockerCommand = Get-Command docker -ErrorAction SilentlyContinue
$dockerExecutable = if ($dockerCommand) { $dockerCommand.Source } else { Join-Path $env:LOCALAPPDATA 'Programs\DockerDesktop\resources\bin\docker.exe' }
$overridePath = Join-Path $deployment 'compose.custom.yaml'
if ($record.hadOverride) {
    Copy-Item -LiteralPath (Join-Path $backup 'compose.custom.yaml') -Destination $overridePath -Force
} elseif (Test-Path -LiteralPath $overridePath) {
    Remove-Item -LiteralPath $overridePath
}
$composeArgs = @('compose', '--project-directory', $deployment, '--env-file', (Join-Path $deployment '.env'), '-f', (Join-Path $deployment 'compose.yaml'))
if ($record.hadOverride) { $composeArgs += @('-f', $overridePath) }
& $dockerExecutable @composeArgs up -d --no-deps --wait cloudreve
if ($LASTEXITCODE -ne 0) { throw 'Rollback failed.' }
Write-Host "Restored image: $($record.previousImage). Existing database and files were retained."
