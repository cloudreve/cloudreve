#Requires -Version 7.0
# Run only against a fresh disposable instance bound to this test port.
[CmdletBinding()]
param([string]$BaseUrl = 'http://127.0.0.1:15212')
$ErrorActionPreference = 'Stop'
if ($BaseUrl -ne 'http://127.0.0.1:15212') { throw 'This test may only target the disposable instance at http://127.0.0.1:15212.' }
$api = "$BaseUrl/api/v4"
function Invoke-TestApi {
    param([string]$Path, [string]$Method = 'Get', [object]$Body = $null, [hashtable]$Headers = @{}, [switch]$ExpectFailure)
    $request = @{ Uri = "$api$Path"; Method = $Method; Headers = $Headers; ContentType = 'application/json' }
    if ($null -ne $Body) { $request.Body = $Body | ConvertTo-Json -Depth 12 -Compress }
    $response = Invoke-RestMethod @request
    if ($ExpectFailure) {
        if ($response.code -eq 0) { throw "Expected access or validation denial: $Path" }
    } elseif ($response.code -ne 0) {
        throw "API failed at ${Path}: $($response.code) $($response.msg)"
    }
    return $response
}
$suffix = [guid]::NewGuid().ToString('N')
$testPassword = [guid]::NewGuid().ToString('N')
$adminEmail = "receive-admin-$suffix@example.com"
$crawlerEmail = "receive-crawler-$suffix@example.com"
Invoke-TestApi '/user' Post @{ email = $adminEmail; password = $testPassword } | Out-Null
Invoke-TestApi '/user' Post @{ email = $crawlerEmail; password = $testPassword } | Out-Null
$adminLogin = Invoke-TestApi '/session/token' Post @{ email = $adminEmail; password = $testPassword }
$crawlerLogin = Invoke-TestApi '/session/token' Post @{ email = $crawlerEmail; password = $testPassword }
$adminHeaders = @{ Authorization = "Bearer $($adminLogin.data.token.access_token)" }
$crawlerHeaders = @{ Authorization = "Bearer $($crawlerLogin.data.token.access_token)" }
$sourceUri = "cloudreve://my/receive-$suffix.txt"
$folderUri = "cloudreve://my/inbox-$suffix"
$contents = [Text.Encoding]::UTF8.GetBytes("receive-once-$suffix")
$source = Invoke-RestMethod -Uri "$api/file/content?uri=$([uri]::EscapeDataString($sourceUri))" -Method Put -Headers $crawlerHeaders -ContentType 'application/octet-stream' -Body $contents
if ($source.code -ne 0) { throw "Test upload failed: $($source.msg)" }
Invoke-TestApi '/file/create' Post @{ uri = $folderUri; type = 'folder' } $adminHeaders | Out-Null
$request = @{ source_id = $source.data.id; dst = $folderUri }
Invoke-TestApi '/admin/file/receive' Post $request $crawlerHeaders -ExpectFailure | Out-Null
Invoke-TestApi '/admin/file/receive' Post $request -ExpectFailure | Out-Null
$received = Invoke-TestApi '/admin/file/receive' Post $request $adminHeaders
if ($received.data.id -eq $source.data.id -or -not $received.data.owned) { throw 'Received file must have an independent administrator-owned ID.' }
if ($received.data.primary_entity -ne $source.data.primary_entity) { throw 'Receive unexpectedly created another physical blob.' }
Invoke-TestApi '/admin/file/receive' Post $request $adminHeaders -ExpectFailure | Out-Null
Invoke-TestApi '/file' Delete @{ uris = @($sourceUri); skip_soft_delete = $true } $crawlerHeaders | Out-Null
$info = Invoke-TestApi "/file/info?id=$($received.data.id)" Get $null $adminHeaders
if ($info.data.id -ne $received.data.id) { throw 'Received file disappeared after source deletion.' }
$download = Invoke-TestApi '/file/url' Post @{ uris = @($received.data.path); download = $true } $adminHeaders
$downloadUri = [uri]$download.data.urls[0].url
if ($downloadUri.Authority -ne ([uri]$BaseUrl).Authority) { throw 'Download URL points outside the disposable test instance.' }
$downloadPath = Join-Path ([IO.Path]::GetTempPath()) "cloudreve-receive-$suffix.bin"
try {
    Invoke-WebRequest -Uri $downloadUri -OutFile $downloadPath
    $actual = [IO.File]::ReadAllBytes($downloadPath)
    if ([Convert]::ToBase64String($actual) -ne [Convert]::ToBase64String($contents)) { throw 'Downloaded contents changed after source deletion.' }
} finally {
    if (Test-Path -LiteralPath $downloadPath) { Remove-Item -LiteralPath $downloadPath }
}
[pscustomobject]@{
    result = 'PASS'; source_id = $source.data.id; received_id = $received.data.id
    shared_blob = $received.data.primary_entity; source_permanently_deleted = $true
    administrator_download_verified = $true; anonymous_and_normal_user_denied = $true
    duplicate_name_denied = $true
}
