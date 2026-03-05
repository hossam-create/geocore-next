# GeoCore Next - Push to GitHub

$GITHUB_USERNAME = "hossam-create"
$REPO_NAME = "geocore-next"
$REPO_URL = "https://github.com/$GITHUB_USERNAME/$REPO_NAME.git"

Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  GeoCore Next - GitHub Push Script" -ForegroundColor Cyan
Write-Host "  Repo: $REPO_URL" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan

if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Write-Host "ERROR: git not found. Install from https://git-scm.com" -ForegroundColor Red
    exit 1
}

if (-not (Test-Path ".git")) {
    Write-Host "Running git init..." -ForegroundColor Yellow
    git init
}

git add .

$DATE = Get-Date -Format "yyyy-MM-dd HH:mm"
git commit -m "docs: add PRD, CLAUDE, TASKS - full gap analysis and 98-task roadmap [$DATE]"
Write-Host "Committed." -ForegroundColor Green

$remoteExists = git remote | Where-Object { $_ -eq "origin" }
if ($remoteExists) {
    git remote set-url origin $REPO_URL
}
else {
    git remote add origin $REPO_URL
}
Write-Host "Remote set: $REPO_URL" -ForegroundColor Cyan

Write-Host "Pushing to GitHub..." -ForegroundColor Yellow
git branch -M main
git push -u origin main --force

if ($LASTEXITCODE -eq 0) {
    Write-Host "============================================" -ForegroundColor Green
    Write-Host "  SUCCESS! Code is now on GitHub" -ForegroundColor Green
    Write-Host "  https://github.com/$GITHUB_USERNAME/$REPO_NAME" -ForegroundColor Green
    Write-Host "============================================" -ForegroundColor Green
}
else {
    Write-Host "PUSH FAILED. Make sure:" -ForegroundColor Red
    Write-Host "  1. Repo exists at https://github.com/new  (name: $REPO_NAME)" -ForegroundColor Yellow
    Write-Host "  2. Use a Personal Access Token as password (not your GitHub password)" -ForegroundColor Yellow
    Write-Host "     GitHub -> Settings -> Developer Settings -> Personal Access Tokens" -ForegroundColor Yellow
}
