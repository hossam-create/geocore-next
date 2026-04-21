$srcBase = "E:\New computer\Development Coding\Projects\(Current Repos)\geocore-next\frontend\artifacts\web\src"
$destBase = "E:\New computer\Development Coding\Projects\(Current Repos)\geocore-next\frontend"
$enc = New-Object System.Text.UTF8Encoding $false

function Transform-WouterToNext {
    param([string]$text, [bool]$addUseClient)

    if ($addUseClient -and -not ($text -match "^'use client'")) {
        $text = "'use client'`n" + $text
    }

    if ($text -notmatch "from ['""]wouter['""]") { return $text }

    $hasLink       = $text -match '\bLink\b'
    $hasUseRouter  = $text -match '\buseLocation\b'
    $hasUseSearch  = $text -match '\buseSearch\b'
    $hasUseParams  = $text -match '\buseParams\b'

    # Remove the entire wouter import line
    $text = $text -replace "(?m)^import\s+\{[^}]+\}\s+from\s+'wouter';?\s*$\r?\n?", ""
    $text = $text -replace '(?m)^import\s+\{[^}]+\}\s+from\s+"wouter";?\s*$\r?\n?', ""

    # Build next imports
    $ni = ""
    if ($hasLink)      { $ni += "import Link from 'next/link';`n" }
    $nav = @()
    if ($hasUseRouter) { $nav += "useRouter" }
    if ($hasUseSearch) { $nav += "useSearchParams" }
    if ($hasUseParams) { $nav += "useParams" }
    if ($nav.Count -gt 0) { $ni += "import { $($nav -join ', ') } from 'next/navigation';`n" }

    # Inject after 'use client' line (or at very top)
    if ($text -match "^'use client'") {
        $text = $text -replace "(?m)^('use client')\r?\n", "`$1`n$ni"
    } else {
        $text = $ni + $text
    }

    # --- Hook replacements ---

    # useSearch() → useSearchParams()
    $text = $text -replace '\buseSearch\(\)', 'useSearchParams()'

    # const [, navigate] = useLocation() variants
    $text = $text -replace 'const \[,\s*navigate\]\s*=\s*useLocation\(\);?', 'const router = useRouter();'
    $text = $text -replace 'const \[location,\s*navigate\]\s*=\s*useLocation\(\);?', "const router = useRouter();`n  const location = typeof window !== 'undefined' ? window.location.pathname : '';"

    # navigate(X, { replace: true }) → router.replace(X)
    $text = $text -replace 'navigate\(([^,)]+),\s*\{\s*replace\s*:\s*true\s*\}\)', 'router.replace($1)'

    # navigate( → router.push(
    $text = $text -replace '\bnavigate\(', 'router.push('

    # Generic useParams<{...}>() → (useParams() as { ... })
    $text = $text -replace 'useParams<\{([^}]+)\}>\(\)', '(useParams() as { $1 })'

    return $text
}

function Copy-Dir {
    param([string]$src, [string]$dest, [bool]$addUseClient = $false, [bool]$transformWouter = $true)
    if (-not (Test-Path $src)) { Write-Host "SKIP missing: $src"; return }
    New-Item -ItemType Directory -Force -Path $dest | Out-Null
    Get-ChildItem $src -Recurse -File | ForEach-Object {
        $rel  = $_.FullName.Substring($src.Length).TrimStart('\','/')
        $out  = Join-Path $dest $rel
        $dir  = Split-Path $out -Parent
        New-Item -ItemType Directory -Force -Path $dir | Out-Null

        if ($_.Extension -in '.ts','.tsx') {
            $content = [System.IO.File]::ReadAllText($_.FullName, [System.Text.Encoding]::UTF8)
            if ($transformWouter) {
                $content = Transform-WouterToNext -text $content -addUseClient $addUseClient
            }
            [System.IO.File]::WriteAllText($out, $content, $enc)
        } else {
            Copy-Item $_.FullName $out -Force
        }
        Write-Host "  OK $rel"
    }
}

Write-Host "`n=== Copying lib/ ==="
Copy-Dir "$srcBase\lib" "$destBase\lib" -addUseClient $false -transformWouter $false

Write-Host "`n=== Copying store/ (add 'use client') ==="
Copy-Dir "$srcBase\store" "$destBase\store" -addUseClient $true -transformWouter $false

Write-Host "`n=== Copying hooks/ ==="
Copy-Dir "$srcBase\hooks" "$destBase\hooks" -addUseClient $false -transformWouter $false

Write-Host "`n=== Copying types/ ==="
Copy-Dir "$srcBase\types" "$destBase\types" -addUseClient $false -transformWouter $false

Write-Host "`n=== Copying components/ (transform wouter, add 'use client') ==="
Copy-Dir "$srcBase\components" "$destBase\components" -addUseClient $true -transformWouter $true

# ---- Pages ----------------------------------------------------------------

$pageMap = [ordered]@{
    "HomePage.tsx"           = "app/page.tsx"
    "ListingsPage.tsx"       = "app/listings/page.tsx"
    "ListingDetailPage.tsx"  = "app/listings/[id]/page.tsx"
    "AuctionsPage.tsx"       = "app/auctions/page.tsx"
    "AuctionDetailPage.tsx"  = "app/auctions/[id]/page.tsx"
    "CheckoutPage.tsx"       = "app/checkout/page.tsx"
    "SellPage.tsx"           = "app/sell/page.tsx"
    "LoginPage.tsx"          = "app/login/page.tsx"
    "RegisterPage.tsx"       = "app/register/page.tsx"
    "ProfilePage.tsx"        = "app/profile/page.tsx"
    "WalletPage.tsx"         = "app/wallet/page.tsx"
    "MyStorefrontPage.tsx"   = "app/my-store/page.tsx"
    "StoreListPage.tsx"      = "app/stores/page.tsx"
    "BrandOutletPage.tsx"    = "app/brand-outlet/page.tsx"
    "StorefrontPage.tsx"     = "app/stores/[slug]/page.tsx"
    "SellerPage.tsx"         = "app/sellers/[id]/page.tsx"
    "DashboardPage.tsx"      = "app/dashboard/page.tsx"
    "SearchPage.tsx"         = "app/search/page.tsx"
    "AdvancedSearchPage.tsx" = "app/advanced-search/page.tsx"
}

Write-Host "`n=== Migrating pages ==="
foreach ($entry in $pageMap.GetEnumerator()) {
    $srcFile  = Join-Path "$srcBase\pages" $entry.Key
    $destFile = Join-Path $destBase $entry.Value

    if (-not (Test-Path $srcFile)) {
        Write-Host "  SKIP (missing): $($entry.Key)"
        continue
    }

    $dir = Split-Path $destFile -Parent
    New-Item -ItemType Directory -Force -Path $dir | Out-Null

    $content = [System.IO.File]::ReadAllText($srcFile, [System.Text.Encoding]::UTF8)
    $content = Transform-WouterToNext -text $content -addUseClient $true
    [System.IO.File]::WriteAllText($destFile, $content, $enc)
    Write-Host "  OK $($entry.Key) -> $($entry.Value)"
}

Write-Host "`nDone."
