#requires -Version 5.0
# Smoke test for notifications: verify auto-trigger on material request workflow.
# Expected flow:
#   1) Staff creates request → admin + supervisor receive a notification.
#   2) Supervisor approves → original requester (staff) receives a notification.

$ErrorActionPreference = 'Stop'
$base = 'http://localhost:8080/api'

function Login {
    param([string]$Email)
    $tmp = "$env:TEMP\login.json"
    @{ email = $Email; password = 'admin123' } | ConvertTo-Json -Compress | Out-File $tmp -Encoding ascii -NoNewline
    $r = curl.exe -s -X POST "$base/auth/login" -H "Content-Type: application/json" --data-binary "@$tmp"
    Remove-Item $tmp
    return ($r | ConvertFrom-Json).token
}

function PostJson { param([string]$Path, [string]$Token, [object]$Body)
    $tmp = "$env:TEMP\req_$([Guid]::NewGuid()).json"
    if ($Body) { ($Body | ConvertTo-Json -Depth 8 -Compress) | Out-File $tmp -Encoding ascii -NoNewline }
    else { '{}' | Out-File $tmp -Encoding ascii -NoNewline }
    $r = curl.exe -s -X POST "$base$Path" -H "Authorization: Bearer $Token" -H "Content-Type: application/json" --data-binary "@$tmp"
    Remove-Item $tmp
    return $r
}
function GetJson { param([string]$Path, [string]$Token) return curl.exe -s "$base$Path" -H "Authorization: Bearer $Token" }

# Tokens
$staffToken = Login 'staff@shipyard.co.id'
$supToken   = Login 'supervisor@shipyard.co.id'
$adminToken = Login 'admin@shipyard.co.id'
Write-Host '[OK] All three roles logged in' -ForegroundColor Green

# Snapshot supervisor + admin unread counts before creating a request.
$supBefore   = ((GetJson '/notifications/stats' $supToken)   | ConvertFrom-Json).unread
$adminBefore = ((GetJson '/notifications/stats' $adminToken) | ConvertFrom-Json).unread
$staffBefore = ((GetJson '/notifications/stats' $staffToken) | ConvertFrom-Json).unread
Write-Host "[INFO] Unread before: staff=$staffBefore, supervisor=$supBefore, admin=$adminBefore"

# === STAFF CREATES MATERIAL REQUEST ===
$proj = ((GetJson '/projects' $staffToken) | ConvertFrom-Json).data | Where-Object { $_.code -eq 'H-2026-001' } | Select-Object -First 1
$mat  = ((GetJson '/materials' $staffToken) | ConvertFrom-Json).data | Select-Object -First 1
$reqBody = @{
    type = 'Material Request'
    projectId = $proj.id
    priority = 'high'
    reason = 'Untuk test notifikasi otomatis'
    items = @(@{ materialId = $mat.id; qty = 5 })
}
$created = PostJson '/material-requests' $staffToken $reqBody
$createdObj = $created | ConvertFrom-Json
Write-Host "[OK] Request created: $($createdObj.requestNo)" -ForegroundColor Green

Start-Sleep -Milliseconds 800  # give async best-effort writes time to land

$supAfter   = ((GetJson '/notifications/stats' $supToken)   | ConvertFrom-Json).unread
$adminAfter = ((GetJson '/notifications/stats' $adminToken) | ConvertFrom-Json).unread
if ($supAfter -le $supBefore)     { throw "Supervisor should have received a notification (was $supBefore, now $supAfter)" }
if ($adminAfter -le $adminBefore) { throw "Admin should have received a notification (was $adminBefore, now $adminAfter)" }
Write-Host "[OK] Supervisor unread: $supBefore to $supAfter" -ForegroundColor Green
Write-Host "[OK] Admin unread:      $adminBefore to $adminAfter" -ForegroundColor Green

# Show latest supervisor notification
$supNotif = ((GetJson '/notifications?limit=1' $supToken) | ConvertFrom-Json).data
if ($supNotif -and $supNotif.Count -gt 0) {
    Write-Host "[INFO] Supervisor latest: '$($supNotif[0].title)' - $($supNotif[0].message)" -ForegroundColor Cyan
}

# === SUPERVISOR APPROVES ===
$null = PostJson "/material-requests/$($createdObj.id)/approve" $supToken @{ notes = 'OK proceed' }
Start-Sleep -Milliseconds 800

$staffAfter = ((GetJson '/notifications/stats' $staffToken) | ConvertFrom-Json).unread
if ($staffAfter -le $staffBefore) { throw "Staff (requester) should have received a notification (was $staffBefore, now $staffAfter)" }
Write-Host "[OK] Staff unread: $staffBefore to $staffAfter (after approval)" -ForegroundColor Green

$staffNotif = ((GetJson '/notifications?limit=1' $staffToken) | ConvertFrom-Json).data
if ($staffNotif -and $staffNotif.Count -gt 0) {
    Write-Host "[INFO] Staff latest: '$($staffNotif[0].title)' - $($staffNotif[0].message)" -ForegroundColor Cyan
}

# === MARK ALL READ ===
$markedTmp = "$env:TEMP\dummy.json"; '{}' | Out-File $markedTmp -Encoding ascii -NoNewline
$markedResp = curl.exe -s -X PATCH "$base/notifications/read-all" -H "Authorization: Bearer $staffToken" -H "Content-Type: application/json" --data-binary "@$markedTmp"
Remove-Item $markedTmp
$marked = ($markedResp | ConvertFrom-Json).updated
Write-Host "[OK] Staff marked $marked notifications as read" -ForegroundColor Green

$staffFinal = ((GetJson '/notifications/stats' $staffToken) | ConvertFrom-Json).unread
if ($staffFinal -ne 0) { throw "Expected staff unread = 0 after mark-all-read, got $staffFinal" }
Write-Host "[OK] Staff unread is now 0" -ForegroundColor Green

Write-Host ""
Write-Host "All notification tests PASSED" -ForegroundColor Green
