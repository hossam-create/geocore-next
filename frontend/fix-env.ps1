$enc = New-Object System.Text.UTF8Encoding $false
$appDir = "E:\New computer\Development Coding\Projects\(Current Repos)\geocore-next\frontend\app"
$dirs = @($appDir,
    "E:\New computer\Development Coding\Projects\(Current Repos)\geocore-next\frontend\components",
    "E:\New computer\Development Coding\Projects\(Current Repos)\geocore-next\frontend\lib",
    "E:\New computer\Development Coding\Projects\(Current Repos)\geocore-next\frontend\store"
)

foreach ($dir in $dirs) {
    Get-ChildItem $dir -Recurse -Include "*.tsx","*.ts" | ForEach-Object {
        $path = $_.FullName
        $content = [System.IO.File]::ReadAllText($path)
        if ($content -match "import\.meta\.env") {
            $content = $content -replace "import\.meta\.env\.VITE_", "process.env.NEXT_PUBLIC_"
            $content = $content -replace "import\.meta\.env\.", "process.env.NEXT_PUBLIC_"
            [System.IO.File]::WriteAllText($path, $content, $enc)
            Write-Host "Fixed: $($_.Name)"
        }
    }
}
Write-Host "Done."
