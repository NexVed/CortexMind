Write-Host "Setting up CORTEX dependencies..." -ForegroundColor Cyan

Write-Host "`n[1/2] Setting up Go Backend..." -ForegroundColor Yellow
go mod tidy
if ($LASTEXITCODE -ne 0) {
    Write-Host "Warning: 'go mod tidy' encountered an error. Ensure Go is installed." -ForegroundColor Red
}

Write-Host "`n[2/2] Setting up SolidJS UI..." -ForegroundColor Yellow
cd ui
npm install
if ($LASTEXITCODE -ne 0) {
    Write-Host "Warning: 'npm install' encountered an error. Ensure Node.js is installed." -ForegroundColor Red
}
cd ..

Write-Host "`nSetup complete! `n" -ForegroundColor Green
Write-Host "To run the backend daemon:"
Write-Host "  go run ./cmd/cortexd`n"
Write-Host "To run the frontend UI:"
Write-Host "  cd ui"
Write-Host "  npm run dev`n"
