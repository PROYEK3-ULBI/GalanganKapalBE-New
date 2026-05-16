#requires -Version 5.0
# Smoke test for the material requests approval workflow.

$ErrorActionPreference = 'Stop'
$base = 'http://localhost:8080/api'

function Login {
    param([string]$Email, [string]$Pass)
    $tmp = "$env:TEMP\login.json"
    @{ email = $Email; password = $Pass } | ConvertTo-Json -Compress | Out-File $tmp -Encoding ascii -NoNewline
    $resp = curl.exe -s -X POST "$base/auth/login" -H "Content-Type: application/json" --data-binary "@$tmp"
    Remove-Item $tmp
    return ($resp | ConvertFrom-Json).token
}

function PostJson {
    param([string]$Path, [string]$Token, [object]$Body)
    $tmp = "$env:TEMP\req_$([Guid]::NewGuid()).json"
    if ($Body) {
        ($Body | ConvertTo-Json -Depth 8 -Compress) | Out-File $tmp -Encoding ascii -NoNewline
        $resp = curl.exe -s -X POST "$base$Path" -H "Authorization: Bearer $Token" -H "Content-Type: application/json" --data-binary "@$tmp"
        Remove-Item $tmp
    } else {
        $resp = curl.exe -s -X POST "$base$Path" -H "Authorization: Bearer $Token" -H "Content-Type: application/json"
    }
    return $resp
}

function GetJson { param([string]$Path, [string]$Token) return curl.exe -s "$base$Path" -H "Authorization: Bearer $Token" }

# === STAFF SUBMITS REQUEST ===
$staffToken = Login -Email 'staff@shipyard.co.id' -Pass 'admin123'
Write-Host '[OK] Staff logged in' -ForegroundColor Green

# Get a project + materials to reference
$proj = ((GetJson '/projects' $staffToken) | ConvertFrom-Json).data | Where-Object { $_.code -eq 'H-2026-001' } | Select-Object -First 1
$mats = ((GetJson '/materials' $staffToken) | ConvertFrom-Json).data
$pickedMats = $mats | Select-Object -First 2

$createBody = @{
    type = 'Material Request'
    projectId = $proj.id
    priority = 'high'
    reason = 'Untuk tahap pemasangan lambung Hull 001 - test E2E'
    items = @(
        @{ materialId = $pickedMats[0].id; qty = 5 },
        @{ materialId = $pickedMats[1].id; qty = 10 }
    )
}
$created = PostJson '/material-requests' $staffToken $createBody
$createdObj = $created | ConvertFrom-Json
Write-Host "[OK] Request created: $($createdObj.requestNo) (status=$($createdObj.status))" -ForegroundColor Green
Write-Host "  Items: $($createdObj.items.Count) | Priority: $($createdObj.priority) | Project: $($createdObj.project)"

$reqId = $createdObj.id

# === STAFF SEES OWN REQUEST ===
$myList = ((GetJson '/material-requests' $staffToken) | ConvertFrom-Json)
Write-Host "[OK] Staff sees $($myList.total) own request(s)"

# === STAFF CANNOT APPROVE (should 403) ===
$http = curl.exe -s -o NUL -w "%{http_code}" -X POST "$base/material-requests/$reqId/approve" -H "Authorization: Bearer $staffToken" -H "Content-Type: application/json"
if ($http -ne '403') { throw "Expected 403 when staff approves, got $http" }
Write-Host "[OK] Staff approve attempt rejected with 403" -ForegroundColor Green

# === SUPERVISOR LOGS IN ===
$supToken = Login -Email 'supervisor@shipyard.co.id' -Pass 'admin123'
Write-Host '[OK] Supervisor logged in' -ForegroundColor Green

# === SUPERVISOR SEES ALL PENDING ===
$pending = ((GetJson '/material-requests?status=pending' $supToken) | ConvertFrom-Json)
Write-Host "[OK] Supervisor sees $($pending.total) pending request(s)"

# === SUPERVISOR APPROVES ===
$approveBody = @{ notes = 'Approved untuk tahap urgent' }
$approved = PostJson "/material-requests/$reqId/approve" $supToken $approveBody
$approvedObj = $approved | ConvertFrom-Json
if ($approvedObj.status -ne 'approved') { throw "Expected approved, got $($approvedObj.status)" }
Write-Host "[OK] Approved by $($approvedObj.approvedBy)" -ForegroundColor Green

# === DOUBLE APPROVE FAILS (already decided) ===
$http = curl.exe -s -o NUL -w "%{http_code}" -X POST "$base/material-requests/$reqId/approve" -H "Authorization: Bearer $supToken" -H "Content-Type: application/json"
if ($http -ne '409') { throw "Expected 409 on double-approve, got $http" }
Write-Host "[OK] Double-approve rejected with 409" -ForegroundColor Green

# === STAFF CANNOT DELETE APPROVED ===
$http = curl.exe -s -o NUL -w "%{http_code}" -X DELETE "$base/material-requests/$reqId" -H "Authorization: Bearer $staffToken"
if ($http -ne '409') { throw "Expected 409 on delete-approved, got $http" }
Write-Host "[OK] Cannot delete approved request (409)" -ForegroundColor Green

# === CREATE ANOTHER REQUEST AND REJECT IT ===
$created2 = PostJson '/material-requests' $staffToken $createBody
$req2Id = ($created2 | ConvertFrom-Json).id
$rejectBody = @{ notes = 'Stok masih cukup di gudang, tidak perlu request' }
$rejected = PostJson "/material-requests/$req2Id/reject" $supToken $rejectBody
$rejObj = $rejected | ConvertFrom-Json
Write-Host "[OK] Rejected: $($rejObj.requestNo) by $($rejObj.approvedBy)" -ForegroundColor Green

# === FINAL: List all requests ===
Write-Host ""
$all = ((GetJson '/material-requests' $supToken) | ConvertFrom-Json)
Write-Host "All requests (visible to supervisor): $($all.total)"
$all.data | ForEach-Object {
    Write-Host "  $($_.requestNo) | $($_.status) | $($_.priority) | by $($_.requester) | $($_.items.Count) items"
}
Write-Host ""
Write-Host "All approval workflow tests PASSED" -ForegroundColor Green
