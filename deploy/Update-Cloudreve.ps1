#Requires -Version 7.0
[CmdletBinding()]
param(
    [string]$Revision = 'HEAD',
    [string]$DeploymentDirectory = 'D:\docker\server\cloudreve',
    [string]$DatabaseContainer = 'postgresql'
)
$ErrorActionPreference = 'Stop'
$repoDirectory = Split-Path -Parent $PSScriptRoot
$commit = (& git -C $repoDirectory rev-parse "$Revision^{commit}").Trim()
if ($LASTEXITCODE -ne 0 -or $commit -notmatch '^[0-9a-f]{40}$') { throw 'A valid committed Git revision is required.' }
$dockerCommand = Get-Command docker -ErrorAction SilentlyContinue
$script:DockerExecutable = if ($dockerCommand) { $dockerCommand.Source } else { Join-Path $env:LOCALAPPDATA 'Programs\DockerDesktop\resources\bin\docker.exe' }
if (-not (Test-Path -LiteralPath $script:DockerExecutable)) { throw 'Docker Desktop executable was not found.' }
function Invoke-Docker {
    param([Parameter(ValueFromRemainingArguments = $true)][string[]]$DockerArguments)
    & $script:DockerExecutable @DockerArguments
    if ($LASTEXITCODE -ne 0) { throw "Docker command failed (exit $LASTEXITCODE)." }
}
$deployment = (Resolve-Path -LiteralPath $DeploymentDirectory).Path
$composePath = Join-Path $deployment 'compose.yaml'
$envPath = Join-Path $deployment '.env'
$overridePath = Join-Path $deployment 'compose.custom.yaml'
if (-not (Test-Path -LiteralPath $composePath) -or -not (Test-Path -LiteralPath $envPath)) { throw 'Existing deployment compose.yaml and .env are required.' }
$image = "local/cloudreve-receive:$($commit.Substring(0, 12))"
$gitContext = "https://github.com/duminqing/cloudreve.git#$commit"

# Docker downloads the committed source from the user's Fork, including the
# pinned frontend submodule. Tests run inside Dockerfile.receive before linking.
Invoke-Docker @('build', '--file', 'Dockerfile.receive', '--build-arg', "SOURCE_COMMIT=$commit", '--tag', $image, $gitContext)

$backupRoot = Join-Path $deployment 'backup'
$backup = Join-Path $backupRoot ("receive-{0}-{1}" -f (Get-Date -Format 'yyyyMMdd-HHmmss'), $commit.Substring(0, 8))
New-Item -ItemType Directory -Path $backup -Force | Out-Null
foreach ($name in @('compose.yaml', '.env', 'cloudreve.env', 'compose.custom.yaml')) {
    $path = Join-Path $deployment $name
    if (Test-Path -LiteralPath $path) { Copy-Item -LiteralPath $path -Destination (Join-Path $backup $name) }
}
$hadOverride = Test-Path -LiteralPath $overridePath
$previousImage = (Invoke-Docker @('inspect', 'cloudreve', '--format', '{{.Config.Image}}')).Trim()
$dbName = 'cloudreve'
foreach ($line in Get-Content -LiteralPath $envPath) {
    if ($line -match '^CLOUDREVE_DB_NAME=(.+)$') { $dbName = $Matches[1].Trim().Trim('"', "'") }
}
$composeArgs = @('compose', '--project-directory', $deployment, '--env-file', $envPath, '-f', $composePath)
$stopped = $false
try {
    # Stop only Cloudreve while taking a consistent DB + physical-files backup.
    Invoke-Docker @('stop', '--time', '60', 'cloudreve')
    $stopped = $true
    $dumpName = "cloudreve-receive-$($commit.Substring(0, 12)).dump"
    Invoke-Docker @('exec', $DatabaseContainer, 'sh', '-c', 'pg_dump --username "${POSTGRES_USER:-postgres}" --format custom --file "/tmp/$2" --dbname "$1"', 'sh', $dbName, $dumpName)
    Invoke-Docker @('cp', "${DatabaseContainer}:/tmp/$dumpName", (Join-Path $backup 'cloudreve.dump'))
    Invoke-Docker @('exec', $DatabaseContainer, 'rm', '-f', "/tmp/$dumpName")
    Invoke-Docker @('run', '--rm', '--pull', 'never', '--network', 'none', '--entrypoint', 'sh',
        '--mount', 'type=volume,source=cloudreve_data,target=/source,readonly',
        '--mount', "type=bind,source=$backup,target=/backup", $previousImage,
        '-c', 'tar -czf /backup/cloudreve-data.tar.gz -C /source .')
    @{
        commit = $commit; image = $image; previousImage = $previousImage
        hadOverride = $hadOverride; created = (Get-Date).ToString('o')
    } | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $backup 'deployment.json') -Encoding utf8NoBOM
    @"
services:
  cloudreve:
    image: $image
    pull_policy: never
    build:
      context: '$gitContext'
      dockerfile: Dockerfile.receive
      args:
        SOURCE_COMMIT: '$commit'
"@ | Set-Content -LiteralPath $overridePath -Encoding utf8NoBOM
    Invoke-Docker @($composeArgs + @('-f', $overridePath, 'up', '-d', '--no-deps', '--wait', 'cloudreve'))
    $stopped = $false
    Write-Host "Deployed commit: $commit"
    Write-Host "Backup: $backup"
} catch {
    $failure = $_
    if ($stopped) {
        if ($hadOverride) {
            Copy-Item -LiteralPath (Join-Path $backup 'compose.custom.yaml') -Destination $overridePath -Force
        } elseif (Test-Path -LiteralPath $overridePath) {
            Remove-Item -LiteralPath $overridePath
        }
        $restoreArgs = $composeArgs
        if ($hadOverride) { $restoreArgs += @('-f', $overridePath) }
        Invoke-Docker @($restoreArgs + @('up', '-d', '--no-deps', '--wait', 'cloudreve'))
    }
    throw $failure
}
