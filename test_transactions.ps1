#requires -Version 5.0
# Quick smoke test for the transactions module.
# Asserts atomic stock updates work correctly across receipt/issue/scrap.

$ErrorActionPreference = 'Stop'
$base = 'http://localhost:8080/api'

function Invoke-Json {
    param(
        [string]$Method,
        [string]$Path,
        [string]$Token,
        [object]$Body
    )
    $headers = @{ Authorization = "Bearer $Token" }
    if ($Body) {
        $tmp = "$env:TEMP\req_$([Guid]::NewGuid()).json"
        ($Body | ConvertTo-Json -Depth 8 -Compress) | Out-File -FilePath $tmp -Encoding ascii -NoNewline
        $resp = curl.exe -s -X $Method "$base$Path" `
            -H "Authorization: Bearer $Token" `
            -H "Content-Type: application/json" `
            --data-binary "@$tmp"
        Remove-Item $tmp -ErrorAction SilentlyContinue
    } else {
        $resp = curl.exe -s -X $Method "$base$Path" -H "Authorization: Bearer $Token"
    }
    return $resp
}

# 1. Login
$loginBody = @{ email = 'admin@shipyard.co.id'; password = 'admin123' }
$loginResp = Invoke-Json -Method POST -Path '/auth/login' -Body $loginBody
$token = ($loginResp | ConvertFrom-Json).token
Write-Host '[OK] Logged in' -ForegroundColor Green

# 2. Snapshot a material's current stock (BLT-HEX-M20)
$matsResp = Invoke-Json -Method GET -Path '/materials' -Token $token
$mats = ($matsResp | ConvertFrom-Json).data
$bolt = $mats | Where-Object { $_.sku -eq 'BLT-HEX-M20' } | Select-Object -First 1
$boltStockBefore = [int]$bolt.stock
Write-Host "[INFO] BLT-HEX-M20 stock BEFORE: $boltStockBefore"

# 3. Snapshot a project + vendor
$projResp = Invoke-Json -Method GET -Path '/projects' -Token $token
$proj = (($projResp | ConvertFrom-Json).data | Where-Object { $_.code -eq 'H-2026-001' } | Select-Object -First 1)
$venResp = Invoke-Json -Method GET -Path '/vendors' -Token $token
$ven = (($venResp | ConvertFrom-Json).data | Select-Object -First 1)

# === RECEIPT (vendor only, no PO) ===
$receiptBody = @{
    vendorId = $ven.id
    items = @(
        @{ materialId = $bolt.id; qty = 100; heatNumber = 'HN-TEST-001' }
    )
}
$rcpResp = Invoke-Json -Method POST -Path '/goods-receipt' -Token $token -Body $receiptBody
$rcp = $rcpResp | ConvertFrom-Json
if ($rcp.total -ne 1) { throw "Expected 1 receipt, got $($rcp.total)" }
Write-Host "[OK] Receipt created: $($rcp.data[0].transactionNo) (qty=$($rcp.data[0].qty))" -ForegroundColor Green

# Verify stock increased
$bolt2 = ((Invoke-Json -Method GET -Path '/materials' -Token $token) | ConvertFrom-Json).data | Where-Object { $_.sku -eq 'BLT-HEX-M20' } | Select-Object -First 1
$expected = $boltStockBefore + 100
if ([int]$bolt2.stock -ne $expected) { throw "Stock mismatch after receipt: expected $expected, got $($bolt2.stock)" }
Write-Host "[OK] Stock after receipt = $($bolt2.stock) (expected $expected)" -ForegroundColor Green

# === ISSUE ===
$issueBody = @{
    projectId = $proj.id
    mandor = 'Test Mandor'
    items = @(
        @{ materialId = $bolt.id; qty = 30 }
    )
}
$issResp = Invoke-Json -Method POST -Path '/goods-issue' -Token $token -Body $issueBody
$iss = $issResp | ConvertFrom-Json
if ($iss.total -ne 1) { throw "Expected 1 issue, got $($iss.total)" }
Write-Host "[OK] Issue created: $($iss.data[0].transactionNo) (qty=$($iss.data[0].qty), notes=$($iss.data[0].notes))" -ForegroundColor Green

$bolt3 = ((Invoke-Json -Method GET -Path '/materials' -Token $token) | ConvertFrom-Json).data | Where-Object { $_.sku -eq 'BLT-HEX-M20' } | Select-Object -First 1
$expected = $boltStockBefore + 100 - 30
if ([int]$bolt3.stock -ne $expected) { throw "Stock mismatch after issue: expected $expected, got $($bolt3.stock)" }
Write-Host "[OK] Stock after issue = $($bolt3.stock) (expected $expected)" -ForegroundColor Green

# === ISSUE BEYOND STOCK (should fail with 422) ===
$badBody = @{
    projectId = $proj.id
    items = @(
        @{ materialId = $bolt.id; qty = 999999 }
    )
}
$tmp = "$env:TEMP\bad.json"
($badBody | ConvertTo-Json -Depth 8 -Compress) | Out-File -FilePath $tmp -Encoding ascii -NoNewline
$badResp = curl.exe -s -o NUL -w '%{http_code}' -X POST "$base/goods-issue" -H "Authorization: Bearer $token" -H "Content-Type: application/json" --data-binary "@$tmp"
Remove-Item $tmp
if ($badResp -ne '422') { throw "Expected 422 on insufficient stock, got $badResp" }
Write-Host "[OK] Insufficient stock rejected with 422" -ForegroundColor Green

# === SCRAP ===
$scrapBody = @{
    type = 'scrap'
    materialId = $bolt.id
    projectId = $proj.id
    qty = 5
    reason = 'Damaged during installation'
}
$scrap = (Invoke-Json -Method POST -Path '/scrap-return' -Token $token -Body $scrapBody) | ConvertFrom-Json
Write-Host "[OK] Scrap created: $($scrap.transactionNo)" -ForegroundColor Green

$bolt4 = ((Invoke-Json -Method GET -Path '/materials' -Token $token) | ConvertFrom-Json).data | Where-Object { $_.sku -eq 'BLT-HEX-M20' } | Select-Object -First 1
$expected = $boltStockBefore + 100 - 30 - 5
if ([int]$bolt4.stock -ne $expected) { throw "Stock mismatch after scrap: expected $expected, got $($bolt4.stock)" }
Write-Host "[OK] Stock after scrap = $($bolt4.stock) (expected $expected)" -ForegroundColor Green

# === RETURN (reusable) ===
$returnBody = @{
    type = 'return'
    materialId = $bolt.id
    projectId = $proj.id
    qty = 3
    reason = 'Excess from project'
}
$ret = (Invoke-Json -Method POST -Path '/scrap-return' -Token $token -Body $returnBody) | ConvertFrom-Json
Write-Host "[OK] Return created: $($ret.transactionNo)" -ForegroundColor Green

$bolt5 = ((Invoke-Json -Method GET -Path '/materials' -Token $token) | ConvertFrom-Json).data | Where-Object { $_.sku -eq 'BLT-HEX-M20' } | Select-Object -First 1
$expected = $boltStockBefore + 100 - 30 - 5 + 3
if ([int]$bolt5.stock -ne $expected) { throw "Stock mismatch after return: expected $expected, got $($bolt5.stock)" }
Write-Host "[OK] Stock after return = $($bolt5.stock) (expected $expected)" -ForegroundColor Green

# === LIST TRANSACTIONS ===
$txList = ((Invoke-Json -Method GET -Path '/transactions?limit=10' -Token $token) | ConvertFrom-Json)
Write-Host ""
Write-Host "Recent transactions ($($txList.total)):"
$txList.data | Select-Object -First 10 | ForEach-Object {
    $line = "  [$($_.type)] $($_.transactionNo) | $($_.sku) qty=$($_.qty) by $($_.user)"
    if ($_.project) { $line += " -> $($_.project)" }
    if ($_.vendor)  { $line += " <- $($_.vendor)" }
    Write-Host $line
}

Write-Host ""
Write-Host "All atomic transaction tests PASSED" -ForegroundColor Green
